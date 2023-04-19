package pkg

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/cast"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
	"text/template"
	"time"
)

type SqletonCommand interface {
	cmds.GlazeCommand
	RunQueryIntoGlaze(
		ctx context.Context,
		db *sqlx.DB,
		parameters map[string]interface{},
		gp cmds.Processor,
	) error
	RenderQuery(parameters map[string]interface{}) (string, error)
}

type SqlCommandDescription struct {
	Name      string                            `yaml:"name"`
	Short     string                            `yaml:"short"`
	Long      string                            `yaml:"long,omitempty"`
	Layout    [][]string                        `yaml:"layout,omitempty"`
	Flags     []*parameters.ParameterDefinition `yaml:"flags,omitempty"`
	Arguments []*parameters.ParameterDefinition `yaml:"arguments,omitempty"`
	Layers    []layers.ParameterLayer           `yaml:"layers,omitempty"`

	SubQueries map[string]string `yaml:"subqueries,omitempty"`
	Query      string            `yaml:"query"`
}

type DBConnectionFactory func(parsedLayers map[string]*layers.ParsedParameterLayer) (*sqlx.DB, error)

// SqlCommand describes a command line command that runs a query
type SqlCommand struct {
	description         *cmds.CommandDescription
	Query               string
	SubQueries          map[string]string
	dbConnectionFactory DBConnectionFactory
}

func (s *SqlCommand) String() string {
	return fmt.Sprintf("SqlCommand{Name: %s, Parents: %s}", s.description.Name, strings.Join(s.description.Parents, " "))
}

type SqlCommandOption func(*SqlCommand)

func WithDbConnectionFactory(factory DBConnectionFactory) SqlCommandOption {
	return func(s *SqlCommand) {
		s.dbConnectionFactory = factory
	}
}

func WithQuery(query string) SqlCommandOption {
	return func(s *SqlCommand) {
		s.Query = query
	}
}

func WithSubQueries(subQueries map[string]string) SqlCommandOption {
	return func(s *SqlCommand) {
		s.SubQueries = subQueries
	}
}

func NewSqlCommand(
	description *cmds.CommandDescription,
	options ...SqlCommandOption,
) (*SqlCommand, error) {
	glazedParameterLayer, err := cli.NewGlazedParameterLayers()
	if err != nil {
		return nil, errors.Wrap(err, "could not create Glazed parameter layer")
	}
	sqlConnectionParameterLayer, err := NewSqlConnectionParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL connection parameter layer")
	}
	dbtParameterLayer, err := NewDbtParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create dbt parameter layer")
	}
	sqlHelpersParameterLayer, err := NewSqlHelpersParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL helpers parameter layer")
	}
	description.Layers = append(description.Layers,
		sqlHelpersParameterLayer,
		glazedParameterLayer,
		sqlConnectionParameterLayer,
		dbtParameterLayer,
	)

	ret := &SqlCommand{
		description: description,
		SubQueries:  make(map[string]string),
	}

	for _, option := range options {
		option(ret)
	}

	return ret, nil
}

func (s *SqlCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp cmds.Processor,
) error {
	if s.dbConnectionFactory == nil {
		return fmt.Errorf("dbConnectionFactory is not set")
	}

	// at this point, the factory can probably be passed the sqleton-connection parsed layer
	db, err := s.dbConnectionFactory(parsedLayers)
	if err != nil {
		return err
	}

	err = db.PingContext(ctx)
	if err != nil {
		return errors.Wrapf(err, "Could not ping database")
	}

	printQuery, _ := ps["print-query"].(bool)
	if printQuery {
		query, err := s.RenderQuery(ctx, ps, db)
		if err != nil {
			return errors.Wrapf(err, "Could not generate query")
		}
		fmt.Println(query)
		return &cmds.ExitWithoutGlazeError{}
	}

	err = s.RunQueryIntoGlaze(ctx, db, ps, gp)
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

func sqlEscape(value string) string {
	return strings.Replace(value, "'", "''", -1)
}

func sqlString(value string) string {
	return fmt.Sprintf("'%s'", value)
}

func sqlStringLike(value string) string {
	return fmt.Sprintf("'%%%s%%'", sqlEscape(value))
}

func sqlStringIn(values interface{}) (string, error) {
	strList, ok := cast.CastList2[string, interface{}](values)
	if !ok {
		return "", fmt.Errorf("could not cast %v to []string", values)
	}
	return fmt.Sprintf("'%s'", strings.Join(strList, "','")), nil
}

func sqlIn(values []interface{}) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%v", v)
	}
	return strings.Join(strValues, ",")
}

func sqlIntIn(values interface{}) string {
	v_, ok := cast.CastInterfaceToIntList[int64](values)
	if !ok {
		return ""
	}
	strValues := make([]string, len(v_))
	for i, v := range v_ {
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

func createTemplate(
	ctx context.Context,
	subQueries map[string]string,
	ps map[string]interface{},
	db *sqlx.DB,
) *template.Template {
	t2 := templating.CreateTemplate("query").
		Funcs(templating.TemplateFuncs).
		Funcs(template.FuncMap{
			"sqlStringIn":   sqlStringIn,
			"sqlStringLike": sqlStringLike,
			"sqlIntIn":      sqlIntIn,
			"sqlIn":         sqlIn,
			"sqlDate":       sqlDate,
			"sqlDateTime":   sqlDateTime,
			"sqlLike":       sqlLike,
			"sqlString":     sqlString,
			"sqlEscape":     sqlEscape,
			"subQuery": func(name string) (string, error) {
				s, ok := subQueries[name]
				if !ok {
					return "", errors.Errorf("Subquery %s not found", name)
				}
				return s, nil
			},
			"sqlSlice": func(query string, args ...interface{}) ([]interface{}, error) {
				_, rows, err := runQuery(ctx, subQueries, query, args, ps, db)
				if err != nil {
					// TODO(manuel, 2023-03-27) This nesting of errors in nested templates becomes quite unpalatable
					// This is what can be output for just one level deep:
					//
					// Error: Could not generate query: template: query:1:13: executing "query" at <sqlColumn (subQuery "post_types")>: error calling sqlColumn: Could not run query: SELECT post_type
					// FROM wp_posts
					// GROUP BY post_type
					// ORDER BY post_type
					// : Error 1146 (42S02): Table 'ttc_analytics.wp_posts' doesn't exist
					// exit status 1
					//
					// Make better error messages:
					return nil, errors.Wrapf(err, "Could not run query: %s", query)
				}
				defer rows.Close()

				ret := []interface{}{}

				for rows.Next() {
					ret_, err := rows.SliceScan()
					if err != nil {
						return nil, errors.Wrapf(err, "Could not scan query: %s", query)
					}

					row := make([]interface{}, len(ret_))
					for i, v := range ret_ {
						row[i] = sqlEltToTemplateValue(v)
					}

					ret = append(ret, row)
				}

				return ret, nil
			},
			"sqlColumn": func(query string, args ...interface{}) ([]interface{}, error) {
				renderedQuery, rows, err := runQuery(ctx, subQueries, query, args, ps, db)
				if err != nil {
					return nil, errors.Wrapf(err, "Could not run query: %s", renderedQuery)
				}
				defer rows.Close()

				ret := make([]interface{}, 0)
				for rows.Next() {
					rows_, err := rows.SliceScan()
					if err != nil {
						return nil, errors.Wrapf(err, "Could not scan query: %s", renderedQuery)
					}

					if len(rows_) != 1 {
						return nil, errors.Errorf("Expected 1 column, got %d", len(rows_))
					}
					elt := rows_[0]

					v := sqlEltToTemplateValue(elt)

					ret = append(ret, v)
				}

				return ret, nil
			},
			"sqlSingle": func(query string, args ...interface{}) (interface{}, error) {
				renderedQuery, rows, err := runQuery(ctx, subQueries, query, args, ps, db)
				if err != nil {
					return nil, errors.Wrapf(err, "Could not run query: %s", renderedQuery)
				}
				defer rows.Close()

				ret := make([]interface{}, 0)
				if rows.Next() {
					rows_, err := rows.SliceScan()
					if err != nil {
						return nil, errors.Wrapf(err, "Could not scan query: %s", renderedQuery)
					}

					if len(rows_) != 1 {
						return nil, errors.Errorf("Expected 1 column, got %d", len(rows_))
					}

					ret = append(ret, rows_[0])
				}

				if rows.Next() {
					return nil, errors.Errorf("Expected 1 row, got more")
				}

				if len(ret) == 0 {
					return nil, nil
				}

				if len(ret) > 1 {
					return nil, errors.Errorf("Expected 1 row, got %d", len(ret))
				}

				return sqlEltToTemplateValue(ret[0]), nil
			},
			"sqlMap": func(query string, args ...interface{}) (interface{}, error) {
				renderedQuery, rows, err := runQuery(ctx, subQueries, query, args, ps, db)
				if err != nil {
					return nil, errors.Wrapf(err, "Could not run query: %s", renderedQuery)
				}
				defer rows.Close()

				ret := []map[string]interface{}{}

				for rows.Next() {
					ret_ := make(map[string]interface{})
					err = rows.MapScan(ret_)
					if err != nil {
						return nil, errors.Wrapf(err, "Could not scan query: %s", renderedQuery)
					}

					row := make(map[string]interface{})
					for k, v := range ret_ {
						row[k] = sqlEltToTemplateValue(v)
					}

					ret = append(ret, row)
				}

				return ret, nil
			},
		})

	return t2
}

func sqlEltToTemplateValue(elt interface{}) interface{} {
	switch v := elt.(type) {
	case []byte:
		return string(v)
	default:
		return v
	}
}

func runQuery(
	ctx context.Context,
	subQueries map[string]string,
	query string,
	args []interface{},
	ps map[string]interface{},
	db *sqlx.DB,
) (string, *sqlx.Rows, error) {
	if db == nil {
		return "", nil, errors.New("No database connection")
	}

	ps2 := map[string]interface{}{}
	for k, v := range ps {
		ps2[k] = v
	}
	// args is k, v, k, v, k, v
	if len(args)%2 != 0 {
		return "", nil, errors.Errorf("Could not run query: %s", query)
	}
	for i := 0; i < len(args); i += 2 {
		k, ok := args[i].(string)
		if !ok {
			return "", nil, errors.Errorf("Could not run query: %s", query)
		}
		ps2[k] = args[i+1]
	}

	t2 := createTemplate(ctx, subQueries, ps2, db)
	t, err := t2.Parse(query)
	if err != nil {
		return "", nil, err
	}

	query_, err := templating.RenderTemplate(t, ps2)
	if err != nil {
		return query_, nil, err
	}

	stmt, err := db.PreparexContext(ctx, query_)
	if err != nil {
		return query_, nil, err
	}

	rows, err := stmt.QueryxContext(ctx)
	if err != nil {
		return query_, nil, err
	}

	return query_, rows, err
}

func (s *SqlCommand) RenderQuery(
	ctx context.Context,
	ps map[string]interface{},
	db *sqlx.DB,
) (string, error) {
	t2 := createTemplate(ctx, s.SubQueries, ps, db)

	t, err := t2.Parse(s.Query)
	if err != nil {
		return "", errors.Wrap(err, "Could not parse query template")
	}

	return templating.RenderTemplate(t, ps)
}

func RunQueryIntoGlaze(
	dbContext context.Context,
	db *sqlx.DB,
	query string,
	parameters []interface{},
	gp cmds.Processor) error {

	// use a prepared statement so that when using mysql, we get native types back
	stmt, err := db.PreparexContext(dbContext, query)
	if err != nil {
		return errors.Wrapf(err, "Could not prepare query: %s", query)
	}

	rows, err := stmt.QueryxContext(dbContext, parameters...)
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
	gp cmds.Processor) error {

	// use a statement so that when using mysql, we get native types back
	stmt, err := db.PrepareNamedContext(dbContext, query)
	if err != nil {
		return errors.Wrapf(err, "Could not prepare query: %s", query)
	}

	rows, err := stmt.QueryxContext(dbContext, parameters)
	if err != nil {
		return errors.Wrapf(err, "Could not execute query: %s", query)
	}

	return processQueryResults(rows, gp)
}

func processQueryResults(rows *sqlx.Rows, gp cmds.Processor) error {
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
	ps map[string]interface{},
	gp cmds.Processor) error {

	query, err := s.RenderQuery(ctx, ps, db)
	if err != nil {
		return err
	}
	return RunQueryIntoGlaze(ctx, db, query, []interface{}{}, gp)
}

type SqlCommandLoader struct {
	DBConnectionFactory DBConnectionFactory
}

func (scl *SqlCommandLoader) LoadCommandAliasFromYAML(s io.Reader, options ...alias.Option) ([]*alias.CommandAlias, error) {
	return loaders.LoadCommandAliasFromYAML(s, options...)
}

func (scl *SqlCommandLoader) LoadCommandFromYAML(
	s io.Reader,
	options ...cmds.CommandDescriptionOption,
) ([]cmds.Command, error) {
	scd := &SqlCommandDescription{}
	err := yaml.NewDecoder(s).Decode(scd)
	if err != nil {
		return nil, err
	}

	options_ := []cmds.CommandDescriptionOption{
		cmds.WithShort(scd.Short),
		cmds.WithLong(scd.Long),
		cmds.WithFlags(scd.Flags...),
		cmds.WithArguments(scd.Arguments...),
		cmds.WithLayers(scd.Layers...),
		cmds.WithLayout(scd.Layout),
	}
	options_ = append(options_, options...)

	sq, err := NewSqlCommand(
		cmds.NewCommandDescription(
			scd.Name,
		),
		WithDbConnectionFactory(scl.DBConnectionFactory),
		WithQuery(scd.Query),
		WithSubQueries(scd.SubQueries),
	)
	if err != nil {
		return nil, err
	}

	for _, option := range options_ {
		option(sq.Description())
	}

	if !sq.IsValid() {
		return nil, errors.New("Invalid command")
	}

	return []cmds.Command{sq}, nil
}

func LoadSqletonCommandFromYAML(
	s io.Reader,
	factory DBConnectionFactory,
	options ...cmds.CommandDescriptionOption) (SqletonCommand, error) {
	loader := &SqlCommandLoader{
		DBConnectionFactory: factory,
	}

	cmds_, err := loader.LoadCommandFromYAML(s, options...)
	if err != nil {
		return nil, err
	}

	if len(cmds_) != 1 {
		return nil, errors.New("expected only one command")
	}

	sqletonCommand := cmds_[0].(SqletonCommand)

	return sqletonCommand, nil

}
