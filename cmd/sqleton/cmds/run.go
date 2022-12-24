package cmds

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/sqleton/pkg"
	"io"
	"os"
)

// TODO(2022-12-18, manuel): Add support for multiple files
// https://github.com/wesen/sqleton/issues/25
var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a SQL query from sql files",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := pkg.OpenDatabaseFromViper()
		if err != nil {
			return errors.Wrapf(err, "Could not open database")
		}

		dbContext := context.Background()
		err = db.PingContext(dbContext)
		if err != nil {
			return errors.Wrapf(err, "Could not ping database")
		}

		for _, arg := range args {
			gp, of, err := cli.SetupProcessor(cmd)
			if err != nil {
				return errors.Wrapf(err, "Could not create glaze processors")
			}

			// read file
			query, err := os.ReadFile(arg)
			if err != nil {
				return errors.Wrapf(err, "Could not read file: %s", arg)
			}

			// TODO(2022-12-20, manuel): collect named parameters here, maybe through prerun?
			// See: https://github.com/wesen/sqleton/issues/40
			err = pkg.RunQueryIntoGlaze(dbContext, db, string(query), map[string]interface{}{}, gp)
			if err != nil {
				return errors.Wrapf(err, "Could not run query")
			}

			s, err := of.Output()
			if err != nil {
				return errors.Wrapf(err, "Could not get output")
			}
			fmt.Print(s)
		}

		return nil
	},
}

var QueryCmd = &cobra.Command{
	Use:   "query <query>",
	Short: "Run a SQL query",
	Long:  "Run a SQL query. The query can be passed as an argument or via stdin if - is passed as the query.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]
		if args[0] == "-" {
			inBytes, err := io.ReadAll(os.Stdin)
			cobra.CheckErr(err)
			query = string(inBytes)
		}

		db, err := pkg.OpenDatabaseFromViper()
		cobra.CheckErr(err)

		dbContext := context.Background()
		err = db.PingContext(dbContext)
		cobra.CheckErr(err)

		gp, of, err := cli.SetupProcessor(cmd)
		cobra.CheckErr(err)

		err = pkg.RunQueryIntoGlaze(dbContext, db, query, map[string]interface{}{}, gp)
		cobra.CheckErr(err)

		s, err := of.Output()
		cobra.CheckErr(err)

		fmt.Print(s)
	},
}

func init() {
	cli.AddOutputFlags(RunCmd)
	cli.AddTemplateFlags(RunCmd)
	cli.AddFieldsFilterFlags(RunCmd, "")
	cli.AddSelectFlags(RunCmd)

	cli.AddOutputFlags(QueryCmd)
	cli.AddTemplateFlags(QueryCmd)
	cli.AddFieldsFilterFlags(QueryCmd, "")
	cli.AddSelectFlags(QueryCmd)
}
