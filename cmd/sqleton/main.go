package main

import (
	"embed"
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/help"
	"github.com/wesen/sqleton/cmd/sqleton/cmds"
)

var rootCmd = &cobra.Command{
	Use:   "sqleton",
	Short: "sqleton runs SQL queries out of template files",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// TODO(2022-12-18) This is where we would add the code to load flags from the environment,
		// and default to dbt profiles if none is set
		// https://github.com/wesen/sqleton/issues/18

		// from ChatGPT
		//
		// Load the variables from the environment
		//			viper.AutomaticEnv()
		//			host = viper.GetString("HOST")
		//			database = viper.GetString("DATABASE")
		//			user = viper.GetString("USER")
		//			password = viper.GetString("PASSWORD")
		//			port = viper.GetInt("PORT")
		//			schema = viper.GetString("SCHEMA")
		//			connectionType = viper.GetString("TYPE")
		//			dsn = viper.GetString("DSN")
		//			driver = viper.GetString("DRIVER")

		// Bind the variables to the command-line flags
		//viper.BindPFlag("host", cmd.Flags().Lookup("host"))
		//viper.BindPFlag("database", cmd.Flags().Lookup("database"))
		//viper.BindPFlag("user", cmd.Flags().Lookup("user"))
		//viper.BindPFlag("password", cmd.Flags().Lookup("password"))
		//viper.BindPFlag("port", cmd.Flags().Lookup("port"))
		//viper.BindPFlag("schema", cmd.Flags().Lookup("schema"))
		//viper.BindPFlag("type", cmd.Flags().Lookup("type"))
		//viper.BindPFlag("dsn", cmd.Flags().Lookup("dsn"))
		//viper.BindPFlag("driver", cmd.Flags().Lookup("driver"))

		// Bind the variables to the command-line flags
		// viper.BindPFlags(cmd.Flags())
	},
}

func main() {
	_ = rootCmd.Execute()
}

//go:embed doc/*
var docFS embed.FS

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

	// dsn and driver
	rootCmd.PersistentFlags().String("dsn", "", "Database DSN")
	rootCmd.PersistentFlags().String("driver", "", "Database driver")

	rootCmd.AddCommand(cmds.DbCmd)
	rootCmd.AddCommand(cmds.RunCmd)
}
