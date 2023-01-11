package cmds

import (
	"context"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
	"syscall"

	"github.com/huandu/go-sqlbuilder"
	"github.com/spf13/cobra"
	"github.com/wesen/glazed/pkg/cli"
	"github.com/wesen/sqleton/pkg"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
)

// TODO(2022-12-18, manuel): Add support for multiple files
// https://github.com/wesen/sqleton/issues/25
var RunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a SQL query from sql files",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := pkg.OpenDatabaseFromViper()
		cobra.CheckErr(err)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		err = db.PingContext(ctx)
		cobra.CheckErr(err)

		watch, _ := cmd.Flags().GetBool("watch")

		errGroup, ctx2 := errgroup.WithContext(ctx)
		errGroup.Go(func() error {
			return pkg.CancelOnSignal(ctx2, syscall.SIGINT, func() {
				cancel()
			})
		})

		if watch {
			errGroup.Go(func() error {
				return watchQueries(ctx2, db, args)
			})
		} else {
			errGroup.Go(func() error {
				for _, arg := range args {
					query := ""

					if arg == "-" {
						inBytes, err := io.ReadAll(os.Stdin)
						if err != nil {
							return err
						}
						query = string(inBytes)
					} else {
						// read file
						queryBytes, err := os.ReadFile(arg)
						if err != nil {
							return err
						}

						query = string(queryBytes)
					}

					// TODO(2022-12-20, manuel): collect named parameters here, maybe through prerun?
					// See: https://github.com/wesen/sqleton/issues/40

					gp, of, err := cli.SetupProcessor(cmd)
					if err != nil {
						return err
					}

					err = pkg.RunNamedQueryIntoGlaze(ctx2, db, string(query), map[string]interface{}{}, gp)
					if err != nil {
						return err
					}

					s, err := of.Output()
					if err != nil {
						return err
					}

					fmt.Print(s)
				}
				return nil
			})
		}

		err = errGroup.Wait()
		cobra.CheckErr(err)
	},
}

// watchQueries takes a list of file names containing SQL queries.
// It will watch for changes and run the appropriate query again when a change is detected.
func watchQueries(ctx context.Context, db *sqlx.DB, args []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	defer watcher.Close()
	queries := make(map[string]string)
	for _, arg := range args {
		queryBytes, err := os.ReadFile(arg)
		if err != nil {
			return err
		}

		queries[arg] = string(queryBytes)

		err = watcher.Add(arg)
		if err != nil {
			return err
		}
	}

	done := make(chan bool)

	// Start a goroutine to handle fsnotify events
	go func() {
		defer close(done)

		log.Debug().Msg("Watching for changes")
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("Context cancelled, stopping watcher")
				return

			case err, ok := <-watcher.Errors:
				if !ok {
					log.Debug().Msg("Watcher closed, stopping watcher")
					return
				}
				log.Error().Err(err).Msg("Watcher error")

			case event, ok := <-watcher.Events:
				if !ok {
					log.Debug().Msg("Watcher channel closed, stopping watcher")
					return
				}
				log.Debug().Str("event", event.String()).Msg("Event received")
				event.Op &= ^fsnotify.Chmod
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Debug().Str("modifiedFile", event.Name).Msg("modified file")
				}
			}
		}
	}()

	// Wait indefinitely
	<-done

	return nil
}

var QueryCmd = &cobra.Command{
	Use:   "query <query>",
	Short: "Run a SQL query passed as a CLI argument",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		query := args[0]

		db, err := pkg.OpenDatabaseFromViper()
		cobra.CheckErr(err)

		dbContext := context.Background()
		err = db.PingContext(dbContext)
		cobra.CheckErr(err)

		gp, of, err := cli.SetupProcessor(cmd)
		cobra.CheckErr(err)

		err = pkg.RunNamedQueryIntoGlaze(dbContext, db, query, map[string]interface{}{}, gp)
		cobra.CheckErr(err)

		s, err := of.Output()
		cobra.CheckErr(err)

		fmt.Print(s)
	},
}

var SelectCmd = &cobra.Command{
	Use: "select <table>",
	// we do the weird plus thing so that golang doesn't parse this
	// as a SQL injection string
	Short: "Select" + " all columns from a table",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		table := args[0]

		columns, err := cmd.Flags().GetStringSlice("columns")
		cobra.CheckErr(err)
		limit, err := cmd.Flags().GetInt("limit")
		cobra.CheckErr(err)
		offset, err := cmd.Flags().GetInt("offset")
		cobra.CheckErr(err)
		count, err := cmd.Flags().GetBool("count")
		cobra.CheckErr(err)
		where, err := cmd.Flags().GetString("where")
		cobra.CheckErr(err)
		order, err := cmd.Flags().GetString("order-by")
		cobra.CheckErr(err)
		distinct, err := cmd.Flags().GetBool("distinct")
		cobra.CheckErr(err)

		sb := sqlbuilder.NewSelectBuilder()
		sb = sb.From(table)

		if count {
			countColumns := strings.Join(columns, ", ")
			if distinct {
				countColumns = "DISTINCT " + countColumns
			}
			columns = []string{sb.As(fmt.Sprintf("COUNT(%s)", countColumns), "count")}
		} else {
			if len(columns) == 0 {
				columns = []string{"*"}
			}
		}
		sb = sb.Select(columns...)
		if distinct && !count {
			sb = sb.Distinct()
		}

		if where != "" {
			sb = sb.Where(where)
		}

		if limit > 0 && !count {
			sb = sb.Limit(limit)
		}
		if offset > 0 {
			sb = sb.Offset(offset)
		}
		if order != "" {
			sb = sb.OrderBy(order)
		}

		createQuery, err := cmd.Flags().GetString("create-query")
		cobra.CheckErr(err)
		if createQuery != "" {
			short := fmt.Sprintf("Select"+" columns from %s", table)
			if count {
				short = fmt.Sprintf("Count all rows from %s", table)
			}
			if where != "" {
				short = fmt.Sprintf("Select"+" from %s where %s", table, where)
			}

			flags := []*pkg.SqlParameter{}
			if where == "" {
				flags = append(flags, &pkg.SqlParameter{
					Name: "where",
					Type: pkg.ParameterTypeString,
				})
			}
			if !count {
				flags = append(flags, &pkg.SqlParameter{
					Name:    "limit",
					Type:    pkg.ParameterTypeInteger,
					Help:    fmt.Sprintf("Limit the number of rows (default: %d), set to 0 to disable", limit),
					Default: limit,
				})
				flags = append(flags, &pkg.SqlParameter{
					Name:    "offset",
					Type:    pkg.ParameterTypeInteger,
					Help:    fmt.Sprintf("Offset the number of rows (default: %d)", offset),
					Default: offset,
				})
				flags = append(flags, &pkg.SqlParameter{
					Name:    "distinct",
					Type:    pkg.ParameterTypeBool,
					Help:    fmt.Sprintf("Whether to select distinct rows (default: %t)", distinct),
					Default: distinct,
				})

				orderByHelp := "Order by"
				var orderDefault interface{}
				if order != "" {
					orderByHelp = fmt.Sprintf("Order by (default: %s)", order)
					orderDefault = order
				}
				flags = append(flags, &pkg.SqlParameter{
					Name:    "order_by",
					Type:    pkg.ParameterTypeString,
					Help:    orderByHelp,
					Default: orderDefault,
				})
			}

			sb := &strings.Builder{}
			_, _ = fmt.Fprintf(sb, "SELECT ")
			if !count {
				_, _ = fmt.Fprintf(sb, "{{ if .distinct }}DISTINCT{{ end }} ")
			}
			_, _ = fmt.Fprintf(sb, "%s FROM %s", strings.Join(columns, ", "), table)
			if where != "" {
				_, _ = fmt.Fprintf(sb, " WHERE %s", where)
			} else {
				_, _ = fmt.Fprintf(sb, "\n{{ if .where  }}  WHERE {{.where}} {{ end }}")
			}

			_, _ = fmt.Fprintf(sb, "\n{{ if .order_by }} ORDER BY {{ .order_by }}{{ end }}")
			_, _ = fmt.Fprintf(sb, "\n{{ if .limit }} LIMIT {{ .limit }}{{ end }}")
			_, _ = fmt.Fprintf(sb, "\nOFFSET {{ .offset }}")

			query := sb.String()
			sqlCommand := &pkg.SqlCommand{
				Name:  createQuery,
				Short: short,
				Flags: flags,
				Query: query,
			}

			// marshal to yaml
			yamlBytes, err := yaml.Marshal(sqlCommand)
			cobra.CheckErr(err)

			fmt.Println(string(yamlBytes))
			return
		}

		query, queryArgs := sb.Build()

		printQuery, err := cmd.Flags().GetBool("print-query")
		cobra.CheckErr(err)
		if printQuery {
			fmt.Println(query)
			fmt.Println(queryArgs)
			return
		}

		db, err := pkg.OpenDatabaseFromViper()
		cobra.CheckErr(err)

		dbContext := context.Background()
		err = db.PingContext(dbContext)
		cobra.CheckErr(err)

		gp, of, err := cli.SetupProcessor(cmd)
		cobra.CheckErr(err)

		err = pkg.RunQueryIntoGlaze(dbContext, db, query, queryArgs, gp)
		cobra.CheckErr(err)

		s, err := of.Output()
		cobra.CheckErr(err)

		fmt.Print(s)
	},
}

func init() {
	cli.AddFlags(RunCmd, cli.NewFlagsDefaults())
	RunCmd.Flags().Bool("watch", false, "Watch for changes and rerun the query")

	cli.AddFlags(QueryCmd, cli.NewFlagsDefaults())

	cli.AddFlags(SelectCmd, cli.NewFlagsDefaults())
	SelectCmd.Flags().String("where", "", "Where clause")
	SelectCmd.Flags().String("order-by", "", "Order by clause")
	SelectCmd.Flags().Int("limit", 50, "Limit clause (default 50, 0 for no limit)")
	SelectCmd.Flags().Int("offset", 0, "Offset clause")
	SelectCmd.Flags().Bool("count", false, "Count clause")
	SelectCmd.Flags().StringSlice("columns", []string{}, "Columns to select")
	SelectCmd.Flags().Bool("print-query", false, "Print the query that is run")
	SelectCmd.Flags().String("create-query", "", "Output the query as yaml to use as a sqleton command")
	SelectCmd.Flags().Bool("distinct", false, "Only return DISTINCT rows")
}
