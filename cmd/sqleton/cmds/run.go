package cmds

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/glazed/pkg/middlewares"
	"os"
)

// TODO(2022-12-18, manuel): Add support for multiple files
// https://github.com/wesen/sqleton/issues/25
var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a SQL query from sql files",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := openDatabase(cmd)
		cobra.CheckErr(err)

		dbContext := context.Background()
		err = db.PingContext(dbContext)
		cobra.CheckErr(err)

		for _, arg := range args {
			gp, of, err := cli.SetupProcessor(cmd)
			if err != nil {
				_, _ = fmt.Fprintf(os.Stderr, "Could not create glaze  procersors: %v\n", err)
				os.Exit(1)
			}

			// read file
			query, err := os.ReadFile(arg)
			cobra.CheckErr(err)

			rows, err := db.QueryxContext(dbContext, string(query))
			cobra.CheckErr(err)

			// we need a way to order the columns
			cols, err := rows.Columns()
			cobra.CheckErr(err)
			of.AddTableMiddleware(middlewares.NewReorderColumnOrderMiddleware(cols))
			// add support for renaming columns (at least to lowercase)
			// https://github.com/wesen/glazed/issues/27

			for rows.Next() {
				row := map[string]interface{}{}
				err = rows.MapScan(row)
				cobra.CheckErr(err)

				for key, value := range row {
					switch value := value.(type) {
					case []byte:
						row[key] = string(value)
					}
				}

				err = gp.ProcessInputObject(row)
				cobra.CheckErr(err)
			}

			s, err := of.Output()
			cobra.CheckErr(err)
			fmt.Print(s)
		}
	},
}

func init() {
	cli.AddOutputFlags(RunCmd)
	cli.AddTemplateFlags(RunCmd)
	cli.AddFieldsFilterFlags(RunCmd, "")
	cli.AddSelectFlags(RunCmd)

}
