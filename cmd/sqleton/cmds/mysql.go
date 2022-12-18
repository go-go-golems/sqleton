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
		Section: &help.Section{
			Slug:           "ps",
			Title:          "List MySQL processes",
			SubTitle:       "SHOW PROCESSLIST",
			Short:          "Return a table of currently running MySQL processes",
			Content:        "# YO\n yo yo yo \n- foo\n- bar\n",
			Topics:         []string{"mysql"},
			Commands:       []string{"ps"},
			SectionType:    help.SectionGeneralTopic,
			IsTopLevel:     true,
			ShowPerDefault: true,
		},
		Query: "SHOW PROCESSLIST",
	}
	cmd, err := sqleton.ToCobraCommand(showProcessSqlCmd)
	if err != nil {
		panic(err)
	}
	hs.AddSection(showProcessSqlCmd.Section)

	MysqlCmd.AddCommand(cmd)
}
