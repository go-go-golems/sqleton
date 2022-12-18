package cmds

import (
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/help"
	sqleton "github.com/wesen/sqleton/pkg"
)

var MysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "MySQL commands",
}

func InitializeMysqlCmd(hs *help.HelpSystem) {
	showProcessSqlCmd := &sqleton.SqlCommand{
		CommandDescription: sqleton.SqletonCommandDescription{
			Name:  "ps",
			Short: "List MySQL processes",
			Long:  "SHOW PROCESSLIST",
		},
		Query: "SHOW PROCESSLIST",
	}
	cmd, err := sqleton.ToCobraCommand(showProcessSqlCmd)
	if err != nil {
		panic(err)
	}

	MysqlCmd.AddCommand(cmd)
}
