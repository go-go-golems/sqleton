package pkg

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

type DatabaseConfig struct {
	Host            string
	Database        string
	User            string
	Password        string
	Port            int
	Schema          string
	Type            string
	DSN             string
	Driver          string
	DbtProfilesPath string
	DbtProfile      string
	UseDbtProfiles  bool
}

func NewDatabaseConfigFromViper() *DatabaseConfig {
	return &DatabaseConfig{
		Host:            viper.GetString("host"),
		Database:        viper.GetString("database"),
		User:            viper.GetString("user"),
		Password:        viper.GetString("password"),
		Port:            viper.GetInt("port"),
		Schema:          viper.GetString("schema"),
		Type:            viper.GetString("type"),
		DSN:             viper.GetString("dsn"),
		Driver:          viper.GetString("driver"),
		DbtProfilesPath: viper.GetString("dbt-profiles-path"),
		DbtProfile:      viper.GetString("dbt-profile"),
		UseDbtProfiles:  viper.GetBool("use-dbt-profiles"),
	}
}

// LogVerbose just outputs information about the database config to the
// debug logging level.
func (c *DatabaseConfig) LogVerbose() {
	if c.UseDbtProfiles {
		log.Debug().
			Str("dbt-profiles-path", c.DbtProfilesPath).
			Str("dbt-profile", c.DbtProfile).
			Msg("Using dbt profiles")
	} else if c.DSN != "" {
		log.Debug().
			Str("dsn", c.DSN).
			Str("driver", c.Driver).
			Msg("Using DSN")
	} else {
		log.Debug().
			Str("host", c.Host).
			Str("database", c.Database).
			Str("user", c.User).
			Int("port", c.Port).
			Str("schema", c.Schema).
			Str("type", c.Type).
			Msg("Using connection string")
	}
}

func (c *DatabaseConfig) ToString() string {
	if c.UseDbtProfiles {
		s, err := c.GetSource()
		if err != nil {
			return fmt.Sprintf("Error: %s", err)
		}
		sourceString := fmt.Sprintf("%s@%s:%d/%s", s.Username, s.Hostname, s.Port, s.Database)

		if c.DbtProfilesPath != "" {
			return fmt.Sprintf("dbt-profiles-path: %s, dbt-profile: %s, %s", c.DbtProfilesPath, c.DbtProfile, sourceString)
		} else {
			return fmt.Sprintf("dbt-profile: %s, %s", c.DbtProfile, sourceString)
		}
	} else if c.DSN != "" {
		return fmt.Sprintf("dsn: %s, driver: %s", c.DSN, c.Driver)
	} else {
		return fmt.Sprintf("%s@%s:%d/%s", c.User, c.Host, c.Port, c.Database)
	}
}

func (c *DatabaseConfig) GetSource() (*Source, error) {
	// TODO(2022-12-18, manuel) This is where we would add support for DSN/Driver loading
	// See https://github.com/wesen/sqleton/issues/21
	_ = c.DSN
	_ = c.Driver

	var source *Source

	if c.UseDbtProfiles {
		sources, err := ParseDbtProfiles(c.DbtProfilesPath)
		if err != nil {
			return nil, err
		}

		for _, s := range sources {
			if s.Name == c.DbtProfile {
				source = s
				break
			}
		}

		if source == nil {
			return nil, errors.Errorf("Source %s not found", c.DbtProfile)
		}
	} else {
		source = &Source{
			Type:     c.Type,
			Hostname: c.Host,
			Port:     c.Port,
			Username: c.User,
			Password: c.Password,
			Database: c.Database,
			Schema:   c.Schema,
		}
	}

	return source, nil
}

func (c *DatabaseConfig) Connect() (*sqlx.DB, error) {
	c.LogVerbose()

	s, err := c.GetSource()
	if err != nil {
		return nil, err
	}
	db, err := sqlx.Connect(s.Type, s.ToConnectionString())

	// TODO(2022-12-18, manuel): this is where we would add support for a ro connection
	// https://github.com/wesen/sqleton/issues/24

	return db, err
}

func OpenDatabaseFromViper() (*sqlx.DB, error) {
	config := NewDatabaseConfigFromViper()
	return config.Connect()
}
