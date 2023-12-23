package cmds

import (
	"context"
	_ "embed"
	"fmt"
	sql2 "github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	cmds2 "github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/go-go-golems/sqleton/pkg/flags"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jmoiron/sqlx"
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

const SelectSlug = "select"

type SelectCommand struct {
	*cmds.CommandDescription
	dbConnectionFactory sql2.DBConnectionFactory
}

type SelectCommandSettings struct {
	Columns     []string `glazed.parameter:"columns"`
	Limit       int      `glazed.parameter:"limit"`
	Offset      int      `glazed.parameter:"offset"`
	Count       bool     `glazed.parameter:"count"`
	Where       string   `glazed.parameter:"where"`
	OrderBy     string   `glazed.parameter:"order-by"`
	Distinct    bool     `glazed.parameter:"distinct"`
	Table       string   `glazed.parameter:"table"`
	CreateQuery string   `glazed.parameter:"create-query"`
}

var _ cmds.GlazeCommand = (*SelectCommand)(nil)

func (sc *SelectCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	s := &SelectCommandSettings{}
	err := parsedLayers.InitializeStruct(SelectSlug, s)
	if err != nil {
		return err
	}

	ss := &flags.SqlHelpersSettings{}
	err = parsedLayers.InitializeStruct(flags.SqlHelpersSlug, ss)
	if err != nil {
		return errors.Wrap(err, "could not initialize sql-helpers settings")
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

	if s.CreateQuery != "" {
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
		sqlCommand, err := cmds2.NewSqlCommand(
			cmds.NewCommandDescription(s.CreateQuery,
				cmds.WithShort(short), cmds.WithFlags(flags...)),
			cmds2.WithDbConnectionFactory(sql2.OpenDatabaseFromDefaultSqlConnectionLayer),
			cmds2.WithQuery(query),
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

	if ss.PrintQuery {
		fmt.Println(query)
		fmt.Println(queryArgs)
		return nil
	}

	db, err := sc.dbConnectionFactory(parsedLayers)
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

	err = sql2.RunQueryIntoGlaze(ctx, db, query, queryArgs, gp)
	if err != nil {
		return err
	}
	return nil
}

func NewSelectCommand(
	dbConnectionFactory sql2.DBConnectionFactory,
	options ...cmds.CommandDescriptionOption,
) (*SelectCommand, error) {
	glazedParameterLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, errors.Wrap(err, "could not create Glazed parameter layer")
	}
	sqlHelpersParameterLayer, err := flags.NewSqlHelpersParameterLayer()
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
		CommandDescription: cmds.NewCommandDescription(
			"select",
			options_...,
		),
	}, nil
}
