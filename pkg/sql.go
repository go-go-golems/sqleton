package pkg

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/helpers"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
	"text/template"
	"time"
)

type SqletonCommand interface {
	cmds.CobraCommand
	RunQueryIntoGlaze(ctx context.Context, db *sqlx.DB, parameters map[string]interface{}, gp *cli.GlazeProcessor) error
	RenderQuery(parameters map[string]interface{}) (string, error)
}

type SqlCommandDescription struct {
	Name      string            `yaml:"name"`
	Short     string            `yaml:"short"`
	Long      string            `yaml:"long,omitempty"`
	Flags     []*cmds.Parameter `yaml:"flags,omitempty"`
	Arguments []*cmds.Parameter `yaml:"arguments,omitempty"`

	Query string `yaml:"query"`
}

// SqlCommand describes a command line command that runs a query
type SqlCommand struct {
	description *cmds.CommandDescription
	Query       string
}

func (s *SqlCommand) String() string {
	return fmt.Sprintf("SqlCommand{Name: %s, Parents: %s}", s.description.Name, strings.Join(s.description.Parents, " "))
}

func (s *SqlCommand) BuildCobraCommand() (*cobra.Command, error) {
	cmd, err := cmds.NewCobraCommand(s)
	if err != nil {
		return nil, err
	}
	cmd.Flags().Bool("print-query", false, "Print the query that will be executed")
	cmd.Flags().Bool("explain", false, "Print the query plan that will be executed")

	// add glazed flags
	cli.AddFlags(cmd, cli.NewFlagsDefaults())

	return cmd, nil
}

func NewSqlCommand(description *cmds.CommandDescription, query string) *SqlCommand {
	return &SqlCommand{
		description: description,
		Query:       query,
	}
}

func (s *SqlCommand) Run(map[string]interface{}) error {
	//TODO implement me
	panic("implement me")
}

func (s *SqlCommand) Description() *cmds.CommandDescription {
	return s.description
}

func (sc *SqlCommand) IsValid() bool {
	return sc.description.Name != "" && sc.Query != "" && sc.description.Short != ""
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
	gp *cli.GlazeProcessor) error {

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
	gp *cli.GlazeProcessor) error {

	rows, err := db.NamedQueryContext(dbContext, query, parameters)
	if err != nil {
		return errors.Wrapf(err, "Could not execute query: %s", query)
	}

	return processQueryResults(rows, gp)
}

func processQueryResults(rows *sqlx.Rows, gp *cli.GlazeProcessor) error {
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
	gp *cli.GlazeProcessor) error {

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
	scd := &SqlCommandDescription{
		Flags:     []*cmds.Parameter{},
		Arguments: []*cmds.Parameter{},
	}
	err := yaml.NewDecoder(s).Decode(scd)
	if err != nil {
		return nil, err
	}

	sq := &SqlCommand{
		Query: scd.Query,
		description: &cmds.CommandDescription{
			Name:      scd.Name,
			Short:     scd.Short,
			Long:      scd.Long,
			Flags:     scd.Flags,
			Arguments: scd.Arguments,
		},
	}

	if !sq.IsValid() {
		return nil, errors.New("Invalid command")
	}

	return []cmds.Command{sq}, nil
}
