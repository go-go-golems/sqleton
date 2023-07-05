package pkg

import (
	"github.com/go-go-golems/clay/pkg/repositories/sql"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

func OpenDatabaseFromSqletonConnectionLayer(parsedLayers map[string]*layers.ParsedParameterLayer) (*sqlx.DB, error) {
	sqlConnectionLayer, ok := parsedLayers["sql-connection"]
	if !ok {
		return nil, errors.New("No sql-connection layer found")
	}
	dbtLayer, ok := parsedLayers["dbt"]
	if !ok {
		return nil, errors.New("No dbt layer found")
	}

	config, err2 := sql.NewConfigFromParsedLayers(sqlConnectionLayer, dbtLayer)
	if err2 != nil {
		return nil, err2
	}
	return config.Connect()
}
