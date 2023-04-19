package main

import (
	"embed"
	"fmt"
	clay "github.com/go-go-golems/clay/pkg"
	clay_cmds "github.com/go-go-golems/clay/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/go-go-golems/sqleton/cmd/sqleton/cmds"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var rootCmd = &cobra.Command{
	Use:   "sqleton",
	Short: "sqleton runs SQL queries out of template files",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// reinitialize the logger because we can now parse --log-level and co
		// from the command line flag
		err := clay.InitLogger()
		cobra.CheckErr(err)
	},
}

func main() {
	// first, check if the args are "run-command file.yaml",
	// because we need to load the file and then run the command itself.
	// we need to do this before cobra, because we don't know which flags to load yet
	if len(os.Args) >= 3 && os.Args[1] == "run-command" && os.Args[2] != "--help" {
		// load the command
		loader := &pkg.SqlCommandLoader{
			DBConnectionFactory: pkg.OpenDatabaseFromSqletonConnectionLayer,
		}
		f, err := os.Open(os.Args[2])
		if err != nil {
			fmt.Printf("Could not open file: %v\n", err)
			os.Exit(1)
		}
		cmds, err := loader.LoadCommandFromYAML(f)
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

		cobraCommand, err := cli.BuildCobraCommandFromGlazeCommand(glazeCommand)
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
	return helpSystem, nil
}

func initAllCommands(helpSystem *help.HelpSystem) error {
	rootCmd.AddCommand(cmds.DbCmd)

	dbtParameterLayer, err := pkg.NewDbtParameterLayer()
	if err != nil {
		return err
	}
	sqlConnectionParameterLayer, err := pkg.NewSqlConnectionParameterLayer()
	if err != nil {
		return err
	}

	runCommand, err := cmds.NewRunCommand(pkg.OpenDatabaseFromSqletonConnectionLayer,
		glazed_cmds.WithLayers(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		return err
	}
	cobraRunCommand, err := cli.BuildCobraCommandFromGlazeCommand(runCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraRunCommand)

	selectCommand, err := cmds.NewSelectCommand(pkg.OpenDatabaseFromSqletonConnectionLayer,
		glazed_cmds.WithLayers(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		return err
	}
	cobraSelectCommand, err := cli.BuildCobraCommandFromGlazeCommand(selectCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraSelectCommand)

	queryCommand, err := cmds.NewQueryCommand(
		pkg.OpenDatabaseFromSqletonConnectionLayer,
		glazed_cmds.WithLayers(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		return err
	}
	cobraQueryCommand, err := cli.BuildCobraCommandFromGlazeCommand(queryCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraQueryCommand)

	rootCmd.AddCommand(cmds.MysqlCmd)

	repositories := viper.GetStringSlice("repositories")

	defaultDirectory := "$HOME/.sqleton/queries"
	repositories = append(repositories, os.ExpandEnv(defaultDirectory))

	locations := clay_cmds.CommandLocations{
		Embedded: []clay_cmds.EmbeddedCommandLocation{
			{
				FS:      queriesFS,
				Name:    "sqleton",
				Root:    "queries",
				DocRoot: "queries/doc",
			},
		},
		Repositories: repositories,
	}

	yamlLoader := loaders.NewYAMLFSCommandLoader(&pkg.SqlCommandLoader{
		DBConnectionFactory: pkg.OpenDatabaseFromSqletonConnectionLayer,
	})
	commandLoader := clay_cmds.NewCommandLoader[glazed_cmds.Command](&locations)
	commands, aliases, err := commandLoader.LoadCommands(yamlLoader, helpSystem)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	sqlCommands, ok := cast.CastList[*pkg.SqlCommand](commands)
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}
	glazeCommands, ok := cast.CastList[glazed_cmds.GlazeCommand](commands)
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	err = cli.AddCommandsToRootCommand(rootCmd, glazeCommands, aliases)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	serveCommand := cmds.NewServeCommand(
		pkg.OpenDatabaseFromSqletonConnectionLayer,
		repositories, commands, aliases,
		glazed_cmds.WithLayers(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	cobraServeCommand, err := cli.BuildCobraCommandFromBareCommand(serveCommand)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(cobraServeCommand)

	queriesCommand, err := cmds.NewQueriesCommand(sqlCommands, aliases)
	if err != nil {
		return err
	}
	cobraQueriesCommand, err := cli.BuildCobraCommandFromGlazeCommand(queriesCommand)
	if err != nil {
		return err
	}

	rootCmd.AddCommand(cobraQueriesCommand)

	return nil
}
