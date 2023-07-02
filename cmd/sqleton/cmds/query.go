package cmds

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/jmoiron/sqlx"
)

type QueryCommand struct {
	dbConnectionFactory pkg.DBConnectionFactory
	description         *cmds.CommandDescription
}

func NewQueryCommand(
	dbConnectionFactory pkg.DBConnectionFactory,
	options ...cmds.CommandDescriptionOption,
) (*QueryCommand, error) {
	glazeParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}
	options_ := append([]cmds.CommandDescriptionOption{
		cmds.WithShort("Run a SQL query passed as a CLI argument"),
		cmds.WithArguments(parameters.NewParameterDefinition(
			"query",
			parameters.ParameterTypeString,
			parameters.WithHelp("The SQL query to run"),
			parameters.WithRequired(true),
		),
		),
		cmds.WithLayers(glazeParameterLayer),
	}, options...)

	return &QueryCommand{
		dbConnectionFactory: dbConnectionFactory,
		description:         cmds.NewCommandDescription("query", options_...),
	}, nil
}

func (q *QueryCommand) Description() *cmds.CommandDescription {
	return q.description
}

func (q *QueryCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor,
) error {
	query := ps["query"].(string)

	db, err := q.dbConnectionFactory(parsedLayers)
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

	err = pkg.RunNamedQueryIntoGlaze(ctx, db, query, map[string]interface{}{}, gp)
	if err != nil {
		return err
	}

	return nil
}
