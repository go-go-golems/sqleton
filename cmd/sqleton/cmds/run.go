package cmds

import (
	"context"
	"fmt"
	"github.com/go-go-golems/clay/pkg/repositories/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	cli "github.com/go-go-golems/glazed/pkg/settings"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"

	"github.com/go-go-golems/sqleton/pkg"
	"github.com/spf13/cobra"
	"io"
	"os"
)

type RunCommand struct {
	description         *cmds.CommandDescription
	dbConnectionFactory pkg.DBConnectionFactory
}

func (c *RunCommand) Run(
	ctx context.Context,
	parsedLayers map[string]*layers.ParsedParameterLayer,
	ps map[string]interface{},
	gp middlewares.Processor) error {
	inputFiles, ok := ps["input-files"].([]string)
	if !ok {
		return fmt.Errorf("input-files is not a string list")
	}

	db, err := c.dbConnectionFactory(parsedLayers)
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

	explain, _ := ps["explain"].(bool)

	for _, arg := range inputFiles {
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

		if explain {
			query = "EXPLAIN " + query
		}

		// TODO(2022-12-20, manuel): collect named parameters here, maybe through prerun?
		// See: https://github.com/wesen/sqleton/issues/40
		err = sql.RunNamedQueryIntoGlaze(ctx, db, query, map[string]interface{}{}, gp)
		cobra.CheckErr(err)
	}

	return nil
}

func (c *RunCommand) Description() *cmds.CommandDescription {
	return c.description
}

func NewRunCommand(
	dbConnectionFactory pkg.DBConnectionFactory,
	options ...cmds.CommandDescriptionOption,
) (*RunCommand, error) {
	glazedParameterLayer, err := cli.NewGlazedParameterLayers()
	if err != nil {
		return nil, errors.Wrap(err, "could not create Glazed parameter layer")
	}
	sqlHelpersParameterLayer, err := pkg.NewSqlHelpersParameterLayer()
	if err != nil {
		return nil, errors.Wrap(err, "could not create SQL helpers parameter layer")
	}

	options_ := append([]cmds.CommandDescriptionOption{
		cmds.WithShort("Run a SQL query from sql files"),
		cmds.WithArguments(
			parameters.NewParameterDefinition(
				"input-files",
				parameters.ParameterTypeStringList,
				parameters.WithRequired(true),
			),
		),
		cmds.WithLayers(
			glazedParameterLayer,
			sqlHelpersParameterLayer,
		),
	}, options...)

	return &RunCommand{
		dbConnectionFactory: dbConnectionFactory,
		description: cmds.NewCommandDescription(
			"run",
			options_...,
		),
	}, nil
}
