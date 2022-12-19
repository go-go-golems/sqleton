package pkg

import (
	"context"
	"embed"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/wesen/glazed/pkg/cli"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	Name    string `yaml:"name"`
	Short   string `yaml:"short"`
	Long    string `yaml:"long"`
	Parents []string
	Query   string `yaml:"query"`
}

func (s *SqlCommand) RunQueryIntoGlaze(ctx context.Context, db *sqlx.DB, gp *cli.GlazeProcessor) error {
	return RunQueryIntoGlaze(ctx, db, s.Query, gp)
}

func (s *SqlCommand) Description() SqletonCommandDescription {
	return SqletonCommandDescription{
		Name:  s.Name,
		Short: s.Short,
		Long:  s.Long,
	}
}

func LoadSqlCommandFromYaml(s io.Reader) (*SqlCommand, error) {
	var sq SqlCommand
	err := yaml.NewDecoder(s).Decode(&sq)
	if err != nil {
		return nil, err
	}

	return &sq, nil
}

func LoadSqlCommandsFromEmbedFS(f embed.FS, dir string, cmdRoot string) ([]*SqlCommand, error) {
	var commands []*SqlCommand

	entries, err := f.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		fileName := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			subCommands, err := LoadSqlCommandsFromEmbedFS(f, fileName, cmdRoot)
			if err != nil {
				return nil, err
			}
			commands = append(commands, subCommands...)
		} else {
			if strings.HasSuffix(entry.Name(), ".yml") ||
				strings.HasSuffix(entry.Name(), ".yaml") {
				command, err := func() (*SqlCommand, error) {
					file, err := f.Open(fileName)
					if err != nil {
						return nil, errors.Wrapf(err, "Could not open file %s", fileName)
					}
					defer func() {
						_ = file.Close()
					}()

					command, err := LoadSqlCommandFromYaml(file)
					if err != nil {
						return nil, errors.Wrapf(err, "Could not load command from file %s", fileName)
					}

					pathToFile := strings.TrimPrefix(dir, cmdRoot)
					command.Parents = strings.Split(pathToFile, "/")

					return command, err
				}()
				if err != nil {
					return nil, err
				}

				commands = append(commands, command)
			}

		}
	}

	return commands, nil
}

func LoadSqlCommandsFromDirectory(dir string, cmdRoot string) ([]*SqlCommand, error) {
	var commands []*SqlCommand

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		fileName := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			subCommands, err := LoadSqlCommandsFromDirectory(fileName, cmdRoot)
			if err != nil {
				return nil, err
			}
			commands = append(commands, subCommands...)
		} else {
			if strings.HasSuffix(entry.Name(), ".yml") ||
				strings.HasSuffix(entry.Name(), ".yaml") {
				command, err := func() (*SqlCommand, error) {
					file, err := os.Open(fileName)
					if err != nil {
						return nil, errors.Wrapf(err, "Could not open file %s", fileName)
					}
					defer func() {
						_ = file.Close()
					}()

					command, err := LoadSqlCommandFromYaml(file)
					if err != nil {
						return nil, errors.Wrapf(err, "Could not load command from file %s", fileName)
					}

					pathToFile := strings.TrimPrefix(dir, cmdRoot)
					pathToFile = strings.TrimPrefix(pathToFile, "/")
					command.Parents = strings.Split(pathToFile, "/")

					return command, err
				}()
				if err != nil {
					return nil, err
				}

				commands = append(commands, command)
			}
		}
	}

	return commands, nil
}

// AddCommandsToRootCommand
func AddCommandsToRootCommand(rootCmd *cobra.Command, commands []*SqlCommand) error {
	for _, command := range commands {
		// find the proper subcommand, or create if it doesn't exist
		parentCmd := rootCmd
		for _, parent := range command.Parents {
			subCmd, _, _ := parentCmd.Find([]string{parent})
			if subCmd == nil || subCmd == rootCmd {
				// TODO(2022-12-19) Load documentation for subcommands from a readme file
				// See https://github.com/wesen/sqleton/issues/34
				parentCmd = &cobra.Command{
					Use:   parent,
					Short: fmt.Sprintf("All commands for %s", parent),
				}
				rootCmd.AddCommand(parentCmd)
			} else {
				parentCmd = subCmd
			}
		}
		cobraCommand, err := ToCobraCommand(command)
		if err != nil {
			return err
		}
		parentCmd.AddCommand(cobraCommand)
	}

	return nil
}
