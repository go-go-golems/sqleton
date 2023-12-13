package cmds

import (
	"context"
	"fmt"
	clay_sql "github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/sqleton/pkg/flags"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"strings"
)

type SqletonCommand interface {
	RunQueryIntoGlaze(
		ctx context.Context,
		db *sqlx.DB,
		parameters map[string]interface{},
		gp middlewares.TableProcessor,
	) error
	RenderQuery(parameters map[string]interface{}) (string, error)
}

var _ cmds.GlazeCommand = (*SqlCommand)(nil)
var _ cmds.CommandWithMetadata = (*SqlCommand)(nil)

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
	renderedQuery       string
}

func (s *SqlCommand) Metadata(ctx context.Context, parsedLayers map[string]*layers.ParsedParameterLayer, ps map[string]interface{}) (map[string]interface{}, error) {
	db, err := s.dbConnectionFactory(parsedLayers)
	if err != nil {
		return nil, err
	}
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	err = db.PingContext(ctx)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not ping database")
	}

	query, err := s.RenderQuery(ctx, ps, db)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not generate query")
	}

	return map[string]interface{}{
		"query": query,
	}, nil
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
	sqlConnectionParameterLayer, err := clay_sql.NewSqlConnectionParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL connection parameter layer")
	}
	dbtParameterLayer, err := clay_sql.NewDbtParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create dbt parameter layer")
	}
	sqlHelpersParameterLayer, err := flags.NewSqlHelpersParameterLayer()
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

	s.renderedQuery, err = s.RenderQuery(ctx, ps, db)
	if err != nil {
		return errors.Wrapf(err, "Could not generate query")
	}

	printQuery, _ := ps["print-query"].(bool)
	if printQuery {
		fmt.Println(s.renderedQuery)
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
	ret, err := clay_sql.RenderQuery(ctx, db, s.Query, s.SubQueries, ps)
	if err != nil {
		return "", errors.Wrap(err, "Could not render query")
	}

	return ret, nil
}

func (s *SqlCommand) RunQueryIntoGlaze(
	ctx context.Context,
	db *sqlx.DB,
	ps map[string]interface{},
	gp middlewares.Processor) error {

	return clay_sql.RunQueryIntoGlaze(ctx, db, s.renderedQuery, []interface{}{}, gp)
}
