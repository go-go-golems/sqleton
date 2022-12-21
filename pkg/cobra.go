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

// TODO(2022-12-21, manuel): Additional parameter/argument ideas
// - number range
// - handle floats
// - generate choices/ranges from SQL statements

// refTime is used to set a reference time for natural date parsing for unit test purposes
var refTime *time.Time

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
			// TODO(2022-12-20, manuel): we should be able to load default values for these parameters from a config file
			// See: https://github.com/wesen/sqleton/issues/39
			parameters, err := gatherFlags(cmd, description.Flags)
			if err != nil {
				return err
			}

			arguments, err := gatherArguments(args, description.Arguments)
			if err != nil {
				return err
			}

			// merge parameters and arguments
			// arguments take precedence over parameters
			for k, v := range arguments {
				parameters[k] = v
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

	err := addFlags(cmd, &description)
	if err != nil {
		return nil, err
	}

	err = addArguments(cmd, &description)
	if err != nil {
		return nil, err
	}

	cmd.Flags().Bool("print-query", false, "Print the query that will be executed")
	cmd.Flags().Bool("explain", false, "Print the query plan that will be executed")

	cli.AddOutputFlags(cmd)
	cli.AddTemplateFlags(cmd)
	cli.AddFieldsFilterFlags(cmd, "")
	cli.AddSelectFlags(cmd)

	return cmd, nil
}

func addArguments(cmd *cobra.Command, description *SqletonCommandDescription) error {
	minArgs := 0
	// -1 signifies unbounded
	maxArgs := 0
	hadOptional := false

	for _, argument := range description.Arguments {
		if maxArgs == -1 {
			// already handling unbounded arguments
			return errors.Errorf("Cannot handle more than one unbounded argument, but found %s", argument.Name)
		}
		err := argument.CheckParameterDefaultValueValidity()
		if err != nil {
			return errors.Wrapf(err, "Invalid default value for argument %s", argument.Name)
		}

		if argument.Required {
			if hadOptional {
				return errors.Errorf("Cannot handle required argument %s after optional argument", argument.Name)
			}
			minArgs++
		} else {
			hadOptional = true
		}
		maxArgs++
		switch argument.Type {
		case ParameterTypeStringList:
			fallthrough
		case ParameterTypeIntegerList:
			maxArgs = -1
		}
	}

	cmd.Args = cobra.MinimumNArgs(minArgs)
	if maxArgs != -1 {
		cmd.Args = cobra.RangeArgs(minArgs, maxArgs)
	}

	return nil
}

func gatherArguments(args []string, arguments []*SqlParameter) (map[string]interface{}, error) {
	_ = args
	result := make(map[string]interface{})
	argsIdx := 0
	for _, argument := range arguments {
		if argsIdx >= len(args) {
			if argument.Required {
				return nil, errors.Errorf("Argument %s not found", argument.Name)
			} else {
				if argument.Default != nil {
					result[argument.Name] = argument.Default
				}
				continue
			}
		}

		v := []string{args[argsIdx]}

		switch argument.Type {
		case ParameterTypeStringList:
			fallthrough
		case ParameterTypeIntegerList:
			v = args[argsIdx:]
			argsIdx = len(args)
		default:
			argsIdx++
		}
		i2, err := argument.ParseParameter(v)
		if err != nil {
			return nil, err
		}

		result[argument.Name] = i2
	}
	if argsIdx < len(args) {
		return nil, errors.Errorf("Too many arguments")
	}
	return result, nil
}

func addFlags(cmd *cobra.Command, description *SqletonCommandDescription) error {
	for _, parameter := range description.Flags {
		err := parameter.CheckParameterDefaultValueValidity()
		if err != nil {
			return errors.Wrapf(err, "Invalid default value for argument %s", parameter.Name)
		}

		flagName := parameter.Name
		// replace _ with -
		flagName = strings.ReplaceAll(flagName, "_", "-")
		shortFlag := parameter.ShortFlag
		ok := false

		switch parameter.Type {
		case ParameterTypeString:
			defaultValue, ok := parameter.Default.(string)
			if !ok {
				return errors.Errorf("Default value for parameter %s is not a string: %v", parameter.Name, parameter.Default)
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().StringP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().String(flagName, defaultValue, parameter.Help)
			}
		case ParameterTypeInteger:
			defaultValue := 0

			if parameter.Default != nil {
				defaultValue, ok = parameter.Default.(int)
				if !ok {
					return errors.Errorf("Default value for parameter %s is not an integer: %v", parameter.Name, parameter.Default)
				}
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().IntP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().Int(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeBool:
			defaultValue := false

			if parameter.Default != nil {
				defaultValue, ok = parameter.Default.(bool)
				if !ok {
					return errors.Errorf("Default value for parameter %s is not a bool: %v", parameter.Name, parameter.Default)
				}
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().BoolP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().Bool(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeDate:
			defaultValue := ""

			if parameter.Default != nil {
				defaultValue, ok = parameter.Default.(string)
				if !ok {
					return errors.Errorf("Default value for parameter %s is not a string: %v", parameter.Name, parameter.Default)
				}
			}

			parsedDate, err2 := parseDate(defaultValue)
			if err2 != nil {
				return err2
			}
			_ = parsedDate

			if parameter.ShortFlag != "" {
				cmd.Flags().StringP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().String(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeStringList:
			defaultValue := []string{}

			if parameter.Default != nil {
				stringList, ok := parameter.Default.([]string)
				if !ok {
					defaultValue, ok := parameter.Default.([]interface{})
					if !ok {
						return errors.Errorf("Default value for parameter %s is not a string list: %v", parameter.Name, parameter.Default)
					}

					// convert to string list
					stringList, err = convertToStringList(defaultValue)
				}

				defaultValue = stringList
			}
			if err != nil {
				return errors.Wrapf(err, "Could not convert default value for parameter %s to string list: %v", parameter.Name, parameter.Default)
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().StringSliceP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().StringSlice(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeIntegerList:
			defaultValue := []int{}
			if parameter.Default != nil {
				defaultValue, ok = parameter.Default.([]int)
				if !ok {
					return errors.Errorf("Default value for parameter %s is not an integer list: %v", parameter.Name, parameter.Default)
				}
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().IntSliceP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().IntSlice(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeChoice:
			defaultValue := ""

			if parameter.Default != nil {
				defaultValue, ok = parameter.Default.(string)
				if !ok {
					return errors.Errorf("Default value for parameter %s is not a string: %v", parameter.Name, parameter.Default)
				}
			}

			choiceString := strings.Join(parameter.Choices, ",")

			if parameter.ShortFlag != "" {
				cmd.Flags().StringP(flagName, shortFlag, defaultValue, fmt.Sprintf("%s (%s)", parameter.Help, choiceString))
			} else {
				cmd.Flags().String(flagName, defaultValue, fmt.Sprintf("%s (%s)", parameter.Help, choiceString))
			}
		}
	}

	return nil
}

func gatherFlags(cmd *cobra.Command, params []*SqlParameter) (map[string]interface{}, error) {
	parameters := map[string]interface{}{}

	for _, parameter := range params {
		// check if the flag is set
		flagName := parameter.Name
		flagName = strings.ReplaceAll(flagName, "_", "-")

		if !cmd.Flags().Changed(flagName) {
			if parameter.Required {
				return nil, errors.Errorf("Parameter %s is required", parameter.Name)
			}

			if parameter.Default == nil {
				continue
			}
		}

		switch parameter.Type {
		case ParameterTypeString:
			fallthrough
		case ParameterTypeChoice:
			v, err := cmd.Flags().GetString(flagName)
			if err != nil {
				return nil, err
			}
			parameters[parameter.Name] = v

		case ParameterTypeInteger:
			v, err := cmd.Flags().GetInt(flagName)
			if err != nil {
				return nil, err
			}
			parameters[parameter.Name] = v

		case ParameterTypeDate:
			v, err := cmd.Flags().GetString(flagName)
			if err != nil {
				return nil, err
			}
			parsedDate, err := dateparse.ParseAny(v)
			if err != nil {
				parsedDate, err = naturaldate.Parse(v, time.Now())
				if err != nil {
					return nil, errors.Wrapf(err, "Could not parse date %s", v)
				}
			}
			parameters[parameter.Name] = parsedDate

		case ParameterTypeBool:
			v, err := cmd.Flags().GetBool(flagName)
			if err != nil {
				return nil, err
			}
			parameters[parameter.Name] = v

		case ParameterTypeStringList:
			v, err := cmd.Flags().GetStringSlice(flagName)
			if err != nil {
				return nil, err
			}
			parameters[parameter.Name] = v

		case ParameterTypeIntegerList:
			v, err := cmd.Flags().GetIntSlice(flagName)
			if err != nil {
				return nil, err
			}
			parameters[parameter.Name] = v
		}
	}
	return parameters, nil
}

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

func init() {
}
