package cmds

import (
	"context"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/middlewares/row"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	sqleton "github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/pkg/errors"
)

type QueriesCommand struct {
	*glazed_cmds.CommandDescription
	queries []*sqleton.SqlCommand
	aliases []*alias.CommandAlias
}

var _ glazed_cmds.GlazeCommand = (*QueriesCommand)(nil)

func (q *QueriesCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	tableProcessor, ok := gp.(*middlewares.TableProcessor)
	if !ok {
		return errors.New("expected a table processor")
	}

	tableProcessor.AddRowMiddleware(
		row.NewReorderColumnOrderMiddleware(
			[]string{"name", "short", "long", "source", "query"}),
	)

	for _, query := range q.queries {
		description := query.Description()
		obj := types.NewRow(
			types.MRP("name", description.Name),
			types.MRP("short", description.Short),
			types.MRP("long", description.Long),
			types.MRP("query", query.Query),
			types.MRP("source", description.Source),
		)
		err := gp.AddRow(ctx, obj)
		if err != nil {
			return err
		}
	}

	for _, alias_ := range q.aliases {
		obj := types.NewRow(
			types.MRP("name", alias_.Name),
			types.MRP("aliasFor", alias_.AliasFor),
			types.MRP("source", alias_.Source),
		)
		err := gp.AddRow(ctx, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewQueriesCommand(
	allQueries []*sqleton.SqlCommand,
	aliases []*alias.CommandAlias,
	options ...glazed_cmds.CommandDescriptionOption,
) (*QueriesCommand, error) {
	glazeParameterLayer, err := settings.NewGlazedParameterLayers(
		settings.WithFieldsFiltersParameterLayerOptions(
			layers.WithDefaults(
				&settings.FieldsFilterFlagsDefaults{
					Fields: []string{"name", "short", "source"},
				},
			),
		),
	)
	if err != nil {
		return nil, err
	}

	options_ := append([]glazed_cmds.CommandDescriptionOption{
		glazed_cmds.WithShort("Commands related to sqleton queries"),
		glazed_cmds.WithLayers(glazeParameterLayer),
	}, options...)

	return &QueriesCommand{
		queries: allQueries,
		aliases: aliases,
		CommandDescription: glazed_cmds.NewCommandDescription(
			"queries",
			options_...,
		),
	}, nil
}
