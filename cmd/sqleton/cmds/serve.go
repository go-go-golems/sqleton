package cmds

import (
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/sqleton/pkg"
)

type ServeCommand struct {
	description         *cmds.CommandDescription
	dbConnectionFactory pkg.DBConnectionFactory
}

func NewServeCommand(dbConnectionFactory pkg.DBConnectionFactory) *ServeCommand {
	return &ServeCommand{
		dbConnectionFactory: dbConnectionFactory,
		description: cmds.NewCommandDescription(
			"serve",
			cmds.WithShort("Serve the API"),
			cmds.WithArguments(),
			cmds.WithFlags(),
		),
	}
}
