package cmds

import (
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/parka/pkg/handlers"
)

func NewRepositoryFactory() handlers.RepositoryFactory {
	loader := &SqlCommandLoader{
		DBConnectionFactory: sql.OpenDatabaseFromDefaultSqlConnectionLayer,
	}
	yamlFSLoader := loaders.NewFSFileCommandLoader(loader)

	return handlers.NewRepositoryFactoryFromReaderLoaders(loader, yamlFSLoader)
}
