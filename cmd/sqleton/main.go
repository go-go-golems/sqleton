package main

import (
	"embed"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	clay "github.com/go-go-golems/clay/pkg"
	clay_doc "github.com/go-go-golems/clay/pkg/doc"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/help"
	help_cmd "github.com/go-go-golems/glazed/pkg/help/cmd"
	"github.com/go-go-golems/glazed/pkg/types"
	parka_doc "github.com/go-go-golems/parka/pkg/doc"
	"github.com/go-go-golems/sqleton/cmd/sqleton/cmds"
	"github.com/go-go-golems/sqleton/cmd/sqleton/cmds/mcp"
	sqleton_cmds "github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/go-go-golems/sqleton/pkg/flags"
	"github.com/pkg/errors"
	"github.com/pkg/profile"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	// #nosec G108 - pprof is imported for profiling and debugging in development environments only.
	// This is gated behind the --mem-profile flag and not enabled by default.
	_ "net/http/pprof"

	clay_commandmeta "github.com/go-go-golems/clay/pkg/cmds/commandmeta"
	clay_profiles "github.com/go-go-golems/clay/pkg/cmds/profiles"
	clay_repositories "github.com/go-go-golems/clay/pkg/cmds/repositories"
)

var version = "dev"
var profiler interface {
	Stop()
}

var rootCmd = &cobra.Command{
	Use:   "sqleton",
	Short: "sqleton runs SQL queries out of template files",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		memProfile, _ := cmd.Flags().GetBool("mem-profile")
		if memProfile {
			log.Info().Msg("Starting memory profiler")
			profiler = profile.Start(profile.MemProfile)

			// on SIGHUP, restart the profiler
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, syscall.SIGHUP)
			go func() {
				for range sigCh {
					log.Info().Msg("Restarting memory profiler")
					profiler.Stop()
					profiler = profile.Start(profile.MemProfile)
				}
			}()
		}
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if profiler != nil {
			log.Info().Msg("Stopping memory profiler")
			profiler.Stop()
		}
	},
	Version: version,
}

func main() {
	helpSystem, err := initRootCmd()
	cobra.CheckErr(err)

	err = initAllCommands(helpSystem)
	cobra.CheckErr(err)

	err = rootCmd.Execute()
	cobra.CheckErr(err)
}

var runCommandCmd = &cobra.Command{
	Use:   "run-command <file> [args...]",
	Short: "Run a command from a local .sql file",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]
		commandArgs := args[1:]

		loader := &sqleton_cmds.SqlCommandLoader{
			DBConnectionFactory: sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		}
		fs_, resolvedPath, err := loaders.FileNameToFsFilePath(filePath)
		if err != nil {
			return errors.Wrap(err, "could not resolve command file path")
		}

		loadedCommands, err := loader.LoadCommands(fs_, resolvedPath, []glazed_cmds.CommandDescriptionOption{}, []alias.Option{})
		if err != nil {
			return errors.Wrap(err, "could not load command")
		}
		if len(loadedCommands) != 1 {
			return errors.Errorf("expected exactly one command, got %d", len(loadedCommands))
		}

		glazeCommand, ok := loadedCommands[0].(glazed_cmds.GlazeCommand)
		if !ok {
			return errors.Errorf("expected GlazeCommand, got %T", loadedCommands[0])
		}

		cobraCommand, err := buildSqletonCobraCommand(glazeCommand)
		if err != nil {
			return errors.Wrap(err, "could not build cobra command")
		}

		cobraCommand.SetArgs(commandArgs)
		cobraCommand.SetContext(cmd.Context())
		return cobraCommand.ExecuteContext(cmd.Context())
	},
}

//go:embed doc/*
var docFS embed.FS

//go:embed queries/*
var queriesFS embed.FS

func initRootCmd() (*help.HelpSystem, error) {
	helpSystem := help.NewHelpSystem()
	err := helpSystem.LoadSectionsFromFS(docFS, ".")
	cobra.CheckErr(err)

	err = parka_doc.AddDocToHelpSystem(helpSystem)
	cobra.CheckErr(err)

	err = clay_doc.AddDocToHelpSystem(helpSystem)
	cobra.CheckErr(err)

	help_cmd.SetupCobraRootCommand(helpSystem, rootCmd)

	err = clay.InitGlazed("sqleton", rootCmd)
	cobra.CheckErr(err)
	rootCmd.AddCommand(runCommandCmd)

	rootCmd.AddCommand(cmds.NewCodegenCommand())
	return helpSystem, nil
}

func buildSqletonCobraCommand(command glazed_cmds.Command, options ...cli.CobraOption) (*cobra.Command, error) {
	return sqleton_cmds.BuildCobraCommandWithSqletonMiddlewares(command, options...)
}

func initAllCommands(helpSystem *help.HelpSystem) error {
	rootCmd.AddCommand(cmds.DbCmd)

	dbtParameterLayer, err := sql.NewDbtParameterLayer()
	if err != nil {
		return err
	}
	sqlConnectionParameterLayer, err := sql.NewSqlConnectionParameterLayer()
	if err != nil {
		return err
	}

	runCommand, err := cmds.NewRunCommand(sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		glazed_cmds.WithSections(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		return err
	}

	cobraRunCommand, err := buildSqletonCobraCommand(runCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraRunCommand)

	selectCommand, err := cmds.NewSelectCommand(sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		glazed_cmds.WithSections(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		return err
	}
	cobraSelectCommand, err := buildSqletonCobraCommand(selectCommand,
		cli.WithCobraShortHelpSections(cmds.SelectSlug),
	)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraSelectCommand)

	queryCommand, err := cmds.NewQueryCommand(
		sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		glazed_cmds.WithSections(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		return err
	}
	cobraQueryCommand, err := buildSqletonCobraCommand(queryCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraQueryCommand)

	repositoryPaths, err := collectRepositoryPaths("sqleton")
	if err != nil {
		return err
	}

	defaultDirectory := "$HOME/.sqleton/queries"
	_, err = os.Stat(os.ExpandEnv(defaultDirectory))
	if err == nil {
		repositoryPaths = append(repositoryPaths, os.ExpandEnv(defaultDirectory))
	}

	loader := &sqleton_cmds.SqlCommandLoader{
		DBConnectionFactory: sql.OpenDatabaseFromDefaultSqlConnectionLayer,
	}
	directories := []repositories.Directory{
		{
			FS:               queriesFS,
			RootDirectory:    "queries",
			RootDocDirectory: "queries/doc",
			Name:             "sqleton",
			SourcePrefix:     "embed",
		}}

	for _, repositoryPath := range repositoryPaths {
		dir := os.ExpandEnv(repositoryPath)
		// check if dir exists
		if fi, err := os.Stat(dir); os.IsNotExist(err) || !fi.IsDir() {
			continue
		}
		directories = append(directories, repositories.Directory{
			FS:               os.DirFS(dir),
			RootDirectory:    ".",
			RootDocDirectory: "doc",
			Name:             dir,
			SourcePrefix:     "file",
		})
	}

	repositories_ := []*repositories.Repository{
		repositories.NewRepository(
			repositories.WithDirectories(directories...),
			repositories.WithCommandLoader(loader),
		),
	}

	allCommands, err := repositories.LoadRepositories(
		helpSystem,
		rootCmd,
		repositories_,
		cli.WithParserConfig(cmds.NewSqletonParserConfig()),
		cli.WithCobraShortHelpSections(schema.DefaultSlug, sql.DbtSlug, sql.SqlConnectionSlug, flags.SqlHelpersSlug),
		cli.WithCreateCommandSettingsSection(),
		cli.WithProfileSettingsSection(),
	)
	if err != nil {
		return err
	}

	// Create and add MCP commands
	mcpCommands := mcp.NewMcpCommands(repositories_)
	mcpCommands.AddToRootCommand(rootCmd)

	serveCommand, err := cmds.NewServeCommand(
		sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		repositoryPaths,
	)
	if err != nil {
		return err
	}
	cobraServeCommand, err := buildSqletonCobraCommand(serveCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraServeCommand)

	// Create and add the unified command management group
	commandManagementCmd, err := clay_commandmeta.NewCommandManagementCommandGroup(
		allCommands, // Pass the loaded commands
		// Pass the existing AddCommandToRowFunc logic as an option
		clay_commandmeta.WithListAddCommandToRowFunc(func(
			command glazed_cmds.Command,
			row types.Row,
			_ *values.Values,
		) ([]types.Row, error) {
			// Example: Set 'type' and 'query' based on command type
			switch c := command.(type) {
			case *sqleton_cmds.SqlCommand:
				row.Set("query", c.Query)
				row.Set("type", "sql")
			case *alias.CommandAlias: // Handle aliases if needed
				row.Set("type", "alias")
				row.Set("aliasFor", c.AliasFor)
			default:
				// Default type handling if needed
				if _, ok := row.Get("type"); !ok {
					row.Set("type", "unknown")
				}
			}
			return []types.Row{row}, nil
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to initialize command management commands: %w", err)
	}
	rootCmd.AddCommand(commandManagementCmd) // Add the group directly to root

	// Create and add the profiles command
	profilesCmd, err := clay_profiles.NewProfilesCommand("sqleton", sqletonInitialProfilesContent)
	if err != nil {
		return fmt.Errorf("failed to initialize profiles command: %w", err)
	}
	rootCmd.AddCommand(profilesCmd)

	// Create and add the repositories command group
	rootCmd.AddCommand(clay_repositories.NewRepositoriesGroupCommand())

	rootCmd.PersistentFlags().Bool("mem-profile", false, "Enable memory profiling")

	return nil
}

// sqletonInitialProfilesContent provides the default YAML content for a new sqleton profiles file.
func sqletonInitialProfilesContent() string {
	return `# Sqleton Profiles Configuration
#
# This file allows defining profiles to override default SQL connection
# settings or query parameters for sqleton commands.
#
# Profiles are selected using the --profile <profile-name> flag.
# Settings within a profile override the default values for the specified layer.
#
# Example:
#
# production-db:
#   # Override settings for the 'sql-connection' layer
#   sql-connection:
#     driver: postgres
#     dsn: "host=prod.db user=reporter password=secret dbname=reports sslmode=require"
#     # You can also specify individual DSN components:
#     # host: prod.db
#     # port: 5432
#     # user: reporter
#     # password: secret
#     # dbname: reports
#     # sslmode: require
#
#   # Override settings for the 'dbt' layer (if using dbt)
#   dbt:
#     dbt-project-path: /path/to/prod/dbt/project
#     dbt-profile: production
#
# You can manage this file using the 'sqleton profiles' commands:
# - list: List all profiles
# - get: Get profile settings
# - set: Set a profile setting
# - delete: Delete a profile, layer, or setting
# - edit: Open this file in your editor
# - init: Create this file if it doesn't exist
# - duplicate: Copy an existing profile
`
}
