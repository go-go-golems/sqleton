package cmds

import (
	"context"
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
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
		cmds.WithLayersList(glazeParameterLayer),
	}, options...)

	return &QueryCommand{
		dbConnectionFactory: dbConnectionFactory,
		CommandDescription:  cmds.NewCommandDescription("query", options_...),
	}, nil
}

type QuerySettings struct {
	Query string `glazed.parameter:"query"`
}

func (q *QueryCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	d := parsedLayers.GetDefaultParameterLayer()
	s := &QuerySettings{}
	err := d.InitializeStruct(s)
	if err != nil {
		return err
	}

	db, err := q.dbConnectionFactory(ctx, parsedLayers)
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
