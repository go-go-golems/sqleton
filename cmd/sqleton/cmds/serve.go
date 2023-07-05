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
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	"github.com/go-go-golems/parka/pkg/handlers"
	"github.com/go-go-golems/parka/pkg/handlers/command-dir"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	"github.com/go-go-golems/parka/pkg/handlers/template"
	"github.com/go-go-golems/parka/pkg/handlers/template-dir"
	"github.com/go-go-golems/parka/pkg/server"
	"github.com/go-go-golems/parka/pkg/utils/fs"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"net/http"
	"os"
	"path/filepath"
)

type ServeCommand struct {
	description         *cmds.CommandDescription
	dbConnectionFactory pkg.DBConnectionFactory
	repositories        []string
}

func (s *ServeCommand) Description() *cmds.CommandDescription {
	return s.description
}

//go:embed static
var staticFiles embed.FS

func (s *ServeCommand) runWithConfigFile(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	configFilePath string,
	serverOptions []server.ServerOption,
) error {
	configData, err := os.ReadFile(configFilePath)
	if err != nil {
		return err
	}

	configFile, err := config.ParseConfig(configData)
	if err != nil {
		return err
	}

	server_, err := server.NewServer(serverOptions...)
	if err != nil {
		return err
	}

	debug := ps["debug"].(bool)
	if debug {
		server_.RegisterDebugRoutes()
	}

	commandDirHandlerOptions := []command_dir.CommandDirHandlerOption{}
	templateDirHandlerOptions := []template_dir.TemplateDirHandlerOption{}

	sqlConnectionLayer := parsedLayers["sql-connection"]
	if sqlConnectionLayer == nil {
		return errors.New("sql-connection layer not found")
	}
	dbtConnectionLayer := parsedLayers["dbt"]
	if dbtConnectionLayer == nil {
		return errors.New("dbt layer not found")
	}

	// TODO(manuel, 2023-06-20): These should be able to be set from the config file itself.
	// See: https://github.com/go-go-golems/parka/issues/51
	devMode := ps["dev"].(bool)
	commandDirHandlerOptions = append(
		commandDirHandlerOptions,
		command_dir.WithLayerDefaults(
			sqlConnectionLayer.Layer.GetSlug(),
			sqlConnectionLayer.Parameters,
		),
		command_dir.WithLayerDefaults(
			dbtConnectionLayer.Layer.GetSlug(),
			dbtConnectionLayer.Parameters,
		),
		command_dir.WithDefaultTemplateName("data-tables.tmpl.html"),
		command_dir.WithDefaultIndexTemplateName("index.tmpl.html"),
		command_dir.WithDevMode(devMode),
	)

	templateDirHandlerOptions = append(
		// pass in the default parka renderer options for being able to render markdown files
		templateDirHandlerOptions,
		template_dir.WithAlwaysReload(devMode),
	)

	templateHandlerOptions := []template.TemplateHandlerOption{
		template.WithAlwaysReload(devMode),
	}

	// TODO(manuel, 2023-06-17): we need to
	//
	// create the server
	// - gather server options
	//   - [x] port, address, gzip (passed in through the call)
	//   - [x] static paths (from embedFS static/) -> can be done through normal option
	//   - default parka static paths: /dist from GetParkaStaticHttpFS
	//   - favicon.ico from embeddedFiles templates/favicon.ico
	//
	// for the config file handler:
	// - [x] gather commandDirHandlerOptions
	//   - [x] templateLookup from cmds/templates/
	//      - should be handled by the templateDirectoryHandler creation function
	//   - [x] override dbt-connection and sql-connection layer from parsedLayers
	//   - [x] defaultTemplateName data-tables.tmpl.html
	//     - should be set from the config file, but setting it in the code will do for the first revision
	//   - [x] defaultIndexTemplateName
	//     - see comment for defaultTemplateName, and also https://github.com/go-go-golems/parka/issues/51
	//   - [x] devMode
	// - [x] gather templateDirHandlerOptions
	//   - [x] default renderer options
	//   - [x] sqletonRendererOptions (seems to be the templateLookup, so done through the config file?)
	// - [x] get repository factory

	cfh := handlers.NewConfigFileHandler(
		configFile,
		handlers.WithAppendCommandDirHandlerOptions(commandDirHandlerOptions...),
		handlers.WithAppendTemplateDirHandlerOptions(templateDirHandlerOptions...),
		handlers.WithAppendTemplateHandlerOptions(templateHandlerOptions...),
		handlers.WithRepositoryFactory(pkg.NewRepositoryFactory()),
		handlers.WithDevMode(devMode),
	)

	err = runConfigFileHandler(ctx, server_, cfh)
	if err != nil {
		return err
	}
	return nil
}

func (s *ServeCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
) error {
	// now set up parka server
	port := ps["serve-port"].(int)
	host := ps["serve-host"].(string)
	debug := ps["debug"].(bool)
	dev, _ := ps["dev"].(bool)

	serverOptions := []server.ServerOption{
		server.WithPort(uint16(port)),
		server.WithAddress(host),
		server.WithGzip(),
	}

	if configFilePath, ok := ps["config-file"]; ok {
		return s.runWithConfigFile(ctx, parsedLayers, ps, configFilePath.(string), serverOptions)
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
		// resolve directory to absolute directory
		dir, err := filepath.Abs(contentDirs[0])
		if err != nil {
			return err
		}
		configFile.Routes = append(configFile.Routes, &config.Route{
			Path: "/",
			TemplateDirectory: &config.TemplateDir{
				LocalDirectory: dir,
			},
		})
	}

	// static paths
	if dev {
		configFile.Routes = append(configFile.Routes, &config.Route{
			Path: "/static",
			Static: &config.Static{
				LocalPath: "cmd/sqleton/cmds/static",
			},
		})

	} else {
		serverOptions = append(serverOptions,
			server.WithStaticPaths(
				fs.NewStaticPath(http.FS(fs.NewAddPrefixPathFS(staticFiles, "static/")), "/static"),
			),
		)
	}

	server_, err := server.NewServer(serverOptions...)
	if err != nil {
		return err
	}

	if debug {
		server_.RegisterDebugRoutes()
	}

	server_.Router.StaticFileFS(
		"favicon.ico",
		"static/favicon.ico",
		http.FS(staticFiles),
	)

	// This section configures the command directory default setting specific to sqleton
	sqlConnectionLayer := parsedLayers["sql-connection"]
	if sqlConnectionLayer == nil {
		return fmt.Errorf("sql-connection layer is required")
	}
	dbtConnectionLayer := parsedLayers["dbt"]
	if dbtConnectionLayer == nil {
		return fmt.Errorf("dbt layer is required")
	}

	// commandDirHandlerOptions will apply to all command dirs loaded by the server
	commandDirHandlerOptions := []command_dir.CommandDirHandlerOption{
		command_dir.WithTemplateLookup(datatables.NewDataTablesLookupTemplate()),
		command_dir.WithReplaceOverrideLayer(
			dbtConnectionLayer.Layer.GetSlug(),
			dbtConnectionLayer.Parameters,
		),
		command_dir.WithReplaceOverrideLayer(
			sqlConnectionLayer.Layer.GetSlug(),
			sqlConnectionLayer.Parameters,
		),
		command_dir.WithDefaultTemplateName("data-tables.tmpl.html"),
		command_dir.WithDefaultIndexTemplateName(""),
		command_dir.WithDevMode(dev),
	}

	templateDirHandlerOptions := []template_dir.TemplateDirHandlerOption{
		// add lookup functions for data-tables.tmpl.html and others
		template_dir.WithAlwaysReload(dev),
	}

	err = configFile.Initialize()
	if err != nil {
		return err
	}

	cfh := handlers.NewConfigFileHandler(
		configFile,
		handlers.WithAppendCommandDirHandlerOptions(commandDirHandlerOptions...),
		handlers.WithAppendTemplateDirHandlerOptions(templateDirHandlerOptions...),
		handlers.WithRepositoryFactory(pkg.NewRepositoryFactory()),
		handlers.WithDevMode(dev),
	)

	err = runConfigFileHandler(ctx, server_, cfh)
	if err != nil {
		return err
	}
	return nil
}

// runConfigFileHandler runs the config file handler and the server.
// The config file handler will watch the config file for changes and reload the server.
// The server will run until the context is canceled (which can be done through Ctrl-C).
func runConfigFileHandler(ctx context.Context, server_ *server.Server, cfh *handlers.ConfigFileHandler) error {
	err := cfh.Serve(server_)
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
		return server_.Run(ctx)
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
				"debug",
				parameters.ParameterTypeBool,
				parameters.WithHelp("Run in debug mode (expose /debug/pprof routes)"),
				parameters.WithDefault(false),
			),
			parameters.NewParameterDefinition(
				"content-dirs",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Serve static and templated files from these directories"),
				parameters.WithDefault([]string{}),
			),
			parameters.NewParameterDefinition(
				"config-file",
				parameters.ParameterTypeString,
				parameters.WithHelp("Config file to configure the serve functionality"),
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
