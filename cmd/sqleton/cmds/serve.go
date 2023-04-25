package cmds

import (
	"context"
	"embed"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/watcher"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers"
	parka "github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"strings"
	"time"
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

	// server as JSON for datatables
	server.Router.GET("/data/*CommandPath", func(c *gin.Context) {
		commandPath := c.Param("CommandPath")
		commandPath = strings.TrimPrefix(commandPath, "/")
		sqlCommand, ok := getRepositoryCommand(c, r, commandPath)
		if !ok {
			c.JSON(404, gin.H{"error": "command not found"})
			return
		}

		jsonProcessorFunc := glazed.CreateJSONProcessor

		handle := server.HandleSimpleQueryCommand(sqlCommand,
			glazed.WithCreateProcessor(jsonProcessorFunc),
			glazed.WithParserOptions(
				glazed.WithStaticLayer("sqleton-connection", sqletonConnectionLayer.Parameters),
				glazed.WithStaticLayer("dbt", dbtConnectionLayer.Parameters),
			),
		)

		handle(c)
	})

	// Here we serve the HTML view of the command
	server.Router.GET("/sqleton/*CommandPath", func(c *gin.Context) {
		commandPath := c.Param("CommandPath")
		commandPath = strings.TrimPrefix(commandPath, "/")
		sqlCommand, ok := getRepositoryCommand(c, r, commandPath)
		if !ok {
			c.JSON(404, gin.H{"error": "command not found"})
			return
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

		type Link struct {
			Href  string
			Text  string
			Class string
		}

		name := sqlCommand.Description().Name
		dateTime := time.Now().Format("2006-01-02--15-04-05")
		links := []Link{
			{
				Href:  fmt.Sprintf("/download/%s/%s-%s.csv", commandPath, dateTime, name),
				Text:  "Download CSV",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("/download/%s/%s-%s.json", commandPath, dateTime, name),
				Text:  "Download JSON",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("/download/%s/%s-%s.xlsx", commandPath, dateTime, name),
				Text:  "Download Excel",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("/download/%s/%s-%s.md", commandPath, dateTime, name),
				Text:  "Download Markdown",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("/download/%s/%s-%s.html", commandPath, dateTime, name),
				Text:  "Download HTML",
				Class: "download",
			},
			{
				Href:  fmt.Sprintf("/download/%s/%s-%s.txt", commandPath, dateTime, name),
				Text:  "Download Text",
				Class: "download",
			},
		}

		dev, _ := ps["dev"].(bool)

		var dataTablesProcessorFunc glazed.CreateProcessorFunc
		if dev {
			dataTablesProcessorFunc = render.NewHTMLTemplateLookupCreateProcessorFunc(
				localTemplateLookup,
				"data-tables.tmpl.html",
				render.WithHTMLTemplateOutputFormatterData(
					map[string]interface{}{
						"Links": links,
					},
				),
			)
		} else {
			dataTablesProcessorFunc, err = render.NewDefaultCreateProcessorFunc(
				render.WithHTMLTemplateOutputFormatterData(
					map[string]interface{}{
						"Links": links,
					},
				),
			)
			if err != nil {
				c.JSON(500, gin.H{"error": "could not create default processor func"})
				return
			}
		}

		handle := server.HandleSimpleQueryCommand(
			sqlCommand,
			glazed.WithCreateProcessor(
				dataTablesProcessorFunc,
			),
			glazed.WithParserOptions(
				glazed.WithStaticLayer("sqleton-connection", sqletonConnectionLayer.Parameters),
				glazed.WithStaticLayer("dbt", dbtConnectionLayer.Parameters),
			),
		)

		handle(c)
	})

	server.Router.GET("/download/*CommandPath", func(c *gin.Context) {
		path := c.Param("CommandPath")
		// get file name at end of path
		index := strings.LastIndex(path, "/")
		if index == -1 {
			c.JSON(500, gin.H{"error": "could not find file name"})
			return
		}
		if index >= len(path)-1 {
			c.JSON(500, gin.H{"error": "could not find file name"})
			return
		}
		fileName := path[index+1:]

		commandPath := strings.TrimPrefix(path[:index], "/")
		sqlCommand, ok := getRepositoryCommand(c, r, commandPath)
		if !ok {
			c.JSON(404, gin.H{"error": "command not found"})
			return
		}

		// create a temporary file for glazed output
		tmpFile, err := os.CreateTemp("/tmp", fmt.Sprintf("glazed-output-*.%s", fileName))
		if err != nil {
			c.JSON(500, gin.H{"error": "could not create temporary file"})
			return
		}
		defer os.Remove(tmpFile.Name())

		// now check file suffix for content-type
		glazedParameters := map[string]interface{}{
			"output-file": tmpFile.Name(),
		}
		if strings.HasSuffix(fileName, ".csv") {
			glazedParameters["output"] = "table"
			glazedParameters["table-format"] = "csv"
		} else if strings.HasSuffix(fileName, ".tsv") {
			glazedParameters["output"] = "table"
			glazedParameters["table-format"] = "tsv"
		} else if strings.HasSuffix(fileName, ".md") {
			glazedParameters["output"] = "table"
			glazedParameters["table-format"] = "markdown"
		} else if strings.HasSuffix(fileName, ".html") {
			glazedParameters["output"] = "table"
			glazedParameters["table-format"] = "html"
		} else if strings.HasSuffix(fileName, ".json") {
			glazedParameters["output"] = "json"
		} else if strings.HasSuffix(fileName, ".yaml") {
			glazedParameters["yaml"] = "yaml"
		} else if strings.HasSuffix(fileName, ".xlsx") {
			glazedParameters["output"] = "excel"
		} else if strings.HasSuffix(fileName, ".txt") {
			glazedParameters["output"] = "table"
			glazedParameters["table-format"] = "ascii"
		} else {
			c.JSON(500, gin.H{"error": "could not determine output format"})
			return
		}

		glazedParameterLayers, err := cli.NewGlazedParameterLayers()
		if err != nil {
			c.JSON(500, gin.H{"error": "could not create glazed parameter layers"})
			return
		}

		handle := server.HandleSimpleQueryOutputFileCommand(
			sqlCommand,
			tmpFile.Name(),
			fileName,
			glazed.WithParserOptions(
				glazed.WithCustomizedParameterLayerParser(glazedParameterLayers, glazedParameters),
				glazed.WithStaticLayer("sqleton-connection", sqletonConnectionLayer.Parameters),
				glazed.WithStaticLayer("dbt", dbtConnectionLayer.Parameters),
			),
		)

		handle(c)
	})

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		return r.Watch(ctx, yamlLoader, nil,
			watcher.WithMask("**/*.yaml"),
		)
	})
	errGroup.Go(func() error {
		return server.Run(ctx)
	})
	errGroup.Go(func() error {
		return helpers.CancelOnSignal(ctx, os.Interrupt, cancel)
	})

	err = errGroup.Wait()
	if err != nil {
		return err
	}

	return nil
}

func getRepositoryCommand(c *gin.Context, r *repositories.Repository, commandPath string) (cmds.GlazeCommand, bool) {
	path := strings.Split(commandPath, "/")
	commands := r.Root.CollectCommands(path, false)
	if len(commands) == 0 {
		c.JSON(404, gin.H{"error": "command not found"})
		return nil, false
	}

	if len(commands) > 1 {
		c.JSON(404, gin.H{"error": "ambiguous command"})
		return nil, false
	}

	sqlCommand, ok := commands[0].(cmds.GlazeCommand)
	if !ok || sqlCommand == nil {
		c.JSON(500, gin.H{"error": "command is not a sql command"})
	}
	return sqlCommand, true
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
			parameters.NewParameterDefinition(
				"dev",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Run in development mode"),
				parameters.WithDefault(false),
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
