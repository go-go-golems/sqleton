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
	"github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/pkg/errors"
)

type QueriesCommand struct {
	*glazed_cmds.CommandDescription
	commands []glazed_cmds.Command
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

	for _, command := range q.commands {
		description := command.Description()
		obj := types.NewRow(
			types.MRP("name", description.Name),
			types.MRP("short", description.Short),
			types.MRP("long", description.Long),
			types.MRP("source", description.Source),
			types.MRP("type", "unknown"),
			types.MRP("parents", description.Parents),
		)
		switch c := command.(type) {
		case *cmds.SqlCommand:
			obj.Set("query", c.Query)
			obj.Set("type", "sql")
		case *alias.CommandAlias:
			obj.Set("aliasFor", c.AliasFor)
			obj.Set("type", "alias")
		default:
		}
		err := gp.AddRow(ctx, obj)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewQueriesCommand(
	allCommands []glazed_cmds.Command,
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
		glazed_cmds.WithLayersList(glazeParameterLayer),
	}, options...)

	return &QueriesCommand{
		commands: allCommands,
		CommandDescription: glazed_cmds.NewCommandDescription(
			"queries",
			options_...,
		),
	}, nil
}
