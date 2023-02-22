package pkg

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
	"text/template"
	"time"
)

type SqletonCommand interface {
	cmds.Command
	RunQueryIntoGlaze(ctx context.Context, db *sqlx.DB, parameters map[string]interface{}, gp *cmds.GlazeProcessor) error
	RenderQuery(parameters map[string]interface{}) (string, error)
}

type SqlCommandDescription struct {
	Name      string                            `yaml:"name"`
	Short     string                            `yaml:"short"`
	Long      string                            `yaml:"long,omitempty"`
	Flags     []*parameters.ParameterDefinition `yaml:"flags,omitempty"`
	Arguments []*parameters.ParameterDefinition `yaml:"arguments,omitempty"`
	Layers    []layers.ParameterLayer           `yaml:"layers,omitempty"`

	Query string `yaml:"query"`
}

type DBConnectionFactory func() (*sqlx.DB, error)

// SqlCommand describes a command line command that runs a query
type SqlCommand struct {
	description         *cmds.CommandDescription
	Query               string
	DBConnectionFactory DBConnectionFactory
}

func (s *SqlCommand) String() string {
	return fmt.Sprintf("SqlCommand{Name: %s, Parents: %s}", s.description.Name, strings.Join(s.description.Parents, " "))
}

func NewSqlCommand(
	description *cmds.CommandDescription,
	query string,
) (*SqlCommand, error) {
	glazedParameterLayer, err := cli.NewGlazedParameterLayers()
	if err != nil {
		return nil, errors.Wrap(err, "could not create Glazed parameter layer")
	}
	sqlConnectionParameterLayer, err := NewSqlConnectionParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL connection parameter layer")
	}
	description.Layers = append(description.Layers,
		glazedParameterLayer,
		sqlConnectionParameterLayer,
	)

	return &SqlCommand{
		description: description,
		Query:       query,
	}, nil
}

func (s *SqlCommand) Run(ps map[string]interface{}, gp *cmds.GlazeProcessor) error {
	if s.DBConnectionFactory == nil {
		return fmt.Errorf("dbConnectionFactory is not set")
	}

	db, err := s.DBConnectionFactory()
	if err != nil {
		return err
	}

	dbContext := context.Background()
	err = db.PingContext(dbContext)
	if err != nil {
		return errors.Wrapf(err, "Could not ping database")
	}

	printQuery, _ := ps["print-query"].(bool)
	if printQuery {
		query, err := s.RenderQuery(ps)
		if err != nil {
			return errors.Wrapf(err, "Could not generate query")
		}
		fmt.Println(query)
		return nil
	}

	err = s.RunQueryIntoGlaze(dbContext, db, ps, gp)
	if err != nil {
		return errors.Wrapf(err, "Could not run query")
	}

	return nil
}

func (s *SqlCommand) Description() *cmds.CommandDescription {
	return s.description
}

func (s *SqlCommand) IsValid() bool {
	return s.description.Name != "" && s.Query != "" && s.description.Short != ""
}

func sqlString(value string) string {
	return fmt.Sprintf("'%s'", value)
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

func stripNewline(value string) string {
	return strings.Replace(value, "\n", " ", -1)
}

func padLeft(value string, length int) string {
	return fmt.Sprintf("%*s", -length, value)
}

func padRight(value string, length int) string {
	return fmt.Sprintf("%-*s", length, value)
}

func (s *SqlCommand) RenderQuery(parameters map[string]interface{}) (string, error) {

	t2 := helpers.CreateTemplate("query").
		Funcs(template.FuncMap{
			"join":         strings.Join,
			"sqlStringIn":  sqlStringIn,
			"sqlIntIn":     sqlIntIn,
			"sqlIn":        sqlIn,
			"sqlDate":      sqlDate,
			"sqlDateTime":  sqlDateTime,
			"sqlLike":      sqlLike,
			"sqlString":    sqlString,
			"stripNewline": stripNewline,
			"padLeft":      padLeft,
			"padRight":     padRight,
		})
	t, err := t2.Parse(s.Query)
	if err != nil {
		return "", errors.Wrap(err, "Could not parse query template")
	}

	return helpers.RenderTemplate(t, parameters)
}

func RunQueryIntoGlaze(
	dbContext context.Context,
	db *sqlx.DB,
	query string,
	parameters []interface{},
	gp *cmds.GlazeProcessor) error {

	rows, err := db.QueryxContext(dbContext, query, parameters...)
	if err != nil {
		return errors.Wrapf(err, "Could not execute query: %s", query)
	}

	return processQueryResults(rows, gp)
}

func RunNamedQueryIntoGlaze(
	dbContext context.Context,
	db *sqlx.DB,
	query string,
	parameters map[string]interface{},
	gp *cmds.GlazeProcessor) error {

	rows, err := db.NamedQueryContext(dbContext, query, parameters)
	if err != nil {
		return errors.Wrapf(err, "Could not execute query: %s", query)
	}

	return processQueryResults(rows, gp)
}

func processQueryResults(rows *sqlx.Rows, gp *cmds.GlazeProcessor) error {
	// we need a way to order the columns
	cols, err := rows.Columns()
	if err != nil {
		return errors.Wrapf(err, "Could not get columns")
	}

	gp.OutputFormatter().SetColumnOrder(cols)

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
	gp *cmds.GlazeProcessor) error {

	query, err := s.RenderQuery(parameters)
	if err != nil {
		return err
	}
	return RunNamedQueryIntoGlaze(ctx, db, query, parameters, gp)
}

type SqlCommandLoader struct {
}

func (scl *SqlCommandLoader) LoadCommandAliasFromYAML(s io.Reader) ([]*cmds.CommandAlias, error) {
	var alias cmds.CommandAlias
	err := yaml.NewDecoder(s).Decode(&alias)
	if err != nil {
		return nil, err
	}

	if !alias.IsValid() {
		return nil, errors.New("Invalid command alias")
	}

	return []*cmds.CommandAlias{&alias}, nil
}

func (scl *SqlCommandLoader) LoadCommandFromYAML(s io.Reader) ([]cmds.Command, error) {
	scd := &SqlCommandDescription{}
	err := yaml.NewDecoder(s).Decode(scd)
	if err != nil {
		return nil, err
	}

	sq, err := NewSqlCommand(
		cmds.NewCommandDescription(
			scd.Name,
			cmds.WithShort(scd.Short),
			cmds.WithLong(scd.Long),
			cmds.WithFlags(scd.Flags...),
			cmds.WithArguments(scd.Arguments...),
			cmds.WithLayers(scd.Layers...),
		),
		scd.Query,
	)
	if err != nil {
		return nil, err
	}

	if !sq.IsValid() {
		return nil, errors.New("Invalid command")
	}

	return []cmds.Command{sq}, nil
}
