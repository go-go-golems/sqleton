package pkg

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wesen/glazed/pkg/cli"
)

func OpenDatabaseFromViper() (*sqlx.DB, error) {
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

	var source *Source

	if useDbtProfiles {

		sources, err := ParseDbtProfiles(dbtProfilesPath)
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
		source = &Source{
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

type SqletonCommandDescription struct {
	Name  string
	Short string
	Long  string
}

type SqletonCommand interface {
	RunQueryIntoGlaze(ctx context.Context, db *sqlx.DB, gp *cli.GlazeProcessor) error
	Description() SqletonCommandDescription
}

func ToCobraCommand(s SqletonCommand) (*cobra.Command, error) {
	description := s.Description()
	cmd := &cobra.Command{
		Use:   description.Name,
		Short: description.Short,
		Long:  description.Long,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := OpenDatabaseFromViper()
			if err != nil {
				return errors.Wrapf(err, "Could not open database")
			}

			dbContext := context.Background()
			err = db.PingContext(dbContext)
			if err != nil {
				return errors.Wrapf(err, "Could not ping database")
			}

			gp, of, err := cli.SetupProcessor(cmd)
			if err != nil {
				return errors.Wrapf(err, "Could not setup processor")
			}

			err = s.RunQueryIntoGlaze(dbContext, db, gp)
			if err != nil {
				return errors.Wrapf(err, "Could not run query")
			}

			s, err := of.Output()
			if err != nil {
				return errors.Wrapf(err, "Could not get output")
			}
			fmt.Print(s)

			return nil
		},
	}

	cli.AddOutputFlags(cmd)
	cli.AddTemplateFlags(cmd)
	cli.AddFieldsFilterFlags(cmd, "")
	cli.AddSelectFlags(cmd)

	return cmd, nil
}

// SqlCommand describes a command line command that runs a query
type SqlCommand struct {
	CommandDescription SqletonCommandDescription
	Query              string
}

func (s *SqlCommand) RunQueryIntoGlaze(ctx context.Context, db *sqlx.DB, gp *cli.GlazeProcessor) error {
	return RunQueryIntoGlaze(ctx, db, s.Query, gp)
}

func (s *SqlCommand) Description() SqletonCommandDescription {
	return s.CommandDescription
}
