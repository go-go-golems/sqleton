package cmds

import (
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wesen/sqleton/pkg"
)

func openDatabase(cmd *cobra.Command) (*sqlx.DB, error) {
	// Load the configuration values from the configuration file
	host := viper.GetString("host")
	database := viper.GetString("database")
	user := viper.GetString("user")
	password := viper.GetString("password")
	port := viper.GetInt("port")
	schema := viper.GetString("schema")
	connectionType := viper.GetString("type")
	dsn := viper.GetString("dsn")
	driver := viper.GetString("driver")
	useDbtProfiles := viper.GetBool("use-dbt-profiles")
	dbtProfilesPath := viper.GetString("dbt-profiles-path")
	dbtProfile := viper.GetString("dbt-profile")

	// TODO(2022-12-18, manuel) This is where we would add support for DSN/Driver loading
	// See https://github.com/wesen/sqleton/issues/21
	_ = dsn
	_ = driver

	var source *pkg.Source

	if useDbtProfiles {

		sources, err := pkg.ParseDbtProfiles(dbtProfilesPath)
		if err != nil {
			return nil, err
		}

		for _, s := range sources {
			if s.Name == dbtProfile {
				source = s
				break
			}
		}

		if source == nil {
			return nil, errors.Errorf("Source %s not found", dbtProfile)
		}
	} else {
		source = &pkg.Source{
			Type:     connectionType,
			Hostname: host,
			Port:     port,
			Username: user,
			Password: password,
			Database: database,
			Schema:   schema,
		}
	}

	db, err := sqlx.Connect(source.Type, source.ToConnectionString())

	// TODO(2022-12-18, manuel): this is where we would add support for a ro connection
	// https://github.com/wesen/sqleton/issues/24

	return db, err
}
