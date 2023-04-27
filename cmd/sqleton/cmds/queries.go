package cmds

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/middlewares/table"
	"github.com/go-go-golems/glazed/pkg/processor"
	sqleton "github.com/go-go-golems/sqleton/pkg"
)

type QueriesCommand struct {
	description *glazed_cmds.CommandDescription
	queries     []*sqleton.SqlCommand
	aliases     []*alias.CommandAlias
}

func (q *QueriesCommand) Description() *glazed_cmds.CommandDescription {
	return q.description
}

func (q *QueriesCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp processor.Processor,
) error {
	gp.OutputFormatter().AddTableMiddleware(
		table.NewReorderColumnOrderMiddleware(
			[]string{"name", "short", "long", "source", "query"}),
	)

	for _, query := range q.queries {
		description := query.Description()
		obj := map[string]interface{}{
			"name":   description.Name,
			"short":  description.Short,
			"long":   description.Long,
			"query":  query.Query,
			"source": description.Source,
		}
		err := gp.ProcessInputObject(ctx, obj)
		if err != nil {
			return err
		}
	}

	for _, alias := range q.aliases {
		obj := map[string]interface{}{
			"name":     alias.Name,
			"aliasFor": alias.AliasFor,
			"source":   alias.Source,
		}
		err := gp.ProcessInputObject(ctx, obj)
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
	glazeParameterLayer, err := cli.NewGlazedParameterLayers(
		cli.WithFieldsFiltersParameterLayerOptions(
			layers.WithDefaults(
				&cli.FieldsFilterFlagsDefaults{
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
		description: glazed_cmds.NewCommandDescription(
			"queries",
			options_...,
		),
	}, nil
}
