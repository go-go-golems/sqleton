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
}

type SqletonCommandDescription struct {
	Name       string
	Short      string
	Long       string
	Parameters []*SqlParameter
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
	Description() SqletonCommandDescription
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
		return errors.Wrap(err, "Could not execute query template")
	}

	return RunQueryIntoGlaze(ctx, db, qb.String(), parameters, gp)
}

func (s *SqlCommand) Description() SqletonCommandDescription {
	return SqletonCommandDescription{
		Name:       s.Name,
		Short:      s.Short,
		Long:       s.Long,
		Parameters: s.Parameters,
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
