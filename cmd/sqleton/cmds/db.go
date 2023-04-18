package cmds

import (
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/helpers/maps"
	"github.com/go-go-golems/glazed/pkg/middlewares/table"
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

func createConfigFromCobra(cmd *cobra.Command) *pkg.DatabaseConfig {
	connectionLayer, err := pkg.NewSqlConnectionParameterLayer()
	cobra.CheckErr(err)

	ps, err := connectionLayer.ParseFlagsFromCobraCommand(cmd)
	cobra.CheckErr(err)

	dbtLayer, err := pkg.NewDbtParameterLayer()
	cobra.CheckErr(err)

	ps2, err := dbtLayer.ParseFlagsFromCobraCommand(cmd)
	cobra.CheckErr(err)

	parsedLayers := map[string]*layers.ParsedParameterLayer{
		"sqleton-connection": {
			Layer:      connectionLayer,
			Parameters: ps,
		},
		"dbt": {
			Layer:      dbtLayer,
			Parameters: ps2,
		},
	}

	config, err := pkg.NewConfigFromParsedLayers(parsedLayers)
	cobra.CheckErr(err)

	return config
}

var dbTestConnectionCmd = &cobra.Command{
	Use:   "test",
	Short: "Test the connection to a database",
	Run: func(cmd *cobra.Command, args []string) {
		config := createConfigFromCobra(cmd)

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

// dbTestConnectionCmdWithPrefix is a test command to use
// configuration flags and settings with a prefix, which can be used to
// mix sqleton commands with say, escuse-me commands
var dbTestConnectionCmdWithPrefix = &cobra.Command{
	Use:   "test-prefix",
	Short: "Test the connection to a database, but all sqleton flags have the test- prefix",
	Run: func(cmd *cobra.Command, args []string) {
		config := createConfigFromCobra(cmd)
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

		gp, err := cli.CreateGlazedProcessorFromCobra(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Could not create glaze  procersors: %v\n", err)
			os.Exit(1)
		}

		// don't output the password
		gp.OutputFormatter().AddTableMiddleware(table.NewFieldsFilterMiddleware([]string{}, []string{"password"}))
		gp.OutputFormatter().AddTableMiddleware(table.NewReorderColumnOrderMiddleware([]string{"name", "type", "hostname", "port", "database", "schema"}))

		for _, source := range sources {
			sourceObj := maps.StructToMap(source, true)
			err := gp.ProcessInputObject(sourceObj)
			cobra.CheckErr(err)
		}

		s, err := gp.OutputFormatter().Output()
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error rendering output: %s\n", err)
			os.Exit(1)
		}
		fmt.Print(s)
	},
}

func init() {
	DbCmd.AddCommand(dbLsCmd)

	err := cli.AddGlazedProcessorFlagsToCobraCommand(dbLsCmd)
	if err != nil {
		panic(err)
	}

	connectionLayer, err := pkg.NewSqlConnectionParameterLayer()
	cobra.CheckErr(err)
	dbtParameterLayer, err := pkg.NewDbtParameterLayer()
	cobra.CheckErr(err)

	err = connectionLayer.AddFlagsToCobraCommand(dbTestConnectionCmd)
	cobra.CheckErr(err)
	DbCmd.AddCommand(dbTestConnectionCmd)

	err = dbtParameterLayer.AddFlagsToCobraCommand(dbTestConnectionCmd)
	cobra.CheckErr(err)

	connectionLayer, err = pkg.NewSqlConnectionParameterLayer(
		layers.WithPrefix("test-"),
	)
	cobra.CheckErr(err)
	err = connectionLayer.AddFlagsToCobraCommand(dbTestConnectionCmdWithPrefix)
	cobra.CheckErr(err)
	DbCmd.AddCommand(dbTestConnectionCmdWithPrefix)
}
