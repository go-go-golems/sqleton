package cmds

import (
	"database/sql"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/glazed/pkg/helpers"
	"github.com/wesen/glazed/pkg/middlewares"
	"github.com/wesen/sqliton/pkg"
	"os"

	_ "github.com/go-sql-driver/mysql" // MySQL driver for database/sql
)

var DbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
}

var dbTestConnectionCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the connection to a database",
	Run: func(cmd *cobra.Command, args []string) {
		useDbtProfiles, err := cmd.Flags().GetBool("use-dbt-profiles")
		cobra.CheckErr(err)

		var source *pkg.Source

		if useDbtProfiles {
			dbtProfilesPath, err := cmd.Flags().GetString("dbt-profiles-path")
			cobra.CheckErr(err)

			sources, err := pkg.ParseDbtProfiles(dbtProfilesPath)
			cobra.CheckErr(err)

			sourceName, err := cmd.Flags().GetString("dbt-profile")
			cobra.CheckErr(err)

			for _, s := range sources {
				if s.Name == sourceName {
					source = s
					break
				}
			}

			if source == nil {
				cobra.CheckErr(fmt.Errorf("source not found"))
			}
		} else {
			source, err = setupSource(cmd)
			cobra.CheckErr(err)
		}

		db, err := sql.Open(source.Type, source.ToConnectionString())
		cobra.CheckErr(err)
		defer db.Close()

		err = db.Ping()
		cobra.CheckErr(err)

		fmt.Println("Connection successful")
	},
}

func setupSource(cmd *cobra.Command) (*pkg.Source, error) {
	source := &pkg.Source{}

	var err error
	source.Type, err = cmd.Flags().GetString("type")
	if err != nil {
		return nil, err
	}
	source.Hostname, err = cmd.Flags().GetString("host")
	if err != nil {
		return nil, err
	}
	source.Port, err = cmd.Flags().GetInt("port")
	if err != nil {
		return nil, err
	}
	source.Username, err = cmd.Flags().GetString("user")
	if err != nil {
		return nil, err
	}
	source.Password, err = cmd.Flags().GetString("password")
	if err != nil {
		return nil, err
	}
	source.Database, err = cmd.Flags().GetString("database")
	if err != nil {
		return nil, err
	}

	return source, nil

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
		of.AddTableMiddleware(middlewares.NewReorderColumnOrderMiddleware([]string{"name", "type", "hostname", "port", "database", "schema"}))

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

	DbCmd.AddCommand(dbTestConnectionCmd)
}
