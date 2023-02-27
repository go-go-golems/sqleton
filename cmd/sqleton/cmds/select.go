package cmds

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/huandu/go-sqlbuilder"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"strings"
)

//go:embed "flags/select.yaml"
var selectFlagsYaml []byte

func NewSelectParameterLayer() (*layers.ParameterLayerImpl, error) {
	ret := &layers.ParameterLayerImpl{}
	err := ret.LoadFromYAML(selectFlagsYaml)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize select parameter layer")
	}
	return ret, nil
}

type SelectCommand struct {
	description         *cmds.CommandDescription
	dbConnectionFactory pkg.DBConnectionFactory
}

func (sc *SelectCommand) Description() *cmds.CommandDescription {
	return sc.description
}

func (sc *SelectCommand) Run(
	ctx context.Context,
	parsedLayers []*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp *cmds.GlazeProcessor,
) error {
	// TODO(2023-02-27) Use the SelectParameterLayer to parse this
	columns, _ := ps["columns"].([]string)
	limit, _ := ps["limit"].(int)
	offset, _ := ps["offset"].(int)
	count, _ := ps["count"].(bool)
	where, _ := ps["where"].(string)
	order, _ := ps["order-by"].(string)
	distinct, _ := ps["distinct"].(bool)
	table, _ := ps["table"].(string)

	sb := sqlbuilder.NewSelectBuilder()
	sb = sb.From(table)

	if count {
		countColumns := strings.Join(columns, ", ")
		if distinct {
			countColumns = "DISTINCT " + countColumns
		}
		columns = []string{sb.As(fmt.Sprintf("COUNT(%s)", countColumns), "count")}
	} else {
		if len(columns) == 0 {
			columns = []string{"*"}
		}
	}
	sb = sb.Select(columns...)
	if distinct && !count {
		sb = sb.Distinct()
	}

	if where != "" {
		sb = sb.Where(where)
	}

	if limit > 0 && !count {
		sb = sb.Limit(limit)
	}
	if offset > 0 {
		sb = sb.Offset(offset)
	}
	if order != "" {
		sb = sb.OrderBy(order)
	}

	createQuery, _ := ps["create-query"].(string)

	if createQuery != "" {
		short := fmt.Sprintf("Select"+" columns from %s", table)
		if count {
			short = fmt.Sprintf("Count all rows from %s", table)
		}
		if where != "" {
			short = fmt.Sprintf("Select"+" from %s where %s", table, where)
		}

		flags := []*parameters.ParameterDefinition{}
		if where == "" {
			flags = append(flags, &parameters.ParameterDefinition{
				Name: "where",
				Type: parameters.ParameterTypeString,
			})
		}
		if !count {
			flags = append(flags, &parameters.ParameterDefinition{
				Name:    "limit",
				Type:    parameters.ParameterTypeInteger,
				Help:    fmt.Sprintf("Limit the number of rows (default: %d), set to 0 to disable", limit),
				Default: limit,
			})
			flags = append(flags, &parameters.ParameterDefinition{
				Name:    "offset",
				Type:    parameters.ParameterTypeInteger,
				Help:    fmt.Sprintf("Offset the number of rows (default: %d)", offset),
				Default: offset,
			})
			flags = append(flags, &parameters.ParameterDefinition{
				Name:    "distinct",
				Type:    parameters.ParameterTypeBool,
				Help:    fmt.Sprintf("Whether to select distinct rows (default: %t)", distinct),
				Default: distinct,
			})

			orderByHelp := "Order by"
			var orderDefault interface{}
			if order != "" {
				orderByHelp = fmt.Sprintf("Order by (default: %s)", order)
				orderDefault = order
			}
			flags = append(flags, &parameters.ParameterDefinition{
				Name:    "order_by",
				Type:    parameters.ParameterTypeString,
				Help:    orderByHelp,
				Default: orderDefault,
			})
		}

		sb := &strings.Builder{}
		_, _ = fmt.Fprintf(sb, "SELECT ")
		if !count {
			_, _ = fmt.Fprintf(sb, "{{ if .distinct }}DISTINCT{{ end }} ")
		}
		_, _ = fmt.Fprintf(sb, "%s FROM %s", strings.Join(columns, ", "), table)
		if where != "" {
			_, _ = fmt.Fprintf(sb, " WHERE %s", where)
		} else {
			_, _ = fmt.Fprintf(sb, "\n{{ if .where  }}  WHERE {{.where}} {{ end }}")
		}

		_, _ = fmt.Fprintf(sb, "\n{{ if .order_by }} ORDER BY {{ .order_by }}{{ end }}")
		_, _ = fmt.Fprintf(sb, "\n{{ if .limit }} LIMIT {{ .limit }}{{ end }}")
		_, _ = fmt.Fprintf(sb, "\nOFFSET {{ .offset }}")

		query := sb.String()
		sqlCommand, err := pkg.NewSqlCommand(&cmds.CommandDescription{
			Name:  createQuery,
			Short: short,
			Flags: flags,
		},
			pkg.OpenDatabaseFromViper,
			query)
		if err != nil {
			return err
		}

		// marshal to yaml
		yamlBytes, err := yaml.Marshal(sqlCommand)
		if err != nil {
			return err
		}

		fmt.Println(string(yamlBytes))
		return nil
	}

	query, queryArgs := sb.Build()

	printQuery, _ := ps["print-query"].(bool)

	if printQuery {
		fmt.Println(query)
		fmt.Println(queryArgs)
		return nil
	}

	db, err := sc.dbConnectionFactory()
	if err != nil {
		return err
	}

	err = db.PingContext(ctx)
	if err != nil {
		return err
	}

	err = pkg.RunQueryIntoGlaze(ctx, db, query, queryArgs, gp)
	if err != nil {
		return err
	}
	return nil
}

func NewSelectCommand(
	dbConnectionFactory pkg.DBConnectionFactory,
	options ...cmds.CommandDescriptionOption,
) (*SelectCommand, error) {
	glazedParameterLayer, err := cli.NewGlazedParameterLayers()
	if err != nil {
		return nil, errors.Wrap(err, "could not create Glazed parameter layer")
	}
	sqlHelpersParameterLayer, err := pkg.NewSqlHelpersParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL helpers parameter layer")
	}
	selectParameterLayer, err := NewSelectParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create select parameter layer")
	}

	options_ := append([]cmds.CommandDescriptionOption{
		cmds.WithShort("Select" + " all columns from a table"),
		cmds.WithArguments(
			parameters.NewParameterDefinition(
				"table",
				parameters.ParameterTypeString,
				parameters.WithHelp("The table to select from"),
				parameters.WithRequired(true),
			),
		),
		cmds.WithLayers(
			selectParameterLayer,
			glazedParameterLayer,
			sqlHelpersParameterLayer,
		),
	}, options...)

	return &SelectCommand{
		dbConnectionFactory: dbConnectionFactory,
		description: cmds.NewCommandDescription(
			"select",
			options_...,
		),
	}, nil
}
