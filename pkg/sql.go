package pkg

import (
	"context"
	"fmt"
	"github.com/go-go-golems/clay/pkg/repositories/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/helpers/templating"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
)

type SqletonCommand interface {
	cmds.GlazeCommand
	RunQueryIntoGlaze(
		ctx context.Context,
		db *sqlx.DB,
		parameters map[string]interface{},
		gp middlewares.TableProcessor,
	) error
	RenderQuery(parameters map[string]interface{}) (string, error)
}

type SqlCommandDescription struct {
	Name      string                            `yaml:"name"`
	Short     string                            `yaml:"short"`
	Long      string                            `yaml:"long,omitempty"`
	Layout    []*layout.Section                 `yaml:"layout,omitempty"`
	Flags     []*parameters.ParameterDefinition `yaml:"flags,omitempty"`
	Arguments []*parameters.ParameterDefinition `yaml:"arguments,omitempty"`
	Layers    []layers.ParameterLayer           `yaml:"layers,omitempty"`

	SubQueries map[string]string `yaml:"subqueries,omitempty"`
	Query      string            `yaml:"query"`
}

type DBConnectionFactory func(parsedLayers map[string]*layers.ParsedParameterLayer) (*sqlx.DB, error)

// SqlCommand describes a command line command that runs a query
type SqlCommand struct {
	*cmds.CommandDescription
	Query               string              `yaml:"query"`
	SubQueries          map[string]string   `yaml:"subqueries,omitempty"`
	dbConnectionFactory DBConnectionFactory `yaml:"-"`
}

func (s *SqlCommand) String() string {
	return fmt.Sprintf("SqlCommand{Name: %s, Parents: %s}", s.Name, strings.Join(s.Parents, " "))
}

func (s *SqlCommand) ToYAML(w io.Writer) error {
	enc := yaml.NewEncoder(w)
	defer func(enc *yaml.Encoder) {
		_ = enc.Close()
	}(enc)

	return enc.Encode(s)
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
	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, errors.Wrap(err, "could not create Glazed parameter layer")
	}
	sqlConnectionParameterLayer, err := sql.NewSqlConnectionParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL connection parameter layer")
	}
	dbtParameterLayer, err := sql.NewDbtParameterLayer()
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
		CommandDescription: description,
		SubQueries:         make(map[string]string),
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
	gp middlewares.Processor,
) error {
	if s.dbConnectionFactory == nil {
		return fmt.Errorf("dbConnectionFactory is not set")
	}

	// at this point, the factory can probably be passed the sql-connection parsed layer
	db, err := s.dbConnectionFactory(parsedLayers)
	if err != nil {
		return err
	}
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

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

func (s *SqlCommand) RenderQueryFull(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
) (string, error) {
	if s.dbConnectionFactory == nil {
		return "", fmt.Errorf("dbConnectionFactory is not set")
	}

	// at this point, the factory can probably be passed the sql-connection parsed layer
	db, err := s.dbConnectionFactory(parsedLayers)
	if err != nil {
		return "", err
	}
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	err = db.PingContext(ctx)
	if err != nil {
		return "", errors.Wrapf(err, "Could not ping database")
	}

	query, err := s.RenderQuery(ctx, ps, db)
	if err != nil {
		return "", errors.Wrapf(err, "Could not generate query")
	}
	return query, nil
}

func (s *SqlCommand) Description() *cmds.CommandDescription {
	return s.CommandDescription
}

func (s *SqlCommand) IsValid() bool {
	return s.Name != "" && s.Query != "" && s.Short != ""
}

func (s *SqlCommand) RenderQuery(
	ctx context.Context,
	ps map[string]interface{},
	db *sqlx.DB,
) (string, error) {
	t2 := sql.CreateTemplate(ctx, s.SubQueries, ps, db)

	t, err := t2.Parse(s.Query)
	if err != nil {
		return "", errors.Wrap(err, "Could not parse query template")
	}

	return templating.RenderTemplate(t, ps)
}

func (s *SqlCommand) RunQueryIntoGlaze(
	ctx context.Context,
	db *sqlx.DB,
	ps map[string]interface{},
	gp middlewares.Processor) error {

	query, err := s.RenderQuery(ctx, ps, db)
	if err != nil {
		return err
	}
	return sql.RunQueryIntoGlaze(ctx, db, query, []interface{}{}, gp)
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
		cmds.WithLayout(&layout.Layout{
			Sections: scd.Layout,
		}),
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
