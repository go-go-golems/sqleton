package cmds

import (
	"context"
	"embed"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/parka/pkg/glazed/handlers/datatables"
	"github.com/go-go-golems/parka/pkg/handlers"
	"github.com/go-go-golems/parka/pkg/handlers/command"
	"github.com/go-go-golems/parka/pkg/handlers/command-dir"
	"github.com/go-go-golems/parka/pkg/handlers/config"
	generic_command "github.com/go-go-golems/parka/pkg/handlers/generic-command"
	"github.com/go-go-golems/parka/pkg/handlers/template"
	"github.com/go-go-golems/parka/pkg/handlers/template-dir"
	"github.com/go-go-golems/parka/pkg/server"
	"github.com/go-go-golems/parka/pkg/utils/fs"
	sqleton_cmds "github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

type ServeCommand struct {
	*cmds.CommandDescription
	dbConnectionFactory sql.DBConnectionFactory
	repositories        []string
}

var _ cmds.BareCommand = (*ServeCommand)(nil)

type ServeSettings struct {
	Dev         bool     `glazed:"dev"`
	Debug       bool     `glazed:"debug"`
	ServePort   int      `glazed:"serve-port"`
	ServeHost   string   `glazed:"serve-host"`
	ContentDirs []string `glazed:"content-dirs"`
	ConfigFile  string   `glazed:"config-file"`
}

func NewServeCommand(
	dbConnectionFactory sql.DBConnectionFactory,
	repositoryPaths []string,
	options ...cmds.CommandDescriptionOption,
) (*ServeCommand, error) {
	sqlConnectionParameterLayer, err := sql.NewSqlConnectionParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL connection parameter layer")
	}
	dbtParameterLayer, err := sql.NewDbtParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create dbt parameter layer")
	}

	options_ := append(options,
		cmds.WithShort("Serve the API"),
		cmds.WithArguments(),
		cmds.WithFlags(
			fields.New(
				"serve-port",
				fields.TypeInteger,
				fields.WithShortFlag("p"),
				fields.WithHelp("Port to serve the API on"),
				fields.WithDefault(8080),
			),
			fields.New(
				"serve-host",
				fields.TypeString,
				fields.WithHelp("Host to serve the API on"),
				fields.WithDefault("localhost"),
			),
			fields.New(
				"dev",
				fields.TypeBool,
				fields.WithHelp("Run in development mode"),
				fields.WithDefault(false),
			),
			fields.New(
				"debug",
				fields.TypeBool,
				fields.WithHelp("Run in debug mode (expose /debug/pprof routes)"),
				fields.WithDefault(false),
			),
			fields.New(
				"content-dirs",
				fields.TypeStringList,
				fields.WithHelp("Serve static and templated files from these directories"),
				fields.WithDefault([]string{}),
			),
			fields.New(
				"config-file",
				fields.TypeString,
				fields.WithHelp("Config file to configure the serve functionality"),
			),
		),
		cmds.WithSections(sqlConnectionParameterLayer, dbtParameterLayer),
	)
	return &ServeCommand{
		dbConnectionFactory: dbConnectionFactory,
		CommandDescription: cmds.NewCommandDescription(
			"serve",
			options_...,
		),
		repositories: repositoryPaths,
	}, nil
}

// NOTE(manuel, 2023-12-13) Why do we embed the favicon.ico here?
//
//go:embed static
var staticFiles embed.FS

func (s *ServeCommand) runWithConfigFile(
	ctx context.Context,
	parsedValues *values.Values,
	configFilePath string,
	serverOptions []server.ServerOption,
) error {
	ss := &ServeSettings{}
	err := parsedValues.DecodeSectionInto(schema.DefaultSlug, ss)
	if err != nil {
		return err
	}

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

	if ss.Debug {
		server_.RegisterDebugRoutes()
	}

	commandDirHandlerOptions := []command_dir.CommandDirHandlerOption{}
	templateDirHandlerOptions := []template_dir.TemplateDirHandlerOption{}

	sqlConnectionValues, ok := parsedValues.Get(sql.SqlConnectionSlug)
	if !ok || sqlConnectionValues == nil {
		return errors.New("sql-connection layer not found")
	}
	dbtConnectionValues, ok := parsedValues.Get(sql.DbtSlug)
	if !ok || dbtConnectionValues == nil {
		return errors.New("dbt layer not found")
	}

	// TODO(manuel, 2023-06-20): These should be able to be set from the config file itself.
	// See: https://github.com/go-go-golems/parka/issues/51
	devMode := ss.Dev

	// NOTE(manuel, 2023-12-13) Why do we append these to the config file?
	commandDirHandlerOptions = append(
		commandDirHandlerOptions,
		command_dir.WithGenericCommandHandlerOptions(
			generic_command.WithParameterFilterOptions(
				// I think this is correct and sets the connection settings?
				config.WithMergeOverrideLayer(
					sqlConnectionValues.Section.GetSlug(),
					sqlConnectionValues.Fields.ToMap(),
				),
				config.WithMergeOverrideLayer(
					dbtConnectionValues.Section.GetSlug(),
					dbtConnectionValues.Fields.ToMap(),
				),
			),
			generic_command.WithDefaultTemplateName("data-tables.tmpl.html"),
			generic_command.WithDefaultIndexTemplateName("commands.tmpl.html"),
		),
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

	cfh := handlers.NewConfigFileHandler(
		configFile,
		handlers.WithAppendCommandDirHandlerOptions(commandDirHandlerOptions...),
		handlers.WithAppendTemplateDirHandlerOptions(templateDirHandlerOptions...),
		handlers.WithAppendTemplateHandlerOptions(templateHandlerOptions...),
		handlers.WithRepositoryFactory(sqleton_cmds.NewRepositoryFactory()),
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
	parsedValues *values.Values,
) error {
	ss := &ServeSettings{}
	err := parsedValues.DecodeSectionInto(schema.DefaultSlug, ss)
	if err != nil {
		return err
	}

	// Validate port is within uint16 range to prevent overflow
	if ss.ServePort < 0 || ss.ServePort > 65535 {
		return errors.Errorf("port number %d is outside the valid range (0-65535)", ss.ServePort)
	}

	serverOptions := []server.ServerOption{
		server.WithPort(uint16(ss.ServePort)),
		server.WithAddress(ss.ServeHost),
		server.WithGzip(),
	}

	// set default logger to log without colors
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, NoColor: true})

	if ss.ConfigFile != "" {
		return s.runWithConfigFile(ctx, parsedValues, ss.ConfigFile, serverOptions)
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

	contentDirs := ss.ContentDirs

	if len(contentDirs) > 1 {
		return errors.Errorf("only one content directory is supported at the moment")
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

	// NOTE(manuel, 2023-12-13) Unsure why we really need the static paths here instead of dealing with this
	// in the package maybe?
	if ss.Dev {
		configFile.Routes = append(configFile.Routes, &config.Route{
			Path: "/static",
			Static: &config.Static{
				LocalPath: "cmd/sqleton/cmd/static",
			},
		})

	} else {
		serverOptions = append(serverOptions,
			server.WithStaticPaths(
				fs.NewStaticPath(fs.NewAddPrefixPathFS(staticFiles, "static/"), "/static"),
				fs.NewStaticPath(staticFiles, "/favicon.ico"),
			),
		)
	}

	server_, err := server.NewServer(serverOptions...)
	if err != nil {
		return err
	}

	if ss.Debug {
		server_.RegisterDebugRoutes()
	}

	// This section configures the command directory default setting specific to sqleton
	sqlConnectionValues, ok := parsedValues.Get(sql.SqlConnectionSlug)
	if !ok || sqlConnectionValues == nil {
		return errors.Errorf("sql-connection layer is required")
	}
	dbtConnectionValues, ok := parsedValues.Get(sql.DbtSlug)
	if !ok || dbtConnectionValues == nil {
		return errors.Errorf("dbt layer is required")
	}

	// commandDirHandlerOptions will apply to all command dirs loaded by the server
	commandDirHandlerOptions := []command_dir.CommandDirHandlerOption{
		command_dir.WithGenericCommandHandlerOptions(
			generic_command.WithTemplateLookup(datatables.NewDataTablesLookupTemplate()),
			generic_command.WithParameterFilterOptions(
				config.WithReplaceOverrideLayer(
					dbtConnectionValues.Section.GetSlug(),
					dbtConnectionValues.Fields.ToMap(),
				),
				config.WithReplaceOverrideLayer(
					sqlConnectionValues.Section.GetSlug(),
					sqlConnectionValues.Fields.ToMap(),
				),
			),
			generic_command.WithDefaultTemplateName("data-tables.tmpl.html"),
			generic_command.WithDefaultIndexTemplateName(""),
		),
		command_dir.WithDevMode(ss.Dev),
	}

	commandHandlerOptions := []command.CommandHandlerOption{
		command.WithGenericCommandHandlerOptions(
			generic_command.WithTemplateLookup(datatables.NewDataTablesLookupTemplate()),
			generic_command.WithParameterFilterOptions(
				config.WithReplaceOverrideLayer(
					dbtConnectionValues.Section.GetSlug(),
					dbtConnectionValues.Fields.ToMap(),
				),
				config.WithReplaceOverrideLayer(
					sqlConnectionValues.Section.GetSlug(),
					sqlConnectionValues.Fields.ToMap(),
				),
			),
			generic_command.WithDefaultTemplateName("data-tables.tmpl.html"),
			generic_command.WithDefaultIndexTemplateName(""),
		),
		command.WithDevMode(ss.Dev),
	}

	templateDirHandlerOptions := []template_dir.TemplateDirHandlerOption{
		// add lookup functions for data-tables.tmpl.html and others
		template_dir.WithAlwaysReload(ss.Dev),
	}

	err = configFile.Initialize()
	if err != nil {
		return err
	}

	cfh := handlers.NewConfigFileHandler(
		configFile,
		handlers.WithAppendCommandDirHandlerOptions(commandDirHandlerOptions...),
		handlers.WithAppendTemplateDirHandlerOptions(templateDirHandlerOptions...),
		handlers.WithAppendCommandHandlerOptions(commandHandlerOptions...),
		handlers.WithRepositoryFactory(sqleton_cmds.NewRepositoryFactory()),
		handlers.WithDevMode(ss.Dev),
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
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	errGroup, ctx := errgroup.WithContext(ctx)
	errGroup.Go(func() error {
		return cfh.Watch(ctx)
	})
	errGroup.Go(func() error {
		return server_.Run(ctx)
	})

	err = errGroup.Wait()
	if err != nil {
		return err
	}

	return nil
}
