package main

import (
	"embed"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wesen/glazed/pkg/help"
	"github.com/wesen/sqleton/cmd/sqleton/cmds"
	sqleton "github.com/wesen/sqleton/pkg"
	"os"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "sqleton",
	Short: "sqleton runs SQL queries out of template files",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
	},
}

func initCommands(rootCmd *cobra.Command) {
	// Load the variables from the environment
	viper.SetEnvPrefix("sqleton")

	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.sqleton")
	viper.AddConfigPath("/etc/sqleton")

	// Read the configuration file into Viper
	err := viper.ReadInConfig()
	// if the file does not exist, continue normally
	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
		// Config file not found; ignore error
	} else {
		// Config file was found but another error was produced
		cobra.CheckErr(err)
	}
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	// Bind the variables to the command-line flags
	err = viper.BindPFlags(rootCmd.PersistentFlags())
	cobra.CheckErr(err)

	repository := viper.GetString("repository")
	useDefaultDirectory := false
	if repository == "" {
		useDefaultDirectory = true
		repository = "$HOME/.sqleton/queries"
	}

	repository = os.ExpandEnv(repository)

	// check that repository exists and is a directory
	s, err := os.Stat(repository)

	if os.IsNotExist(err) {
		if !useDefaultDirectory {
			cobra.CheckErr(err)
		}
	} else if err != nil {
		cobra.CheckErr(err)
	}

	if s == nil || !s.IsDir() {
		if !useDefaultDirectory {
			cobra.CheckErr("repository is not a directory")
		}
	} else {
		commands, err := sqleton.LoadSqlCommandsFromDirectory(repository, repository)
		cobra.CheckErr(err)
		err = sqleton.AddCommandsToRootCommand(rootCmd, commands)
		cobra.CheckErr(err)
	}
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
	err := helpSystem.LoadSectionsFromEmbedFS(docFS, ".")
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

	// db connection persistent base flags
	rootCmd.PersistentFlags().Bool("use-dbt-profiles", false, "Use dbt profiles.yml to connect to databases")
	rootCmd.PersistentFlags().String("dbt-profiles-path", "", "Path to dbt profiles.yml (default: ~/.dbt/profiles.yml)")
	rootCmd.PersistentFlags().String("dbt-profile", "default", "Name of dbt profile to use (default: default)")

	// more normal flags
	rootCmd.PersistentFlags().StringP("host", "H", "", "Database host")
	rootCmd.PersistentFlags().StringP("database", "D", "", "Database name")
	rootCmd.PersistentFlags().StringP("user", "u", "", "Database user")
	rootCmd.PersistentFlags().StringP("password", "p", "", "Database password")
	rootCmd.PersistentFlags().IntP("port", "P", 3306, "Database port")
	rootCmd.PersistentFlags().StringP("schema", "s", "", "Database schema (when applicable)")
	rootCmd.PersistentFlags().StringP("type", "t", "mysql", "Database type (mysql, postgres, etc.)")

	rootCmd.PersistentFlags().String("repository", "", "Directory with additional commands to load (default ~/.sqleton/queries)")

	// dsn and driver
	rootCmd.PersistentFlags().String("dsn", "", "Database DSN")
	rootCmd.PersistentFlags().String("driver", "", "Database driver")

	rootCmd.AddCommand(cmds.DbCmd)
	rootCmd.AddCommand(cmds.RunCmd)
	rootCmd.AddCommand(cmds.MysqlCmd)

	cmds.InitializeMysqlCmd(queriesFS, helpSystem)
	commands, err := sqleton.LoadSqlCommandsFromEmbedFS(queriesFS, ".", "queries/")
	if err != nil {
		panic(err)
	}
	err = sqleton.AddCommandsToRootCommand(rootCmd, commands)
	if err != nil {
		panic(err)
	}

	initCommands(rootCmd)

}
