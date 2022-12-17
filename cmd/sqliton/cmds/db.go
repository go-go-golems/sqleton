package cmds

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/glazed/pkg/helpers"
	"github.com/wesen/glazed/pkg/middlewares"
	"github.com/wesen/sqliton/pkg"
	"os"
)

var DbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
}

var dbLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List databases from profiles",
	Run: func(cmd *cobra.Command, args []string) {
		useDbtProfiles, err := cmd.Flags().GetBool("use-dbt-profiles")
		cobra.CheckErr(err)

		if !useDbtProfiles {
			cmd.PrintErrln("Not using dbt profiles")
			return
		}

		dbtProfilesPath, err := cmd.Flags().GetString("dbt-profiles-path")
		cobra.CheckErr(err)

		sources, err := pkg.ParseDbtProfiles(dbtProfilesPath)
		cobra.CheckErr(err)

		gp, of, err := cli.SetupProcessor(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Could not create glaze  procersors: %v\n", err)
			os.Exit(1)
		}

		// don't output the password
		of.AddTableMiddleware(middlewares.NewFieldsFilterMiddleware([]string{}, []string{"password"}))

		for _, source := range sources {
			sourceObj := helpers.StructToMap(source, true)
			err := gp.ProcessInputObject(sourceObj)
			cobra.CheckErr(err)
		}

		s, err := of.Output()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error rendering output: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(s)
	},
}

func init() {
	DbCmd.AddCommand(dbLsCmd)

	cli.AddOutputFlags(dbLsCmd)
	cli.AddTemplateFlags(dbLsCmd)
	cli.AddFieldsFilterFlags(dbLsCmd, "")
	cli.AddSelectFlags(dbLsCmd)
}
