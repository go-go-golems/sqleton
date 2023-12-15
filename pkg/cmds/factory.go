package cmds

import (
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/parka/pkg/handlers"
)

func NewRepositoryFactory() handlers.RepositoryFactory {
	loader := &SqlCommandLoader{
		DBConnectionFactory: sql.OpenDatabaseFromDefaultSqlConnectionLayer,
	}

	return handlers.NewRepositoryFactoryFromReaderLoaders(loader)
}
