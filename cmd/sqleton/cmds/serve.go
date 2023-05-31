package cmds

import (
	"context"
	"embed"
	"fmt"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers"
	parka "github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/handlers"
	"github.com/go-go-golems/parka/pkg/handlers/command-dir"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/handlers/template-dir"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
)

type ServeCommand struct {
	description         *cmds.CommandDescription
	dbConnectionFactory pkg.DBConnectionFactory
	repositories        []string
}

func (s *ServeCommand) Description() *cmds.CommandDescription {
	return s.description
}

//go:embed templates
var embeddedFiles embed.FS

//go:embed static
var staticFiles embed.FS

func (s *ServeCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
) error {
	// now set up parka server
	port := ps["serve-port"].(int)
	host := ps["serve-host"].(string)
	dev, _ := ps["dev"].(bool)

	serverOptions := []parka.ServerOption{
		parka.WithPort(uint16(port)),
		parka.WithAddress(host),
		parka.WithGzip(),
	}

	configFile := &config.Config{
		Routes: []*config.Route{
			{
				Path: "/",
				CommandDirectory: &config.CommandDir{
					Repositories: s.repositories,
				},
			},
		},
	}

	contentDirs := ps["content-dirs"].([]string)

	if len(contentDirs) > 1 {
		return fmt.Errorf("only one content directory is supported at the moment")
	}

	if len(contentDirs) == 1 {
		configFile.Routes = append(configFile.Routes, &config.Route{
			Path: "/",
			TemplateDirectory: &config.TemplateDir{
				LocalDirectory: contentDirs[0],
			},
		})
	}

	// static paths
	sqletonRendererOptions := []render.RendererOption{}
	if dev {
		configFile.Routes = append(configFile.Routes, &config.Route{
			Path: "/static",
			Static: &config.Static{
				LocalPath: "cmd/sqleton/cmds/static",
			},
		})

		templateLookup := render.NewLookupTemplateFromFS(
			render.WithFS(os.DirFS(".")),
			render.WithBaseDir("cmd/sqleton/cmds/templates/static"),
			render.WithPatterns("**/*.tmpl.*"),
			render.WithAlwaysReload(true),
		)
		err := templateLookup.Reload()
		if err != nil {
			return fmt.Errorf("failed to load local template: %w", err)
		}
		sqletonRendererOptions = append(sqletonRendererOptions,
			render.WithAppendTemplateLookups(templateLookup),
		)
	} else {
		templateLookup := render.NewLookupTemplateFromFS(
			render.WithFS(embeddedFiles),
			render.WithBaseDir("templates"),
			render.WithPatterns("**/*.tmpl.*"),
		)
		err := templateLookup.Reload()
		if err != nil {
			return fmt.Errorf("failed to load embedded template: %w", err)
		}
		sqletonRendererOptions = append(sqletonRendererOptions,
			render.WithAppendTemplateLookups(templateLookup),
		)
		serverOptions = append(serverOptions,
			parka.WithStaticPaths(
				parka.NewStaticPath(http.FS(parka.NewAddPrefixPathFS(staticFiles, "static/")), "/static"),
			),
		)
	}

	serverOptions = append(serverOptions,
		parka.WithDefaultParkaStaticPaths(),
	)

	server, err := parka.NewServer(serverOptions...)
	if err != nil {
		return err
	}

	server.Router.StaticFileFS(
		"favicon.ico",
		"templates/favicon.ico",
		http.FS(embeddedFiles),
	)

	// This section configures the command directory default setting specific to sqleton
	sqletonConnectionLayer := parsedLayers["sqleton-connection"]
	dbtConnectionLayer := parsedLayers["dbt"]

	// commandDirHandlerOptions will apply to all command dirs loaded by the server
	commandDirHandlerOptions := []command_dir.CommandDirHandlerOption{}
	if dev {
		commandDirHandlerOptions = append(commandDirHandlerOptions,
			command_dir.WithTemplateLookup(
				render.NewLookupTemplateFromFS(
					render.WithFS(os.DirFS(".")),
					render.WithBaseDir("cmd/sqleton/cmds/templates"),
				)),
		)
	} else {
		commandDirHandlerOptions = append(commandDirHandlerOptions,
			command_dir.WithTemplateLookup(
				render.NewLookupTemplateFromFS(
					render.WithFS(embeddedFiles),
					render.WithBaseDir("templates"),
				)),
		)
	}
	commandDirHandlerOptions = append(commandDirHandlerOptions,
		command_dir.WithReplaceOverrideLayer(
			dbtConnectionLayer.Layer.GetSlug(),
			dbtConnectionLayer.Parameters,
		),
		command_dir.WithReplaceOverrideLayer(
			sqletonConnectionLayer.Layer.GetSlug(),
			sqletonConnectionLayer.Parameters,
		),
		command_dir.WithDefaultTemplateName("data-tables.tmpl.html"),
		command_dir.WithDefaultIndexTemplateName(""),
		command_dir.WithDevMode(dev),
	)

	// templateDirHandlerOptions
	parkaDefaultRendererOptions, err := parka.GetDefaultParkaRendererOptions()
	if err != nil {
		return fmt.Errorf("failed to get default parka renderer options: %w", err)
	}

	templateDirHandlerOptions := []template_dir.TemplateDirHandlerOption{
		template_dir.WithAppendRendererOptions(parkaDefaultRendererOptions...),
		// add lookup functions for data-tables.tmpl.html and others
		template_dir.WithAppendRendererOptions(sqletonRendererOptions...),
		template_dir.WithAlwaysReload(dev),
	}

	cfh := handlers.NewConfigFileHandler(
		configFile,
		handlers.WithAppendCommandDirHandlerOptions(commandDirHandlerOptions...),
		handlers.WithAppendTemplateDirHandlerOptions(templateDirHandlerOptions...),
		handlers.WithRepositoryFactory(CreateSqletonRepository),
	)

	err = cfh.Serve(server)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		return cfh.Watch(ctx)
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
			parameters.NewParameterDefinition(
				"content-dirs",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Serve static and templated files from these directories"),
				parameters.WithDefault([]string{}),
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
	}
}

// CreateSqletonRepository uses the configured repositories to load a single repository watcher, and load all
// the necessary commands and aliases at startup.
//
// NOTE(manuel, 2023-05-26) This could probably be extracted out of the CommandHandler and maybe submitted as
// a utility, as this currently ties the YAML load and the whole sqleton thing directly into the CommandDirHandler.
func CreateSqletonRepository(dirs []string) (*repositories.Repository, error) {
	yamlFSLoader := loaders.NewYAMLFSCommandLoader(&pkg.SqlCommandLoader{
		DBConnectionFactory: pkg.OpenDatabaseFromSqletonConnectionLayer,
	})
	yamlLoader := &loaders.YAMLReaderCommandLoader{
		YAMLCommandLoader: &pkg.SqlCommandLoader{
			DBConnectionFactory: pkg.OpenDatabaseFromSqletonConnectionLayer,
		},
	}

	r := repositories.NewRepository(
		repositories.WithDirectories(dirs),
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
			// NOTE(manuel, 2023-05-25) Regarding the above TODO, why?
			// We don't need to recompute the func, since it fetches the command at runtime.
			return nil
		}),
		repositories.WithCommandLoader(yamlLoader),
		repositories.WithFSLoader(yamlFSLoader),
	)

	err := r.LoadCommands()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	return r, nil
}
