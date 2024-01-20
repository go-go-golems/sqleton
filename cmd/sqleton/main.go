package main

import (
	"embed"
	"fmt"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/clay/pkg/cmds/ls-commands"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/sqleton/cmd/sqleton/cmds"
	sqleton_cmds "github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/go-go-golems/sqleton/pkg/flags"
	"github.com/pkg/profile"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	"os/signal"
	"syscall"
)

import _ "net/http/pprof"

var version = "dev"
var profiler interface {
	Stop()
}

var rootCmd = &cobra.Command{
	Use:   "sqleton",
	Short: "sqleton runs SQL queries out of template files",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// reinitialize the logger because we can now parse --log-level and co
		// from the command line flag
		err := clay.InitLogger()
		cobra.CheckErr(err)

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
	// first, check if the args are "run-command file.yaml",
	// because we need to load the file and then run the command itself.
	// we need to do this before cobra, because we don't know which flags to load yet
	if len(os.Args) >= 3 && os.Args[1] == "run-command" && os.Args[2] != "--help" {
		// load the command
		loader := &sqleton_cmds.SqlCommandLoader{
			DBConnectionFactory: sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		}
		fs_, filePath, err := loaders.FileNameToFsFilePath(os.Args[2])
		if err != nil {
			fmt.Printf("Could not get absolute path: %v\n", err)
			os.Exit(1)
		}
		cmds, err := loader.LoadCommands(fs_, filePath, []glazed_cmds.CommandDescriptionOption{}, []alias.Option{})
		if err != nil {
			fmt.Printf("Could not load command: %v\n", err)
			os.Exit(1)
		}
		if len(cmds) != 1 {
			fmt.Printf("Expected exactly one command, got %d", len(cmds))
		}

		glazeCommand, ok := cmds[0].(glazed_cmds.GlazeCommand)
		if !ok {
			fmt.Printf("Expected GlazeCommand, got %T", cmds[0])
			os.Exit(1)
		}

		cobraCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(glazeCommand)
		if err != nil {
			fmt.Printf("Could not build cobra command: %v\n", err)
			os.Exit(1)
		}

		_, err = initRootCmd()
		cobra.CheckErr(err)

		rootCmd.AddCommand(cobraCommand)
		restArgs := os.Args[3:]
		os.Args = append([]string{os.Args[0], cobraCommand.Use}, restArgs...)
	} else {
		helpSystem, err := initRootCmd()
		cobra.CheckErr(err)

		err = initAllCommands(helpSystem)
		cobra.CheckErr(err)
	}

	err := rootCmd.Execute()
	cobra.CheckErr(err)
}

var runCommandCmd = &cobra.Command{
	Use:   "run-command",
	Short: "Run a command from a file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		panic(fmt.Errorf("not implemented"))
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

	helpSystem.SetupCobraRootCommand(rootCmd)

	err = clay.InitViper("sqleton", rootCmd)
	cobra.CheckErr(err)
	err = clay.InitLogger()
	cobra.CheckErr(err)

	rootCmd.AddCommand(runCommandCmd)

	rootCmd.AddCommand(cmds.NewCodegenCommand())
	return helpSystem, nil
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
		glazed_cmds.WithLayersList(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		return err
	}

	cobraRunCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(runCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraRunCommand)

	selectCommand, err := cmds.NewSelectCommand(sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		glazed_cmds.WithLayersList(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		return err
	}
	cobraSelectCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(selectCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraSelectCommand)

	queryCommand, err := cmds.NewQueryCommand(
		sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		glazed_cmds.WithLayersList(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		return err
	}
	cobraQueryCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(queryCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraQueryCommand)

	rootCmd.AddCommand(cmds.MysqlCmd)

	repositoryPaths := viper.GetStringSlice("repositories")

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
			Directory:        dir,
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
		cli.WithCobraMiddlewaresFunc(sql.GetCobraCommandSqletonMiddlewares),
		cli.WithCobraShortHelpLayers(layers.DefaultSlug, sql.DbtSlug, sql.SqlConnectionSlug, flags.SqlHelpersSlug),
	)
	if err != nil {
		return err
	}

	serveCommand, err := cmds.NewServeCommand(
		sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		repositoryPaths,
	)
	if err != nil {
		return err
	}
	cobraServeCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(serveCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraServeCommand)

	queriesCommand, err := ls_commands.NewListCommandsCommand(allCommands,
		ls_commands.WithCommandDescriptionOptions(
			glazed_cmds.WithShort("Commands related to sqleton queries"),
		),
		ls_commands.WithAddCommandToRowFunc(func(
			command glazed_cmds.Command,
			row types.Row,
			parsedLayers *layers.ParsedLayers,
		) ([]types.Row, error) {
			ret := []types.Row{row}
			switch c := command.(type) {
			case *sqleton_cmds.SqlCommand:
				row.Set("query", c.Query)
				row.Set("type", "sql")
			default:
			}

			return ret, nil
		}),
	)

	if err != nil {
		return err
	}
	cobraQueriesCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(queriesCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraQueriesCommand)

	rootCmd.PersistentFlags().Bool("mem-profile", false, "Enable memory profiling")

	return nil
}
