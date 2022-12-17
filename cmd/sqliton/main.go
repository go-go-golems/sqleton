package main

import (
	"embed"
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/help"
)

var rootCmd = &cobra.Command{
	Use:   "sqliton",
	Short: "sqliton runs SQL queries out of template files",
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
}
