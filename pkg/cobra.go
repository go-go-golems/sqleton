package pkg

import (
	"context"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tj/go-naturaldate"
	"github.com/wesen/glazed/pkg/cli"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"time"
)

// refTime is used to set a reference time for natural date parsing for unit test purposes
var refTime *time.Time

// gatherParameters takes a cobra command, an argument list as well as a description
// of the sqleton command arguments, and returns a list of parsed parameters as a
// hashmap. It does so by parsing both the flags and the positional arguments.
func gatherParameters(
	cmd *cobra.Command,
	description *SqletonCommandDescription,
	args []string,
) (map[string]interface{}, error) {
	parameters, err := gatherFlags(cmd, description.Flags, false)
	if err != nil {
		return nil, err
	}

	arguments, err := gatherArguments(args, description.Arguments, false)
	if err != nil {
		return nil, err
	}

	createAlias, err := cmd.Flags().GetString("create-alias")
	if err != nil {
		return nil, err
	}
	if createAlias != "" {
		alias := &CommandAlias{
			Name:      createAlias,
			AliasFor:  description.Name,
			Arguments: args,
			Flags:     map[string]string{},
		}

		cmd.Flags().Visit(func(flag *pflag.Flag) {
			if flag.Name != "create-alias" {
				alias.Flags[flag.Name] = flag.Value.String()
			}
		})

		// marshal alias to yaml
		sb := strings.Builder{}
		encoder := yaml.NewEncoder(&sb)
		err = encoder.Encode(alias)
		if err != nil {
			return nil, err
		}

		fmt.Println(sb.String())
		os.Exit(0)
	}

	// merge parameters and arguments
	// arguments take precedence over parameters
	for k, v := range arguments {
		parameters[k] = v
	}

	return parameters, nil
}

// runSqletonCommand actually runs the given SqletonCommand by using the cobra command
// to parse the necessary flags. It then first pings the database, and then renders
// the query results into a GlazedProcessor.
func runSqletonCommand(cmd *cobra.Command, s SqletonCommand, args []string) error {
	// TODO(2022-12-20, manuel): we should be able to load default values for these parameters from a config file
	// See: https://github.com/wesen/sqleton/issues/39
	description := s.Description()

	parameters, err := gatherParameters(cmd, &description, args)
	if err != nil {
		return err
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

	output, err := of.Output()
	if err != nil {
		return errors.Wrapf(err, "Could not get output")
	}
	fmt.Print(output)

	return nil
}

// ToCobraCommand converts a SqletonCommand into a concrete cobra command.
// It adds the necessary flags to the cobra command, configures the positional arguments
// validity checks and sets the Run command to actually run the command.
func ToCobraCommand(s SqletonCommand) (*cobra.Command, error) {
	description := s.Description()
	cmd, err := NewCobraCommandFromDescription(description)
	if err != nil {
		return nil, err
	}

	cmd.Run = func(cmd *cobra.Command, args []string) {
		err := runSqletonCommand(cmd, s, args)
		cobra.CheckErr(err)
	}

	return cmd, nil
}

func NewCobraCommandFromDescription(description SqletonCommandDescription) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   description.Name,
		Short: description.Short,
		Long:  description.Long,
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
	cmd.Flags().String("create-alias", "", "Create an alias for the query")

	cli.AddFlags(cmd, cli.NewFlagsDefaults())

	return cmd, nil
}

// addArguments adds the arguments (not the flags) of a SqletonCommandDescription to a cobra command
// as positional arguments.
// An optional argument cannot be followed by a required argument.
// Similarly, a list of arguments cannot be followed by any argument (since we otherwise wouldn't
// know how many belong to the list and where to do the cut off).
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

// gatherArguments parses the positional arguments passed as a list of strings into a map of
// parsed values. If onlyProvided is true, then only arguments that are provided are returned
// (i.e. the default values are not included).
func gatherArguments(args []string, arguments []*SqlParameter, onlyProvided bool) (map[string]interface{}, error) {
	_ = args
	result := make(map[string]interface{})
	argsIdx := 0
	for _, argument := range arguments {
		if argsIdx >= len(args) {
			if argument.Required {
				return nil, errors.Errorf("Argument %s not found", argument.Name)
			} else {
				if argument.Default != nil && !onlyProvided {
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

// addFlags takes the parameters from a SqletonCommandDescription and converts them
// to cobra flags, before adding them to the Flags() of a the passed cobra command.
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
			defaultValue := ""

			if parameter.Default != nil {
				defaultValue, ok = parameter.Default.(string)
				if !ok {
					return errors.Errorf("Default value for parameter %s is not a string: %v", parameter.Name, parameter.Default)
				}
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

				parsedDate, err2 := parseDate(defaultValue)
				if err2 != nil {
					return err2
				}
				_ = parsedDate
			}

			if parameter.ShortFlag != "" {
				cmd.Flags().StringP(flagName, shortFlag, defaultValue, parameter.Help)
			} else {
				cmd.Flags().String(flagName, defaultValue, parameter.Help)
			}

		case ParameterTypeStringList:
			var defaultValue []string

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
			var defaultValue []int
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

// gatherFlags gathers the flags from the cobra command, and parses them according
// to the parameter description passed in params. The result is a map of parameter
// names to parsed values. If onlyProvided is true, only parameters that are provided
// by the user are returned (i.e. not the default values).
// If a parameter cannot be parsed correctly, or is missing even though it is not optional,
// an error is returned.
func gatherFlags(cmd *cobra.Command, params []*SqlParameter, onlyProvided bool) (map[string]interface{}, error) {
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

			if onlyProvided {
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

// findOrCreateParentCommand will create empty commands to anchor the passed in parents.
func findOrCreateParentCommand(rootCmd *cobra.Command, parents []string) *cobra.Command {
	parentCmd := rootCmd
	for _, parent := range parents {
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
	return parentCmd
}

func AddCommandsToRootCommand(rootCmd *cobra.Command, commands []*SqlCommand, aliases []*CommandAlias) error {
	commandsByName := map[string]*SqlCommand{}

	for _, command := range commands {
		// find the proper subcommand, or create if it doesn't exist
		parentCmd := findOrCreateParentCommand(rootCmd, command.Parents)
		description := command.Description()
		cobraCommand, err := NewCobraCommandFromDescription(description)
		if err != nil {
			return err
		}

		command2 := command
		cobraCommand.Run = func(cmd *cobra.Command, args []string) {
			err := runSqletonCommand(cmd, command2, args)
			cobra.CheckErr(err)
		}
		parentCmd.AddCommand(cobraCommand)

		path := strings.Join(append(command.Parents, command.Name), " ")
		commandsByName[path] = command
	}

	for _, alias := range aliases {
		path := strings.Join(alias.Parents, " ")
		aliasedCommand, ok := commandsByName[path]
		if !ok {
			return errors.Errorf("Command %s not found for alias %s", path, alias.Name)
		}
		alias.AliasedCommand = aliasedCommand

		parentCmd := findOrCreateParentCommand(rootCmd, alias.Parents)
		cobraCommand, err := NewCobraCommandFromDescription(alias.Description())
		if err != nil {
			return err
		}
		alias2 := alias
		cobraCommand.Run = func(cmd *cobra.Command, args []string) {
			for flagName, flagValue := range alias2.Flags {
				if !cmd.Flags().Changed(flagName) {
					err = cmd.Flags().Set(flagName, flagValue)
					cobra.CheckErr(err)
				}
			}
			// TODO(2022-12-22, manuel) This is not right because the args count is checked earlier, but when,
			// and how can i override it
			if len(args) == 0 {
				args = alias2.Arguments
			}
			err = runSqletonCommand(cmd, aliasedCommand, args)
			cobra.CheckErr(err)
		}
		parentCmd.AddCommand(cobraCommand)
	}

	return nil
}

func init() {
}
