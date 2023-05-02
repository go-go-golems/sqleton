package cmds

import (
	"encoding/json"
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

// dbTestConnectionCmdWithPrefix is a test command to use
// configuration flags and settings with a prefix, which can be used to
// mix sqleton commands with say, escuse-me commands
var dbPrintEvidenceSettingsCmd = &cobra.Command{
	Use:   "print-evidence-settings",
	Short: "Output the settings to connect to a database for evidence.dev",
	Run: func(cmd *cobra.Command, args []string) {
		config := createConfigFromCobra(cmd)
		source, err := config.GetSource()
		cobra.CheckErr(err)

		gitRepo, _ := cmd.Flags().GetString("git-repo")

		type EvidenceCredentials struct {
			Host     string `json:"host"`
			Database string `json:"database"`
			User     string `json:"user"`
			Password string `json:"password"`
			Port     string `json:"port"`
		}

		type EvidenceSettings struct {
			GitRepo     string              `json:"gitRepo,omitempty"`
			Database    string              `json:"database"`
			Credentials EvidenceCredentials `json:"credentials"`
		}

		credentials := EvidenceCredentials{
			Host:     source.Hostname,
			Database: source.Database,
			User:     source.Username,
			Password: source.Password,
			Port:     fmt.Sprintf("%d", source.Port),
		}

		settings := EvidenceSettings{
			GitRepo:     gitRepo,
			Database:    source.Type,
			Credentials: credentials,
		}

		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		err = encoder.Encode(settings)
		cobra.CheckErr(err)
	},
}

var dbLsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List databases from profiles",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

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
			err := gp.ProcessInputObject(ctx, sourceObj)
			cobra.CheckErr(err)
		}

		err = gp.OutputFormatter().Output(ctx, os.Stdout)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error rendering output: %s\n", err)
			os.Exit(1)
		}
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

	err = connectionLayer.AddFlagsToCobraCommand(dbPrintEvidenceSettingsCmd)
	cobra.CheckErr(err)
	dbPrintEvidenceSettingsCmd.Flags().String("git-repo", "", "Git repo to use for evidence.dev")
	DbCmd.AddCommand(dbPrintEvidenceSettingsCmd)

	connectionLayer, err = pkg.NewSqlConnectionParameterLayer(
		layers.WithPrefix("test-"),
	)
	cobra.CheckErr(err)
	err = connectionLayer.AddFlagsToCobraCommand(dbTestConnectionCmdWithPrefix)
	cobra.CheckErr(err)
	DbCmd.AddCommand(dbTestConnectionCmdWithPrefix)
}
