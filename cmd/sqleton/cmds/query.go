package cmds

import (
	"context"

	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	schema "github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/jmoiron/sqlx"
)

type QueryCommand struct {
	dbConnectionFactory sql.DBConnectionFactory
	*cmds.CommandDescription
}

var _ cmds.GlazeCommand = (*QueryCommand)(nil)

func NewQueryCommand(
	dbConnectionFactory sql.DBConnectionFactory,
	options ...cmds.CommandDescriptionOption,
) (*QueryCommand, error) {
	glazedSection, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}
	options_ := append([]cmds.CommandDescriptionOption{
		cmds.WithShort("Run a SQL query passed as a CLI argument"),
		cmds.WithArguments(fields.New(
			"query",
			fields.TypeString,
			fields.WithHelp("The SQL query to run"),
			fields.WithRequired(true),
		),
		),
		cmds.WithSections(glazedSection),
	}, options...)

	return &QueryCommand{
		dbConnectionFactory: dbConnectionFactory,
		CommandDescription:  cmds.NewCommandDescription("query", options_...),
	}, nil
}

type QuerySettings struct {
	Query string `glazed:"query"`
}

func (q *QueryCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	s := &QuerySettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}

	db, err := q.dbConnectionFactory(ctx, parsedValues)
	if err != nil {
		return err
	}
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	err = db.PingContext(ctx)
	if err != nil {
		return err
	}

	err = sql.RunNamedQueryIntoGlaze(ctx, db, s.Query, map[string]interface{}{}, gp)
	if err != nil {
		return err
	}

	return nil
}
