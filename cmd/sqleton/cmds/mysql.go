package cmds

import (
	"embed"
	cmds2 "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/help"
	sqleton "github.com/go-go-golems/sqleton/pkg"
	"github.com/spf13/cobra"
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
	cmd, err := showProcessSqlCmd.BuildCobraCommand()
	if err != nil {
		panic(err)
	}

	MysqlCmd.AddCommand(cmd)
}
