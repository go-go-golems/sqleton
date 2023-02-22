package cmds

import (
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/helpers"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"

	_ "github.com/go-sql-driver/mysql" // MySQL driver for database/sql
)

// From chatGPT:
// To run SQL commands against a PostgreSQL or SQLite database, you can use a similar
// approach, but you will need to use the appropriate driver for the database.
// For example, to use PostgreSQL, you can use the github.com/lib/pq driver, and to use SQLite,
// you can use the github.com/mattn/go-sqlite3

var DbCmd = &cobra.Command{
	Use:   "db",
	Short: "Manage databases",
}

var dbTestConnectionCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the connection to a database",
	Run: func(cmd *cobra.Command, args []string) {
		config := pkg.NewDatabaseConfigFromViper()

		fmt.Printf("Testing connection to %s\n", config.ToString())
		db, err := config.Connect()
		cobra.CheckErr(err)

		cobra.CheckErr(err)
		defer db.Close()

		err = db.Ping()
		cobra.CheckErr(err)

		fmt.Println("Connection successful")
	},
}

var dbLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List databases from profiles",
	Run: func(cmd *cobra.Command, args []string) {
		useDbtProfiles := viper.GetBool("use-dbt-profiles")

		if !useDbtProfiles {
			cmd.PrintErrln("Not using dbt profiles")
			return
		}

		dbtProfilesPath := viper.GetString("dbt-profiles-path")

		sources, err := pkg.ParseDbtProfiles(dbtProfilesPath)
		cobra.CheckErr(err)

		gp, of, err := cli.CreateGlazedProcessorFromCobra(cmd)
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

	err := cli.AddGlazedProcessorFlagsToCobraCommand(dbLsCmd, nil)
	if err != nil {
		panic(err)
	}

	DbCmd.AddCommand(dbTestConnectionCmd)
}
