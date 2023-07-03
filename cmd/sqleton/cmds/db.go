package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/jmoiron/sqlx"
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
		defer func(db *sqlx.DB) {
			_ = db.Close()
		}(db)

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
		defer func(db *sqlx.DB) {
			_ = db.Close()
		}(db)

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

var dbPrintEnvCmd = &cobra.Command{
	Use:   "print-env",
	Short: "Output the settings to connect to a database as environment variables",
	Run: func(cmd *cobra.Command, args []string) {
		config := createConfigFromCobra(cmd)
		source, err := config.GetSource()
		cobra.CheckErr(err)

		isEnvRc, _ := cmd.Flags().GetBool("envrc")
		envPrefix, _ := cmd.Flags().GetString("env-prefix")

		prefix := ""
		if isEnvRc {
			prefix = "export "
		}
		prefix = prefix + envPrefix
		fmt.Printf("%s%s=%s\n", prefix, "TYPE", source.Type)
		fmt.Printf("%s%s=%s\n", prefix, "HOST", source.Hostname)
		fmt.Printf("%s%s=%s\n", prefix, "PORT", fmt.Sprintf("%d", source.Port))
		fmt.Printf("%s%s=%s\n", prefix, "DATABASE", source.Database)
		fmt.Printf("%s%s=%s\n", prefix, "USER", source.Username)
		fmt.Printf("%s%s=%s\n", prefix, "PASSWORD", source.Password)
		fmt.Printf("%s%s=%s\n", prefix, "SCHEMA", source.Schema)
		if config.UseDbtProfiles {
			fmt.Printf("%s%s=1\n", prefix, "USE_DBT_PROFILES")
		} else {
			fmt.Printf("%s%s=\n", prefix, "USE_DBT_PROFILES")
		}
		fmt.Printf("%s%s=%s\n", prefix, "DBT_PROFILES_PATH", config.DbtProfilesPath)
		fmt.Printf("%s%s=%s\n", prefix, "DBT_PROFILE", config.DbtProfile)
	},
}

var dbPrintSettingsCmd = &cobra.Command{
	Use:   "print-settings",
	Short: "Output the settings to connect to a database using glazed",
	Run: func(cmd *cobra.Command, args []string) {
		config := createConfigFromCobra(cmd)
		source, err := config.GetSource()
		cobra.CheckErr(err)

		gp, _, err := cli.CreateGlazedProcessorFromCobra(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Could not create glaze  procersors: %v\n", err)
			os.Exit(1)
		}

		individualRows, _ := cmd.Flags().GetBool("individual-rows")
		useSqletonEnvNames, _ := cmd.Flags().GetBool("use-env-names")
		withEnvPrefix, _ := cmd.Flags().GetString("with-env-prefix")

		ctx := cmd.Context()

		host := "host"
		port := "port"
		database := "database"
		user := "user"
		password := "password"
		type_ := "type"
		schema := "schema"
		dbtProfile := "dbtProfile"
		useDbtProfiles := "useDbtProfiles"
		dbtProfilesPath := "dbtProfilesPath"

		if useSqletonEnvNames {
			host = "SQLETON_HOST"
			port = "SQLETON_PORT"
			database = "SQLETON_DATABASE"
			user = "SQLETON_USER"
			password = "SQLETON_PASSWORD"
			type_ = "SQLETON_TYPE"
			schema = "SQLETON_SCHEMA"
			dbtProfile = "SQLETON_DBT_PROFILE"
			useDbtProfiles = "SQLETON_USE_DBT_PROFILES"
			dbtProfilesPath = "SQLETON_DBT_PROFILES_PATH"
		} else if withEnvPrefix != "" {
			host = fmt.Sprintf("%sHOST", withEnvPrefix)
			port = fmt.Sprintf("%sPORT", withEnvPrefix)
			database = fmt.Sprintf("%sDATABASE", withEnvPrefix)
			user = fmt.Sprintf("%sUSER", withEnvPrefix)
			password = fmt.Sprintf("%sPASSWORD", withEnvPrefix)
			type_ = fmt.Sprintf("%sTYPE", withEnvPrefix)
			schema = fmt.Sprintf("%sSCHEMA", withEnvPrefix)
			dbtProfile = fmt.Sprintf("%sDBT_PROFILE", withEnvPrefix)
			useDbtProfiles = fmt.Sprintf("%sUSE_DBT_PROFILES", withEnvPrefix)
			dbtProfilesPath = fmt.Sprintf("%sDBT_PROFILES_PATH", withEnvPrefix)
		}

		addRow := func(name string, value interface{}) {
			_ = gp.AddRow(ctx, types.NewRow(
				types.MRP("name", name),
				types.MRP("value", value),
			))
		}
		if individualRows {
			addRow(host, source.Hostname)
			addRow(port, source.Port)
			addRow(database, source.Database)
			addRow(user, source.Username)
			addRow(password, source.Password)
			addRow(type_, source.Type)
			addRow(schema, source.Schema)
			addRow(dbtProfile, config.DbtProfile)
			addRow(useDbtProfiles, config.UseDbtProfiles)
			addRow(dbtProfilesPath, config.DbtProfilesPath)
		} else {
			_ = gp.AddRow(ctx, types.NewRow(
				types.MRP(host, source.Hostname),
				types.MRP(port, source.Port),
				types.MRP(database, source.Database),
				types.MRP(user, source.Username),
				types.MRP(password, source.Password),
				types.MRP(type_, source.Type),
				types.MRP(schema, source.Schema),
				types.MRP(dbtProfile, config.DbtProfile),
				types.MRP(useDbtProfiles, config.UseDbtProfiles),
				types.MRP(dbtProfilesPath, config.DbtProfilesPath),
			))
		}

		err = gp.Close(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error rendering output: %s\n", err)
			os.Exit(1)
		}
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

		gp, _, err := cli.CreateGlazedProcessorFromCobra(cmd)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Could not create glaze  procersors: %v\n", err)
			os.Exit(1)
		}

		// don't output the password
		gp.AddRowMiddleware(row.NewFieldsFilterMiddleware([]string{}, []string{"password"}))
		gp.AddRowMiddleware(row.NewReorderColumnOrderMiddleware([]string{"name", "type", "hostname", "port", "database", "schema"}))

		for _, source := range sources {
			row := types.NewRowFromStruct(source, true)
			err := gp.AddRow(ctx, row)
			cobra.CheckErr(err)
		}

		err = gp.Close(ctx)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error rendering output: %s\n", err)
			os.Exit(1)
		}
		cobra.CheckErr(err)
	},
}

func init() {
	err := cli.AddGlazedProcessorFlagsToCobraCommand(dbLsCmd)
	cobra.CheckErr(err)
	DbCmd.AddCommand(dbLsCmd)

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

	err = connectionLayer.AddFlagsToCobraCommand(dbPrintEnvCmd)
	cobra.CheckErr(err)
	dbPrintEnvCmd.Flags().Bool("envrc", false, "Output as an .envrc file")
	dbPrintEnvCmd.Flags().String("env-prefix", "SQLETON_", "Prefix for environment variables")
	DbCmd.AddCommand(dbPrintEnvCmd)

	err = connectionLayer.AddFlagsToCobraCommand(dbPrintSettingsCmd)
	cobra.CheckErr(err)
	dbPrintSettingsCmd.Flags().Bool("individual-rows", false, "Output as individual rows")
	dbPrintSettingsCmd.Flags().String("with-env-prefix", "", "Output as environment variables with a prefix")
	dbPrintSettingsCmd.Flags().Bool("use-env-names", false, "Output as SQLETON_ environment variables with a prefix")
	err = cli.AddGlazedProcessorFlagsToCobraCommand(dbPrintSettingsCmd)
	cobra.CheckErr(err)
	DbCmd.AddCommand(dbPrintSettingsCmd)

	connectionLayer, err = pkg.NewSqlConnectionParameterLayer(
		layers.WithPrefix("test-"),
	)
	cobra.CheckErr(err)
	err = connectionLayer.AddFlagsToCobraCommand(dbTestConnectionCmdWithPrefix)
	cobra.CheckErr(err)
	DbCmd.AddCommand(dbTestConnectionCmdWithPrefix)
}
