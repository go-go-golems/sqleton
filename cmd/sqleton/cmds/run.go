package cmds

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/sqleton/pkg"
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

			err = pkg.RunQueryIntoGlaze(dbContext, db, string(query), gp)
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

func init() {
	cli.AddOutputFlags(RunCmd)
	cli.AddTemplateFlags(RunCmd)
	cli.AddFieldsFilterFlags(RunCmd, "")
	cli.AddSelectFlags(RunCmd)
}
