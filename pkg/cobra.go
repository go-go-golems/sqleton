package pkg

import (
	"context"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tj/go-naturaldate"
	"github.com/wesen/glazed/pkg/cli"
	"strings"
	"time"
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

func ToCobraCommand(s SqletonCommand) (*cobra.Command, error) {
	description := s.Description()
	cmd := &cobra.Command{
		Use:   description.Name,
		Short: description.Short,
		Long:  description.Long,
		RunE: func(cmd *cobra.Command, args []string) error {

			parameters := map[string]interface{}{}

			for _, parameter := range description.Parameters {
				switch parameter.Type {
				case ParameterTypeString:
					fallthrough
				case ParameterTypeChoice:
					v, err := cmd.Flags().GetString(parameter.Name)
					if err != nil {
						return err
					}
					parameters[parameter.Name] = v

				case ParameterTypeInteger:
					v, err := cmd.Flags().GetInt(parameter.Name)
					if err != nil {
						return err
					}
					parameters[parameter.Name] = v

				case ParameterTypeDate:
					v, err := cmd.Flags().GetString(parameter.Name)
					if err != nil {
						return err
					}
					parsedDate, err := dateparse.ParseAny(v)
					if err != nil {
						parsedDate, err = naturaldate.Parse(v, time.Now())
						if err != nil {
							return errors.Wrapf(err, "Could not parse date %s", v)
						}
					}
					parameters[parameter.Name] = parsedDate

				case ParameterTypeBool:
					v, err := cmd.Flags().GetBool(parameter.Name)
					if err != nil {
						return err
					}
					parameters[parameter.Name] = v

				case ParameterTypeStringList:
					v, err := cmd.Flags().GetStringSlice(parameter.Name)
					if err != nil {
						return err
					}
					parameters[parameter.Name] = v

				case ParameterTypeIntegerList:
					v, err := cmd.Flags().GetIntSlice(parameter.Name)
					if err != nil {
						return err
					}
					parameters[parameter.Name] = v
				}
			}

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

			printQuery, _ := cmd.Flags().GetBool("print-query")
			if printQuery {
				query, err := s.RenderQuery(parameters)
				if err != nil {
					return errors.Wrapf(err, "Could not generate query")
				}
				fmt.Println(query)
				return nil
			}

			// TODO(2022-12-21, manuel): Add explain functionality
			// See: https://github.com/wesen/sqleton/issues/45
			explain, _ := cmd.Flags().GetBool("explain")
			_ = explain

			err = s.RunQueryIntoGlaze(dbContext, db, parameters, gp)
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

	// TODO(2022-12-20, manuel): we should be able to load these parameters from a config file
	// See: https://github.com/wesen/sqleton/issues/39
	for _, parameter := range description.Parameters {
		flagName := parameter.Name
		shortFlag := parameter.ShortFlag
		switch parameter.Type {
		case ParameterTypeString:
			defaultValue, ok := parameter.Default.(string)
			if !ok {
				return nil, errors.Errorf("Default value for parameter %s is not a string: %v", parameter.Name, parameter.Default)
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().StringP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().String(flagName, defaultValue, parameter.Help)
			}
		case ParameterTypeInteger:
			defaultValue, ok := parameter.Default.(int)
			if !ok {
				return nil, errors.Errorf("Default value for parameter %s is not an integer: %v", parameter.Name, parameter.Default)
			}
			if parameter.ShortFlag != "" {
				cmd.Flags().IntP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().Int(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeBool:
			defaultValue, ok := parameter.Default.(bool)
			if !ok {
				return nil, errors.Errorf("Default value for parameter %s is not a bool: %v", parameter.Name, parameter.Default)
			}
			if parameter.ShortFlag != "" {
				cmd.Flags().BoolP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().Bool(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeDate:
			defaultValue, ok := parameter.Default.(string)
			if !ok {
				return nil, errors.Errorf("Default value for parameter %s is not a string: %v", parameter.Name, parameter.Default)
			}

			_, err := dateparse.ParseAny(defaultValue)
			if err != nil {
				_, err = naturaldate.Parse(defaultValue, time.Now())
				if err != nil {
					return nil, errors.Wrapf(err, "Could not parse default value for parameter %s: %s", parameter.Name, defaultValue)
				}
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().StringP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().String(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeStringList:
			defaultValue, ok := parameter.Default.([]interface{})
			if !ok {
				return nil, errors.Errorf("Default value for parameter %s is not a string list: %v", parameter.Name, parameter.Default)
			}

			// convert to string list
			stringList, err := convertToStringList(defaultValue)
			if err != nil {
				return nil, errors.Wrapf(err, "Could not convert default value for parameter %s to string list: %v", parameter.Name, parameter.Default)
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().StringSliceP(flagName, shortFlag, stringList, parameter.Help)
			} else {
				cmd.Flags().StringSlice(flagName, stringList, parameter.Help)
			}

		case ParameterTypeIntegerList:
			defaultValue, ok := parameter.Default.([]int)
			if !ok {
				return nil, errors.Errorf("Default value for parameter %s is not an integer list: %v", parameter.Name, parameter.Default)
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().IntSliceP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().IntSlice(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeChoice:
			defaultValue, ok := parameter.Default.(string)
			if !ok {
				return nil, errors.Errorf("Default value for parameter %s is not a string: %v", parameter.Name, parameter.Default)
			}

			choiceString := strings.Join(parameter.Choices, ",")

			if parameter.ShortFlag != "" {
				cmd.Flags().StringP(flagName, shortFlag, defaultValue, fmt.Sprintf("%s (%s)", parameter.Help, choiceString))
			} else {
				cmd.Flags().String(flagName, defaultValue, fmt.Sprintf("%s (%s)", parameter.Help, choiceString))
			}
		}
	}

	cmd.Flags().Bool("print-query", false, "Print the query that will be executed")
	cmd.Flags().Bool("explain", false, "Print the query plan that will be executed")

	cli.AddOutputFlags(cmd)
	cli.AddTemplateFlags(cmd)
	cli.AddFieldsFilterFlags(cmd, "")
	cli.AddSelectFlags(cmd)

	return cmd, nil
}

func convertToStringList(value []interface{}) ([]string, error) {
	stringList := make([]string, len(value))
	for i, v := range value {
		s, ok := v.(string)
		if !ok {
			return nil, errors.Errorf("Not a string: %v", v)
		}
		stringList[i] = s
	}
	return stringList, nil
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
