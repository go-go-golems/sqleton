package cmds

import (
	"context"
	"embed"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers"
	parka "github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/go-go-golems/sqleton/pkg/serve"
	"github.com/go-go-golems/sqleton/pkg/serve/command-dir"
	"github.com/go-go-golems/sqleton/pkg/serve/config"
	template_dir "github.com/go-go-golems/sqleton/pkg/serve/template-dir"
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

		templateLookup, err := render.LookupTemplateFromFSReloadable(
			os.DirFS("."),
			"cmd/sqleton/cmds/templates/static",
			"**/*.tmpl.*",
		)
		if err != nil {
			return fmt.Errorf("failed to load local template: %w", err)
		}
		sqletonRendererOptions = append(sqletonRendererOptions,
			render.WithAppendTemplateLookups(templateLookup),
		)
	} else {
		embeddedTemplateLookup, err := render.LookupTemplateFromFS(embeddedFiles, "templates", "**/*.tmpl.*")
		if err != nil {
			return fmt.Errorf("failed to load embedded template: %w", err)
		}
		sqletonRendererOptions = append(sqletonRendererOptions,
			render.WithAppendTemplateLookups(embeddedTemplateLookup),
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

	commandDirHandlerOptions := []command_dir.CommandDirHandlerOption{}
	if dev {
		commandDirHandlerOptions = append(commandDirHandlerOptions,
			command_dir.WithDefaultTemplateFS(os.DirFS("."), "cmd/sqleton/cmds/templates"),
		)
	} else {
		commandDirHandlerOptions = append(commandDirHandlerOptions,
			command_dir.WithDefaultTemplateFS(embeddedFiles, "templates"),
		)
	}
	commandDirHandlerOptions = append(commandDirHandlerOptions,
		command_dir.WithDbtConnectionLayer(dbtConnectionLayer),
		command_dir.WithSqletonConnectionLayer(sqletonConnectionLayer),
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
	}

	cfh := serve.NewConfigFileHandler(
		configFile,
		serve.WithAppendCommandDirHandlerOptions(commandDirHandlerOptions...),
		serve.WithAppendTemplateDirHandlerOptions(templateDirHandlerOptions...),
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
