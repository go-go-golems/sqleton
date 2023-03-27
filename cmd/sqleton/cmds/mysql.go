package cmds

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	sqleton "github.com/go-go-golems/sqleton/pkg"
	"github.com/spf13/cobra"
)

var MysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "MySQL commands",
}

func init() {
	psCommand, err := sqleton.NewSqlCommand(
		glazed_cmds.NewCommandDescription("ps",
			glazed_cmds.WithShort("List MySQL processes"),
			glazed_cmds.WithLong("SHOW PROCESSLIST"),
		),
		sqleton.WithDbConnectionFactory(sqleton.OpenDatabaseFromSqletonConnectionLayer),
		sqleton.WithQuery("SHOW PROCESSLIST"),
	)
	if err != nil {
		panic(err)
	}
	cobraPsCommand, err := cli.BuildCobraCommandFromGlazeCommand(psCommand)
	if err != nil {
		panic(err)
	}
	MysqlCmd.AddCommand(cobraPsCommand)
}
