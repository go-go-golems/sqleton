package cmds

import (
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/cli"
	cmds2 "github.com/wesen/glazed/pkg/cmds"
	"github.com/wesen/glazed/pkg/middlewares"
	sqleton "github.com/wesen/sqleton/pkg"
)

func AddQueriesCmd(allQueries []*sqleton.SqlCommand, aliases []*cmds2.CommandAlias) *cobra.Command {
	var queriesCmd = &cobra.Command{
		Use:   "queries",
		Short: "Commands related to sqleton queries",
		Run: func(cmd *cobra.Command, args []string) {
			gp, of, err := cli.SetupProcessor(cmd)
			cobra.CheckErr(err)
			of.AddTableMiddleware(
				middlewares.NewReorderColumnOrderMiddleware(
					[]string{"name", "short", "long", "source", "query"}),
			)

			for _, query := range allQueries {
				description := query.Description()
				obj := map[string]interface{}{
					"name":   description.Name,
					"short":  description.Short,
					"long":   description.Long,
					"query":  query.Query,
					"source": description.Source,
				}
				err := gp.ProcessInputObject(obj)
				cobra.CheckErr(err)
			}

			for _, alias := range aliases {
				obj := map[string]interface{}{
					"name":     alias.Name,
					"aliasFor": alias.AliasFor,
					"source":   alias.Source,
				}
				err = gp.ProcessInputObject(obj)
				cobra.CheckErr(err)
			}

			s, err := of.Output()
			cobra.CheckErr(err)
			cmd.Println(s)
		},
	}

	flagsDefaults := cli.NewFlagsDefaults()
	flagsDefaults.FieldsFilter.Fields = "name,short,source"
	cli.AddFlags(queriesCmd, flagsDefaults)

	return queriesCmd
}
