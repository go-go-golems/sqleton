package cmds

import (
	"github.com/go-go-golems/clay/pkg/sql"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	sqleton "github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/spf13/cobra"
)

var MysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "MySQL commands",
}

func init() {
	// This is an example of how to programmatically create a sqleton query
	psCommand, err := sqleton.NewSqlCommand(
		glazed_cmds.NewCommandDescription("ps-manual",
			glazed_cmds.WithShort("List MySQL processes"),
			glazed_cmds.WithLong("SHOW PROCESSLIST"),
		),
		sqleton.WithDbConnectionFactory(sql.OpenDatabaseFromDefaultSqlConnectionLayer),
		sqleton.WithQuery("SHOW PROCESSLIST"),
	)
	if err != nil {
		panic(err)
	}
	cobraPsCommand, err := sql.BuildCobraCommandWithSqletonMiddlewares(psCommand)
	if err != nil {
		panic(err)
	}
	MysqlCmd.AddCommand(cobraPsCommand)
}
