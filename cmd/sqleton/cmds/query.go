package cmds

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/sqleton/pkg"
)

type QueryCommand struct {
	dbConnectionFactory pkg.DBConnectionFactory
	description         *cmds.CommandDescription
}

func NewQueryCommand(
	dbConnectionFactory pkg.DBConnectionFactory,
	options ...cmds.CommandDescriptionOption,
) (*QueryCommand, error) {
	options_ := append([]cmds.CommandDescriptionOption{
		cmds.WithShort("Run a SQL query passed as a CLI argument"),
		cmds.WithArguments(parameters.NewParameterDefinition(
			"query",
			parameters.ParameterTypeString,
			parameters.WithHelp("The SQL query to run"),
			parameters.WithRequired(true),
		),
		),
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
	parsedLayers []*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp *cmds.GlazeProcessor,
) error {
	query := ps["query"].(string)

	db, err := q.dbConnectionFactory()
	if err != nil {
		return err
	}

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
