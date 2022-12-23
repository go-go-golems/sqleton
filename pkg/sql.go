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
	"time"
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

func (s *SqlParameter) Copy() *SqlParameter {
	return &SqlParameter{
		Name:      s.Name,
		ShortFlag: s.ShortFlag,
		Type:      s.Type,
		Help:      s.Help,
		Default:   s.Default,
		Choices:   s.Choices,
		Required:  s.Required,
	}
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
	// we can have no default
	if sp.Default == nil {
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
			fixedDefault, err := convertToStringList(defaultValue)
			if err != nil {
				return errors.Wrapf(err, "Could not convert default value for parameter %s to string list: %v", sp.Name, sp.Default)
			}
			sp.Default = fixedDefault
		}

	case ParameterTypeIntegerList:
		_, ok := sp.Default.([]int)
		if !ok {
			return errors.Errorf("Default value for parameter %s is not an integer list: %v", sp.Name, sp.Default)
		}

	case ParameterTypeChoice:
		if len(sp.Choices) == 0 {
			return errors.Errorf("Parameter %s is a choice parameter but has no choices", sp.Name)
		}

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
	Name      string          `yaml:"name"`
	Short     string          `yaml:"short"`
	Long      string          `yaml:"long"`
	Flags     []*SqlParameter `yaml:"flags"`
	Arguments []*SqlParameter `yaml:"arguments"`
	Query     string          `yaml:"query"`

	Parents []string `yaml:",omitempty"`
	Source  string   `yaml:",omitempty"`
}

func (sc *SqlCommand) IsValid() bool {
	return sc.Name != "" && sc.Query != "" && sc.Short != ""
}

func sqlStringIn(values []string) string {
	return fmt.Sprintf("'%s'", strings.Join(values, "','"))
}

func sqlIn(values []interface{}) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(strValues, ",")
}

func sqlIntIn(values []int) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%d", v)
	}
	return strings.Join(strValues, ",")
}

func sqlDate(date time.Time) string {
	return "'" + date.Format("2006-01-02") + "'"
}

func sqlDateTime(date time.Time) string {
	return "'" + date.Format("2006-01-02 15:04:05") + "'"
}

func sqlLike(value string) string {
	return "'%" + value + "%'"
}

func (s *SqlCommand) RenderQuery(parameters map[string]interface{}) (string, error) {
	t2 := template.New("query")
	t2.Funcs(template.FuncMap{
		"join":        strings.Join,
		"sqlStringIn": sqlStringIn,
		"sqlIntIn":    sqlIntIn,
		"sqlIn":       sqlIn,
		"sqlDate":     sqlDate,
		"sqlDateTime": sqlDateTime,
		"sqlLike":     sqlLike,
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
		Name:      s.Name,
		Short:     s.Short,
		Long:      s.Long,
		Flags:     s.Flags,
		Arguments: s.Arguments,
	}
}

func LoadCommandAliasFromYaml(s io.Reader) (*CommandAlias, error) {
	var alias CommandAlias
	err := yaml.NewDecoder(s).Decode(&alias)
	if err != nil {
		return nil, err
	}

	if !alias.IsValid() {
		return nil, errors.New("Invalid command alias")
	}

	return &alias, nil
}

func LoadSqlCommandFromYaml(s io.Reader) (*SqlCommand, error) {
	sq := &SqlCommand{
		Flags:     []*SqlParameter{},
		Arguments: []*SqlParameter{},
	}
	err := yaml.NewDecoder(s).Decode(sq)
	if err != nil {
		return nil, err
	}

	if !sq.IsValid() {
		return nil, errors.New("Invalid command")
	}

	return sq, nil
}

func LoadSqlCommandsFromEmbedFS(f embed.FS, dir string, cmdRoot string) ([]*SqlCommand, []*CommandAlias, error) {
	var commands []*SqlCommand
	var aliases []*CommandAlias

	entries, err := f.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, entry := range entries {
		// skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		fileName := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			subCommands, _, err := LoadSqlCommandsFromEmbedFS(f, fileName, cmdRoot)
			if err != nil {
				return nil, nil, err
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
					alias, err := func() (*CommandAlias, error) {
						file, err := f.Open(fileName)
						if err != nil {
							return nil, errors.Wrapf(err, "Could not open file %s", fileName)
						}
						defer func() {
							_ = file.Close()
						}()

						alias, err := LoadCommandAliasFromYaml(file)
						if err != nil {
							return nil, err
						}
						alias.Source = "embed:" + fileName

						pathToFile := strings.TrimPrefix(dir, cmdRoot)
						alias.Parents = strings.Split(pathToFile, "/")

						return alias, err
					}()

					if err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "Could not load command or alias from file %s: %s\n", fileName, err)
						continue
					} else {
						aliases = append(aliases, alias)
					}
				} else {
					commands = append(commands, command)
				}
			}
		}
	}

	return commands, aliases, nil
}

func LoadSqlCommandsFromDirectory(dir string, cmdRoot string) ([]*SqlCommand, []*CommandAlias, error) {
	var commands []*SqlCommand
	var aliases []*CommandAlias

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, entry := range entries {
		// skip hidden files
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		fileName := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			subCommands, subAliases, err := LoadSqlCommandsFromDirectory(fileName, cmdRoot)
			if err != nil {
				return nil, nil, err
			}
			commands = append(commands, subCommands...)
			aliases = append(aliases, subAliases...)
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
					alias, err := func() (*CommandAlias, error) {
						file, err := os.Open(fileName)
						if err != nil {
							return nil, errors.Wrapf(err, "Could not open file %s", fileName)
						}
						defer func() {
							_ = file.Close()
						}()

						alias, err := LoadCommandAliasFromYaml(file)
						if err != nil {
							return nil, err
						}
						alias.Source = "file:" + fileName

						pathToFile := strings.TrimPrefix(dir, cmdRoot)
						pathToFile = strings.TrimPrefix(pathToFile, "/")
						alias.Parents = strings.Split(pathToFile, "/")

						return alias, err
					}()

					if err != nil {
						_, _ = fmt.Fprintf(os.Stderr, "Could not load command or alias from file %s: %s\n", fileName, err)
						continue

					} else {
						aliases = append(aliases, alias)
					}
				} else {
					commands = append(commands, command)
				}
			}
		}
	}

	return commands, aliases, nil
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
