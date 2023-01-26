package cmds

import (
	"embed"
	"github.com/spf13/cobra"
	cmds2 "github.com/wesen/glazed/pkg/cmds"
	"github.com/wesen/glazed/pkg/help"
	sqleton "github.com/wesen/sqleton/pkg"
)

var MysqlCmd = &cobra.Command{
	Use:   "mysql",
	Short: "MySQL commands",
}

func InitializeMysqlCmd(queriesFS embed.FS, _ *help.HelpSystem) {
	showProcessSqlCmd := sqleton.NewSqlCommand(
		&cmds2.CommandDescription{Name: "ps",
			Short: "List MySQL processes",
			Long:  "SHOW PROCESSLIST",
		},
		"SHOW PROCESSLIST",
	)
	cmd, err := cmds2.ToCobraCommand(showProcessSqlCmd)
	if err != nil {
		panic(err)
	}

	MysqlCmd.AddCommand(cmd)
}
