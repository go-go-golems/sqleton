package cmds

import (
	"context"
	_ "embed"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/processor"
	"github.com/go-go-golems/glazed/pkg/settings"
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

type SelectCommandSettings struct {
	Columns  []string `glazed.parameter:"columns"`
	Limit    int      `glazed.parameter:"limit"`
	Offset   int      `glazed.parameter:"offset"`
	Count    bool     `glazed.parameter:"count"`
	Where    string   `glazed.parameter:"where"`
	OrderBy  string   `glazed.parameter:"order-by"`
	Distinct bool     `glazed.parameter:"distinct"`
	Table    string   `glazed.parameter:"table"`
}

func (sc *SelectCommand) Description() *cmds.CommandDescription {
	return sc.description
}

func (sc *SelectCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp processor.Processor,
) error {
	s := &SelectCommandSettings{}

	// pass in ps so we also get the `table` arguments
	err := parameters.InitializeStructFromParameters(s, ps)
	if err != nil {
		return errors.Wrap(err, "Failed to initialize select command settings")
	}

	sb := sqlbuilder.NewSelectBuilder()
	sb = sb.From(s.Table)

	if s.Count {
		countColumns := strings.Join(s.Columns, ", ")
		if s.Distinct {
			countColumns = "DISTINCT " + countColumns
		}
		s.Columns = []string{sb.As(fmt.Sprintf("COUNT(%s)", countColumns), "count")}
	} else {
		if len(s.Columns) == 0 {
			s.Columns = []string{"*"}
		}
	}
	sb = sb.Select(s.Columns...)
	if s.Distinct && !s.Count {
		sb = sb.Distinct()
	}

	if s.Where != "" {
		sb = sb.Where(s.Where)
	}

	if s.Limit > 0 && !s.Count {
		sb = sb.Limit(s.Limit)
	}
	if s.Offset > 0 {
		sb = sb.Offset(s.Offset)
	}
	if s.OrderBy != "" {
		sb = sb.OrderBy(s.OrderBy)
	}

	createQuery, _ := ps["create-query"].(string)

	if createQuery != "" {
		short := fmt.Sprintf("Select"+" columns from %s", s.Table)
		if s.Count {
			short = fmt.Sprintf("Count all rows from %s", s.Table)
		}
		if s.Where != "" {
			short = fmt.Sprintf("Select"+" from %s where %s", s.Table, s.Where)
		}

		flags := []*parameters.ParameterDefinition{}
		if s.Where == "" {
			flags = append(flags, &parameters.ParameterDefinition{
				Name: "where",
				Type: parameters.ParameterTypeString,
			})
		}
		if !s.Count {
			flags = append(flags, &parameters.ParameterDefinition{
				Name:    "limit",
				Type:    parameters.ParameterTypeInteger,
				Help:    fmt.Sprintf("Limit the number of rows (default: %d), set to 0 to disable", s.Limit),
				Default: s.Limit,
			})
			flags = append(flags, &parameters.ParameterDefinition{
				Name:    "offset",
				Type:    parameters.ParameterTypeInteger,
				Help:    fmt.Sprintf("Offset the number of rows (default: %d)", s.Offset),
				Default: s.Offset,
			})
			flags = append(flags, &parameters.ParameterDefinition{
				Name:    "distinct",
				Type:    parameters.ParameterTypeBool,
				Help:    fmt.Sprintf("Whether to select distinct rows (default: %t)", s.Distinct),
				Default: s.Distinct,
			})

			orderByHelp := "Order by"
			var orderDefault interface{}
			if s.OrderBy != "" {
				orderByHelp = fmt.Sprintf("Order by (default: %s)", s.OrderBy)
				orderDefault = s.OrderBy
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
		if !s.Count {
			_, _ = fmt.Fprintf(sb, "{{ if .distinct }}DISTINCT{{ end }} ")
		}
		_, _ = fmt.Fprintf(sb, "%s FROM %s", strings.Join(s.Columns, ", "), s.Table)
		if s.Where != "" {
			_, _ = fmt.Fprintf(sb, " WHERE %s", s.Where)
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
			pkg.WithDbConnectionFactory(pkg.OpenDatabaseFromSqletonConnectionLayer),
			pkg.WithQuery(query),
		)
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

	db, err := sc.dbConnectionFactory(parsedLayers)
	if err != nil {
		return err
	}
	defer db.Close()

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
	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
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
