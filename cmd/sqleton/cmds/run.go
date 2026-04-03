package cmds

import (
	"context"
	"io"
	"os"

	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	schema "github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/sqleton/pkg/flags"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/spf13/cobra"
)

type RunCommand struct {
	*cmds.CommandDescription
	dbConnectionFactory sql.DBConnectionFactory
}

var _ cmds.GlazeCommand = (*RunCommand)(nil)

type RunSettings struct {
	InputFiles []string `glazed:"input-files"`
}

func (c *RunCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {

	s := &RunSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
		return err
	}
	ss := &flags.SqlHelpersSettings{}
	if err := parsedValues.DecodeSectionInto(flags.SqlHelpersSlug, ss); err != nil {
		return errors.Wrap(err, "could not initialize sql-helpers settings")
	}

	db, err := c.dbConnectionFactory(ctx, parsedValues)
	if err != nil {
		return errors.Wrap(err, "could not open database")
	}
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	err = db.PingContext(ctx)
	if err != nil {
		return errors.Wrapf(err, "Could not ping database")
	}

	for _, arg := range s.InputFiles {
		query := ""

		if arg == "-" {
			inBytes, err := io.ReadAll(os.Stdin)
			cobra.CheckErr(err)
			query = string(inBytes)
		} else {
			// read file
			queryBytes, err := os.ReadFile(arg)
			cobra.CheckErr(err)

			query = string(queryBytes)
		}

		if ss.Explain {
			query = "EXPLAIN " + query
		}

		// TODO(2022-12-20, manuel): collect named parameters here, maybe through prerun?
		// See: https://github.com/wesen/sqleton/issues/40
		err = sql.RunNamedQueryIntoGlaze(ctx, db, query, map[string]interface{}{}, gp)
		cobra.CheckErr(err)
	}

	return nil
}

func NewRunCommand(
	dbConnectionFactory sql.DBConnectionFactory,
	options ...cmds.CommandDescriptionOption,
) (*RunCommand, error) {
	glazedSection, err := settings.NewGlazedSection()
	if err != nil {
		return nil, errors.Wrap(err, "could not create glazed section")
	}
	sqlHelpersSection, err := flags.NewSqlHelpersParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL helpers section")
	}

	options_ := append([]cmds.CommandDescriptionOption{
		cmds.WithShort("Run a SQL query from sql files"),
		cmds.WithArguments(
			fields.New(
				"input-files",
				fields.TypeStringList,
				fields.WithRequired(true),
			),
		),
		cmds.WithSections(
			glazedSection,
			sqlHelpersSection,
		),
	}, options...)

	return &RunCommand{
		dbConnectionFactory: dbConnectionFactory,
		CommandDescription: cmds.NewCommandDescription(
			"run",
			options_...,
		),
	}, nil
}
