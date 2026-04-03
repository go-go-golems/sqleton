package cmds

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	sql2 "github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	schema "github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	cmds2 "github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/go-go-golems/sqleton/pkg/flags"
	"github.com/huandu/go-sqlbuilder"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

//go:embed "flags/select.yaml"
var selectFlagsYaml []byte

const SelectSlug = "select"

func NewSelectSection() (*schema.SectionImpl, error) {
	ret, err := schema.NewSectionFromYAML(selectFlagsYaml)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize select parameter layer")
	}
	return ret, nil
}

type SelectCommand struct {
	*cmds.CommandDescription
	dbConnectionFactory sql2.DBConnectionFactory
}

type SelectCommandSettings struct {
	Columns     []string `glazed:"columns"`
	Limit       int      `glazed:"limit"`
	Offset      int      `glazed:"offset"`
	Count       bool     `glazed:"count"`
	Where       []string `glazed:"where"`
	OrderBy     string   `glazed:"order-by"`
	Distinct    bool     `glazed:"distinct"`
	Table       string   `glazed:"table"`
	CreateQuery string   `glazed:"create-query"`
}

var _ cmds.GlazeCommand = (*SelectCommand)(nil)

func (sc *SelectCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	s := &SelectCommandSettings{}
	if err := parsedValues.DecodeSectionInto(SelectSlug, s); err != nil {
		return err
	}

	ss := &flags.SqlHelpersSettings{}
	if err := parsedValues.DecodeSectionInto(flags.SqlHelpersSlug, ss); err != nil {
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

	for _, where := range s.Where {
		sb = sb.Where(where)
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
		if len(s.Where) > 0 {
			short = fmt.Sprintf("Select"+" from %s where %s", s.Table, strings.Join(s.Where, " AND "))
		}

		queryFlags := []*fields.Definition{}
		if len(s.Where) == 0 {
			queryFlags = append(queryFlags, fields.New("where", fields.TypeStringList))
		}
		if !s.Count {
			queryFlags = append(queryFlags, fields.New(
				"limit",
				fields.TypeInteger,
				fields.WithHelp(fmt.Sprintf("Limit the number of rows (default: %d), set to 0 to disable", s.Limit)),
				fields.WithDefault(s.Limit),
			))
			queryFlags = append(queryFlags, fields.New(
				"offset",
				fields.TypeInteger,
				fields.WithHelp(fmt.Sprintf("Offset the number of rows (default: %d)", s.Offset)),
				fields.WithDefault(s.Offset),
			))
			queryFlags = append(queryFlags, fields.New(
				"distinct",
				fields.TypeBool,
				fields.WithHelp(fmt.Sprintf("Whether to select distinct rows (default: %t)", s.Distinct)),
				fields.WithDefault(s.Distinct),
			))

			orderByHelp := "Order by"
			orderByOptions := []fields.Option{
				fields.WithHelp(orderByHelp),
			}
			if s.OrderBy != "" {
				orderByHelp = fmt.Sprintf("Order by (default: %s)", s.OrderBy)
				orderByOptions = append(orderByOptions, fields.WithHelp(orderByHelp), fields.WithDefault(s.OrderBy))
			}
			queryFlags = append(queryFlags, fields.New("order_by", fields.TypeString, orderByOptions...))
		}

		sb := &strings.Builder{}
		_, _ = fmt.Fprintf(sb, "SELECT ")
		if !s.Count {
			_, _ = fmt.Fprintf(sb, "{{ if .distinct }}DISTINCT{{ end }} ")
		}
		_, _ = fmt.Fprintf(sb, "%s FROM %s", strings.Join(s.Columns, ", "), s.Table)
		if len(s.Where) > 0 {
			_, _ = fmt.Fprintf(sb, " WHERE %s", strings.Join(s.Where, " AND "))
		} else {
			_, _ = fmt.Fprintf(sb, "\nWHERE 1=1\n{{ range .where  }}  AND {{.}} {{ end }}")
		}

		_, _ = fmt.Fprintf(sb, "\n{{ if .order_by }} ORDER BY {{ .order_by }}{{ end }}")
		_, _ = fmt.Fprintf(sb, "\n{{ if .limit }} LIMIT {{ .limit }}{{ end }}")
		_, _ = fmt.Fprintf(sb, "\nOFFSET {{ .offset }}")

		query := sb.String()
		sqlFile, err := cmds2.MarshalSpecToSQLFile(&cmds2.SqlCommandSpec{
			Name:  s.CreateQuery,
			Short: short,
			Flags: queryFlags,
			Query: query,
		})
		if err != nil {
			return err
		}

		fmt.Print(sqlFile)
		return nil
	}

	query, queryArgs := sb.Build()

	if ss.PrintQuery {
		fmt.Println(query)
		if len(queryArgs) > 0 {
			fmt.Println("Args:")
			fmt.Println(queryArgs)
		}
		return nil
	}

	db, err := sc.dbConnectionFactory(ctx, parsedValues)
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
	glazedSection, err := settings.NewGlazedSection()
	if err != nil {
		return nil, errors.Wrap(err, "could not create glazed section")
	}
	sqlHelpersSection, err := flags.NewSqlHelpersParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL helpers section")
	}
	selectSection, err := NewSelectSection()
	if err != nil {
		return nil, errors.Wrap(err, "could not create select section")
	}

	options_ := append([]cmds.CommandDescriptionOption{
		cmds.WithShort("Select" + " all columns from a table"),
		cmds.WithSections(
			selectSection,
			glazedSection,
			sqlHelpersSection,
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
