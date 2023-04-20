package cmds

import (
	"context"
	"embed"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/watcher"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	parka "github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"strings"
)

type ServeCommand struct {
	description         *cmds.CommandDescription
	dbConnectionFactory pkg.DBConnectionFactory
	repositories        []string
	commands            []cmds.Command
	aliases             []*alias.CommandAlias
}

func (s *ServeCommand) Description() *cmds.CommandDescription {
	return s.description
}

//go:embed templates
var embeddedFiles embed.FS

func (s *ServeCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
) error {
	// set up repository watcher
	r := repositories.NewRepository(
		repositories.WithDirectories(s.repositories),
		repositories.WithUpdateCallback(func(cmd cmds.Command) error {
			description := cmd.Description()
			log.Info().Str("name", description.Name).
				Str("source", description.Source).
				Msg("Updating cmd")
			// TODO(manuel, 2023-04-19) This is where we would recompute the HandlerFunc used below in GET and POST
			return nil
		}),
		repositories.WithRemoveCallback(func(cmd cmds.Command) error {
			description := cmd.Description()
			log.Info().Str("name", description.Name).
				Str("source", description.Source).
				Msg("Removing cmd")
			// TODO(manuel, 2023-04-19) This is where we would recompute the HandlerFunc used below in GET and POST
			return nil
		}),
	)

	for _, command := range s.commands {
		r.Add(command)
	}
	for _, alias := range s.aliases {
		r.Add(alias)
	}

	yamlLoader := &loaders.YAMLReaderCommandLoader{
		YAMLCommandLoader: &pkg.SqlCommandLoader{
			DBConnectionFactory: pkg.OpenDatabaseFromSqletonConnectionLayer,
		},
	}

	// now set up parka server
	port := ps["serve-port"].(int)
	host := ps["serve-host"].(string)

	serverOptions := []parka.ServerOption{
		parka.WithPort(uint16(port)),
		parka.WithAddress(host),
	}
	server, err := parka.NewServer(serverOptions...)
	if err != nil {
		return err
	}

	//glazedParameterLayers, err := cli.NewGlazedParameterLayers()
	if err != nil {
		return err
	}

	server.Router.StaticFileFS(
		"favicon.ico",
		"templates/favicon.ico",
		http.FS(embeddedFiles),
	)

	sqletonConnectionLayer := parsedLayers["sqleton-connection"]
	dbtConnectionLayer := parsedLayers["dbt"]

	// Here we serve the HTML view of the command
	server.Router.GET("/sqleton/*CommandPath", func(c *gin.Context) {
		commandPath := c.Param("CommandPath")
		commandPath = strings.TrimPrefix(commandPath, "/")
		path := strings.Split(commandPath, "/")
		commands := r.Root.CollectCommands(path, false)
		if len(commands) == 0 {
			c.JSON(404, gin.H{"error": "command not found"})
			return
		}

		if len(commands) > 1 {
			c.JSON(404, gin.H{"error": "ambiguous command"})
			return
		}

		sqlCommand, ok := commands[0].(cmds.GlazeCommand)
		if !ok || sqlCommand == nil {
			c.JSON(500, gin.H{"error": "command is not a sql command"})
		}

		// NOTE(2023-04-19, manuel): This would lookup a precomputed handlerFunc that is computed by the repository watcher
		// See note in WithUpdateCallback above.
		// let's make our own template lookup from a local directory, with blackjack and footers
		// TODO(manuel, 2023-04-19) This local loading from a directory reloadable is a dev mode kind of thing
		localTemplateLookup, err := render.LookupTemplateFromFSReloadable(os.DirFS("."), "cmd/sqleton/cmds/templates", "cmd/sqleton/cmds/templates/**.tmpl.html")
		if err != nil {
			c.JSON(500, gin.H{"error": "could not create template lookup"})
			return
		}

		dataTablesProcessorFunc := render.NewTemplateLookupCreateProcessorFunc(localTemplateLookup, "data-tables.tmpl.html")

		handleSimpleQueryCommand := server.HandleSimpleQueryCommand(
			sqlCommand,
			glazed.WithCreateProcessor(
				dataTablesProcessorFunc,
			),
			glazed.WithParserOptions(
				glazed.WithStaticLayer("sqleton-connection", sqletonConnectionLayer.Parameters),
				glazed.WithStaticLayer("dbt", dbtConnectionLayer.Parameters),
			),
		)

		handleSimpleQueryCommand(c)
	})

	server.Router.POST("/sqleton/*CommandPath", func(c *gin.Context) {
		commandPath := c.Param("CommandPath")
		commandPath = strings.TrimPrefix(commandPath, "/")
		path := strings.Split(commandPath, "/")
		commands := r.Root.CollectCommands(path, false)
		if len(commands) == 0 {
			c.JSON(404, gin.H{"error": "command not found"})
			return
		}

		if len(commands) > 1 {
			c.JSON(404, gin.H{"error": "ambiguous command"})
			return
		}

		sqlCommand, ok := commands[0].(*pkg.SqlCommand)
		if !ok || sqlCommand == nil {
			c.JSON(500, gin.H{"error": "command is not a sql command"})
		}
		handleSimpleFormCommand := server.HandleSimpleFormCommand(
			sqlCommand,
			glazed.WithParserOptions(
				glazed.WithStaticLayer("sqleton-connection", sqletonConnectionLayer.Parameters),
				glazed.WithStaticLayer("dbt", dbtConnectionLayer.Parameters),
				//glazed.WithGlazeOutputParserOption(glazedParameterLayers, "table", "html"),
			),
		)

		handleSimpleFormCommand(c)
	})

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		return r.Watch(ctx, yamlLoader, nil,
			watcher.WithMask("**/*.yaml"),
		)
	})
	errGroup.Go(func() error {
		return server.Run(ctx)
	})

	err = errGroup.Wait()
	if err != nil {
		return err
	}

	return nil
}

func NewServeCommand(
	dbConnectionFactory pkg.DBConnectionFactory,
	repositories []string, commands []cmds.Command, aliases []*alias.CommandAlias,
	options ...cmds.CommandDescriptionOption,
) *ServeCommand {
	options_ := append(options,
		cmds.WithShort("Serve the API"),
		cmds.WithArguments(),
		cmds.WithFlags(
			parameters.NewParameterDefinition(
				"serve-port",
				parameters.ParameterTypeInteger,
				parameters.WithShortFlag("p"),
				parameters.WithHelp("Port to serve the API on"),
				parameters.WithDefault(8080),
			),
			parameters.NewParameterDefinition(
				"serve-host",
				parameters.ParameterTypeString,
				parameters.WithHelp("Host to serve the API on"),
				parameters.WithDefault("localhost"),
			),
		),
	)
	return &ServeCommand{
		dbConnectionFactory: dbConnectionFactory,
		description: cmds.NewCommandDescription(
			"serve",
			options_...,
		),
		repositories: repositories,
		commands:     commands,
		aliases:      aliases,
	}
}
