package main

import (
	"embed"
	"fmt"
	clay "github.com/go-go-golems/clay/pkg"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/help"
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
	_ = rootCmd.Execute()
}

//go:embed doc/*
var docFS embed.FS

//go:embed queries/*
var queriesFS embed.FS

func init() {
	helpSystem := help.NewHelpSystem()
	err := helpSystem.LoadSectionsFromFS(docFS, ".")
	if err != nil {
		panic(err)
	}

	helpFunc, usageFunc := help.GetCobraHelpUsageFuncs(helpSystem)
	helpTemplate, usageTemplate := help.GetCobraHelpUsageTemplates(helpSystem)

	_ = usageFunc
	_ = usageTemplate

	rootCmd.SetHelpFunc(helpFunc)
	rootCmd.SetUsageFunc(usageFunc)
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetUsageTemplate(usageTemplate)

	helpCmd := help.NewCobraHelpCommand(helpSystem)
	rootCmd.SetHelpCommand(helpCmd)

	rootCmd.AddCommand(cmds.DbCmd)

	dbtParameterLayer, err := pkg.NewDbtParameterLayer()
	if err != nil {
		panic(err)
	}
	sqlConnectionParameterLayer, err := pkg.NewSqlConnectionParameterLayer()
	if err != nil {
		panic(err)
	}

	runCommand, err := cmds.NewRunCommand(pkg.OpenDatabaseFromSqletonConnectionLayer,
		glazed_cmds.WithLayers(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		panic(err)
	}
	cobraRunCommand, err := cli.BuildCobraCommand(runCommand)
	if err != nil {
		panic(err)
	}
	rootCmd.AddCommand(cobraRunCommand)

	selectCommand, err := cmds.NewSelectCommand(pkg.OpenDatabaseFromSqletonConnectionLayer,
		glazed_cmds.WithLayers(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		panic(err)
	}
	cobraSelectCommand, err := cli.BuildCobraCommand(selectCommand)
	if err != nil {
		panic(err)
	}
	rootCmd.AddCommand(cobraSelectCommand)

	queryCommand, err := cmds.NewQueryCommand(pkg.OpenDatabaseFromSqletonConnectionLayer,
		glazed_cmds.WithLayers(
			dbtParameterLayer,
			sqlConnectionParameterLayer,
		))
	if err != nil {
		panic(err)
	}
	cobraQueryCommand, err := cli.BuildCobraCommand(queryCommand)
	if err != nil {
		panic(err)
	}
	rootCmd.AddCommand(cobraQueryCommand)

	rootCmd.AddCommand(cmds.MysqlCmd)

	err = clay.InitViper("sqleton", rootCmd)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing config: %s\n", err)
		os.Exit(1)
	}
	err = clay.InitLogger()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing logger: %s\n", err)
		os.Exit(1)
	}

	repositories := viper.GetStringSlice("repositories")

	defaultDirectory := "$HOME/.sqleton/queries"
	repositories = append(repositories, defaultDirectory)

	locations := clay.CommandLocations{
		Embedded: []clay.EmbeddedCommandLocation{
			{
				FS:      queriesFS,
				Name:    "embed",
				Root:    ".",
				DocRoot: "queries/doc",
			},
		},
		Repositories: repositories,
	}

	yamlLoader := glazed_cmds.NewYAMLFSCommandLoader(
		&pkg.SqlCommandLoader{
			DBConnectionFactory: pkg.OpenDatabaseFromSqletonConnectionLayer,
		}, "", "")
	commands, aliases, err := locations.LoadCommands(yamlLoader, helpSystem, rootCmd)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	sqlCommands, ok := clay.CastList[*pkg.SqlCommand](commands)
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}
	queriesCommand, err := cmds.NewQueriesCommand(sqlCommands, aliases)
	if err != nil {
		panic(err)
	}
	cobraQueriesCommand, err := cli.BuildCobraCommand(queriesCommand)
	if err != nil {
		panic(err)
	}

	rootCmd.AddCommand(cobraQueriesCommand)
}
