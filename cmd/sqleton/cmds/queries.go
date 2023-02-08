package cmds

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	glazed_cmds "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	sqleton "github.com/go-go-golems/sqleton/pkg"
	"github.com/spf13/cobra"
)

// TODO(manuel, 2023-02-07) This should go to glazed into the commands section
// although, it's actually printing out the query in this case, and probably should be
// used for application specification additional information anyway
func AddQueriesCmd(allQueries []*sqleton.SqlCommand, aliases []*glazed_cmds.CommandAlias) *cobra.Command {
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
