package pkg

import (
	"github.com/go-go-golems/clay/pkg/repositories/sql"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/parka/pkg/handlers"
)

func NewRepositoryFactory() handlers.RepositoryFactory {
	yamlFSLoader := loaders.NewYAMLFSCommandLoader(&SqlCommandLoader{
		DBConnectionFactory: sql.OpenDatabaseFromDefaultSqlConnectionLayer,
	})
	yamlLoader := &loaders.YAMLReaderCommandLoader{
		YAMLCommandLoader: &SqlCommandLoader{
			DBConnectionFactory: sql.OpenDatabaseFromDefaultSqlConnectionLayer,
		},
	}

	return handlers.NewRepositoryFactoryFromLoaders(yamlLoader, yamlFSLoader)
}
