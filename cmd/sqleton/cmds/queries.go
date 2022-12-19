package cmds

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/glazed/pkg/middlewares"
	sqleton "github.com/wesen/sqleton/pkg"
)

func AddQueriesCmd(allQueries []*sqleton.SqlCommand) *cobra.Command {
	var queriesCmd = &cobra.Command{
		Use:   "queries",
		Short: "Commands related to sqleton queries",
		RunE: func(cmd *cobra.Command, args []string) error {
			gp, of, err := cli.SetupProcessor(cmd)
			if err != nil {
				return errors.Wrapf(err, "Could not create glaze processors")
			}
			of.AddTableMiddleware(
				middlewares.NewReorderColumnOrderMiddleware(
					[]string{"name", "short", "long", "source", "query"}),
			)

			for _, query := range allQueries {
				obj := map[string]interface{}{
					"name":   query.Name,
					"short":  query.Short,
					"long":   query.Long,
					"query":  query.Query,
					"source": query.Source,
				}
				err := gp.ProcessInputObject(obj)
				if err != nil {
					return errors.Wrapf(err, "Could not process input object")
				}
			}

			s, err := of.Output()
			if err != nil {
				return errors.Wrapf(err, "Could not get output")
			}
			cmd.Println(s)

			return nil
		},
	}

	cli.AddOutputFlags(queriesCmd)
	cli.AddTemplateFlags(queriesCmd)
	cli.AddFieldsFilterFlags(queriesCmd, "name,short,source")
	cli.AddSelectFlags(queriesCmd)
	return queriesCmd
}
