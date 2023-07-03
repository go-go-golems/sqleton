package pkg

import (
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/parka/pkg/handlers"
)

func NewRepositoryFactory() handlers.RepositoryFactory {
	yamlFSLoader := loaders.NewYAMLFSCommandLoader(&SqlCommandLoader{
		DBConnectionFactory: OpenDatabaseFromSqletonConnectionLayer,
	})
	yamlLoader := &loaders.YAMLReaderCommandLoader{
		YAMLCommandLoader: &SqlCommandLoader{
			DBConnectionFactory: OpenDatabaseFromSqletonConnectionLayer,
		},
	}

	return handlers.NewRepositoryFactoryFromLoaders(yamlLoader, yamlFSLoader)
}
