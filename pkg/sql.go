package pkg

import (
	"context"
	"embed"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/glazed/pkg/middlewares"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

type SqlParameter struct {
	Name      string        `yaml:"name"`
	ShortFlag string        `yaml:"shortFlag"`
	Type      ParameterType `yaml:"type"`
	Help      string        `yaml:"help"`
	Default   interface{}   `yaml:"default"`
	Choices   []string      `yaml:"choices"`
	Required  bool          `yaml:"required"`
}

type SqletonCommandDescription struct {
	Name      string
	Short     string
	Long      string
	Flags     []*SqlParameter
	Arguments []*SqlParameter
}

// string enum for parameters
// a parameter can be:
// - a string
// - a number
// - a boolean
// - a date
// - a list of numbers
// - a list of strings
// - a choice out of strings

// TODO(2022-12-21, manuel): Add list of choices as a type
// what about list of dates? list of bools?
// should list just be a flag?

type ParameterType string

const (
	ParameterTypeString      ParameterType = "string"
	ParameterTypeInteger     ParameterType = "int"
	ParameterTypeBool        ParameterType = "bool"
	ParameterTypeDate        ParameterType = "date"
	ParameterTypeStringList  ParameterType = "stringList"
	ParameterTypeIntegerList ParameterType = "intList"
	ParameterTypeChoice      ParameterType = "choice"
)

type SqletonCommand interface {
	RunQueryIntoGlaze(ctx context.Context, db *sqlx.DB, parameters map[string]interface{}, gp *cli.GlazeProcessor) error
	RenderQuery(parameters map[string]interface{}) (string, error)
	Description() SqletonCommandDescription
}

func (sp *SqlParameter) CheckParameterDefaultValueValidity() error {
	// optional parameters can have a nil value
	if sp.Required && sp.Default == nil {
		return nil
	}

	switch sp.Type {
	case ParameterTypeString:
		_, ok := sp.Default.(string)
		if !ok {
			return errors.Errorf("Default value for parameter %s is not a string: %v", sp.Name, sp.Default)
		}
	case ParameterTypeInteger:
		_, ok := sp.Default.(int)
		if !ok {
			return errors.Errorf("Default value for parameter %s is not an integer: %v", sp.Name, sp.Default)
		}

	case ParameterTypeBool:
		_, ok := sp.Default.(bool)
		if !ok {
			return errors.Errorf("Default value for parameter %s is not a bool: %v", sp.Name, sp.Default)
		}

	case ParameterTypeDate:
		defaultValue, ok := sp.Default.(string)
		if !ok {
			return errors.Errorf("Default value for parameter %s is not a string: %v", sp.Name, sp.Default)
		}

		_, err2 := parseDate(defaultValue)
		if err2 != nil {
			return errors.Wrapf(err2, "Default value for parameter %s is not a valid date: %v", sp.Name, sp.Default)
		}

	case ParameterTypeStringList:
		_, ok := sp.Default.([]string)
		if !ok {
			defaultValue, ok := sp.Default.([]interface{})
			if !ok {
				return errors.Errorf("Default value for parameter %s is not a string list: %v", sp.Name, sp.Default)
			}

			// convert to string list
			_, err := convertToStringList(defaultValue)
			if err != nil {
				return errors.Wrapf(err, "Could not convert default value for parameter %s to string list: %v", sp.Name, sp.Default)
			}
		}

	case ParameterTypeIntegerList:
		_, ok := sp.Default.([]int)
		if !ok {
			return errors.Errorf("Default value for parameter %s is not an integer list: %v", sp.Name, sp.Default)
		}

	case ParameterTypeChoice:
		defaultValue, ok := sp.Default.(string)
		if !ok {
			return errors.Errorf("Default value for parameter %s is not a string: %v", sp.Name, sp.Default)
		}

		found := false
		for _, choice := range sp.Choices {
			if choice == defaultValue {
				found = true
			}
		}
		if !found {
			return errors.Errorf("Default value for parameter %s is not a valid choice: %v", sp.Name, sp.Default)
		}
	}

	return nil
}

// SqlCommand describes a command line command that runs a query
type SqlCommand struct {
	Name       string          `yaml:"name"`
	Short      string          `yaml:"short"`
	Long       string          `yaml:"long"`
	Parameters []*SqlParameter `yaml:"parameters"`
	Query      string          `yaml:"query"`

	Parents []string
	Source  string
}

func (s *SqlCommand) RenderQuery(parameters map[string]interface{}) (string, error) {
	t2 := template.New("query")
	t2.Funcs(template.FuncMap{
		"join": strings.Join,
		"sqlStringIn": func(values []string) string {
			return "'" + strings.Join(values, "','") + "'"
		},
		"sqlIn": func(values []interface{}) string {
			strValues := make([]string, len(values))
			for i, v := range values {
				strValues[i] = fmt.Sprintf("%v", v)
			}
			return strings.Join(strValues, ",")
		},
	})
	t := template.Must(t2.Parse(s.Query))
	var qb strings.Builder
	err := t.Execute(&qb, parameters)
	if err != nil {
		return "", errors.Wrap(err, "Could not execute query template")
	}

	return qb.String(), nil
}

func RunQueryIntoGlaze(
	dbContext context.Context,
	db *sqlx.DB,
	query string,
	parameters map[string]interface{},
	gp *cli.GlazeProcessor) error {

	rows, err := db.NamedQueryContext(dbContext, query, parameters)
	if err != nil {
		return errors.Wrapf(err, "Could not execute query: %s", query)
	}

	// we need a way to order the columns
	cols, err := rows.Columns()
	if err != nil {
		return errors.Wrapf(err, "Could not get columns")
	}

	gp.OutputFormatter().AddTableMiddleware(middlewares.NewReorderColumnOrderMiddleware(cols))
	// add support for renaming columns (at least to lowercase)
	// https://github.com/wesen/glazed/issues/27

	for rows.Next() {
		row := map[string]interface{}{}
		err = rows.MapScan(row)
		if err != nil {
			return errors.Wrapf(err, "Could not scan row")
		}

		for key, value := range row {
			switch value := value.(type) {
			case []byte:
				row[key] = string(value)
			}
		}

		err = gp.ProcessInputObject(row)
		if err != nil {
			return errors.Wrapf(err, "Could not process input object")
		}
	}

	return nil
}

func (s *SqlCommand) RunQueryIntoGlaze(
	ctx context.Context,
	db *sqlx.DB,
	parameters map[string]interface{},
	gp *cli.GlazeProcessor) error {

	query, err := s.RenderQuery(parameters)
	if err != nil {
		return err
	}
	return RunQueryIntoGlaze(ctx, db, query, parameters, gp)
}

func (s *SqlCommand) Description() SqletonCommandDescription {
	return SqletonCommandDescription{
		Name:  s.Name,
		Short: s.Short,
		Long:  s.Long,
		Flags: s.Parameters,
	}
}

func LoadSqlCommandFromYaml(s io.Reader) (*SqlCommand, error) {
	sq := &SqlCommand{
		Parameters: []*SqlParameter{},
	}
	err := yaml.NewDecoder(s).Decode(sq)
	if err != nil {
		return nil, err
	}

	return sq, nil
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
					command.Source = "embed:" + fileName

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

					command.Source = "file:" + fileName

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

func (sp *SqlParameter) ParseParameter(v []string) (interface{}, error) {
	if len(v) == 0 {
		if sp.Required {
			return nil, errors.Errorf("Argument %s not found", sp.Name)
		} else {
			return sp.Default, nil
		}
	}

	switch sp.Type {
	case ParameterTypeString:
		return v[0], nil
	case ParameterTypeInteger:
		i, err := strconv.Atoi(v[0])
		if err != nil {
			return nil, errors.Wrapf(err, "Could not parse argument %s as integer", sp.Name)
		}
		return i, nil
	case ParameterTypeStringList:
		return v, nil
	case ParameterTypeIntegerList:
		ints := make([]int, 0)
		for _, arg := range v {
			i, err := strconv.Atoi(arg)
			if err != nil {
				return nil, errors.Wrapf(err, "Could not parse argument %s as integer", sp.Name)
			}
			ints = append(ints, i)
		}
		return ints, nil

	case ParameterTypeBool:
		b, err := strconv.ParseBool(v[0])
		if err != nil {
			return nil, errors.Wrapf(err, "Could not parse argument %s as bool", sp.Name)
		}
		return b, nil

	case ParameterTypeChoice:
		choice := v[0]
		found := false
		for _, c := range sp.Choices {
			if c == choice {
				found = true
			}
		}
		if !found {
			return nil, errors.Errorf("Argument %s has invalid choice %s", sp.Name, choice)
		}
		return choice, nil

	case ParameterTypeDate:
		parsedDate, err := parseDate(v[0])
		if err != nil {
			return nil, errors.Wrapf(err, "Could not parse argument %s as date", sp.Name)
		}
		return parsedDate, nil
	}

	return nil, errors.Errorf("Unknown parameter type %s", sp.Type)
}
