package cmds

import (
	"context"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/watcher"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	parka "github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
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
			return nil
		}),
		repositories.WithRemoveCallback(func(cmd cmds.Command) error {
			description := cmd.Description()
			log.Info().Str("name", description.Name).
				Str("source", description.Source).
				Msg("Removing cmd")
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

	glazedParameterLayers, err := cli.NewGlazedParameterLayers()
	if err != nil {
		return err
	}

	sqletonConnectionLayer := parsedLayers["sqleton-connection"]
	dbtConnectionLayer := parsedLayers["dbt"]

	// XXX this won't allow reload, is gin even the best choice here? Guess we can just wildcard /api
	for _, command := range s.commands {
		d := command.Description()
		sqlCommand := command.(*pkg.SqlCommand)
		path := "api/" + strings.Join(d.Parents, "/") + "/" + d.Name
		server.Router.GET(path, server.HandleSimpleQueryCommand(sqlCommand,
			[]parka.ParserHandlerOption{
				parka.WithReplaceParameterLayerParser("sqleton-connection",
					parka.NewStaticParameterDefinitionParser(sqletonConnectionLayer.Parameters)),
				parka.WithReplaceParameterLayerParser("dbt",
					parka.NewStaticParameterDefinitionParser(dbtConnectionLayer.Parameters)),
				parka.WithGlazeOutputParserOption(glazedParameterLayers, "table", "html"),
			},
		))
		server.Router.POST(path, server.HandleSimpleFormCommand(sqlCommand))
	}
	for _, alias := range s.aliases {
		d := alias.Description()
		path := "api/" + strings.Join(d.Parents, "/") + "/" + d.Name
		server.Router.GET(path, server.HandleSimpleQueryCommand(alias,
			[]parka.ParserHandlerOption{
				parka.WithGlazeOutputParserOption(glazedParameterLayers, "table", "html"),
			},
		))
		server.Router.POST(path, server.HandleSimpleFormCommand(alias))
	}

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		return r.Watch(ctx, yamlLoader, nil,
			watcher.WithMask("**/*.yaml"),
		)
	})
	errGroup.Go(func() error {
		return server.Run()
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
