package pkg

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/glazed/pkg/middlewares"
)

func RunQueryIntoGlaze(
	dbContext context.Context,
	db *sqlx.DB,
	query string,
	gp *cli.GlazeProcessor) error {
	rows, err := db.QueryxContext(dbContext, query)
	if err != nil {
		return errors.Wrapf(err, "Could not execute query: %s", query)
	}

	// we need a way to order the columns
	cols, err := rows.Columns()
	if err != nil {
		return errors.Wrapf(err, "Could not get columns")
	}

	gp.OutputFormatter().AddTableMiddleware(middlewares.NewReorderColumnOrderMiddleware(cols))
	// add support for renaming columns (at least to lowercase)
	// https://github.com/wesen/glazed/issues/27

	for rows.Next() {
		row := map[string]interface{}{}
		err = rows.MapScan(row)
		if err != nil {
			return errors.Wrapf(err, "Could not scan row")
		}

		for key, value := range row {
			switch value := value.(type) {
			case []byte:
				row[key] = string(value)
			}
		}

		err = gp.ProcessInputObject(row)
		if err != nil {
			return errors.Wrapf(err, "Could not process input object")
		}
	}

	return nil
}
