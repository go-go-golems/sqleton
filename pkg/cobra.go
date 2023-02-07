package pkg

import (
	"context"
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// RunFromCobra actually runs the given SqletonCommand by using the cobra command
// to parse the necessary flags. It then first pings the database, and then renders
// the query results into a GlazedProcessor.
func (s *SqlCommand) RunFromCobra(cmd *cobra.Command, args []string) error {
	// TODO(2022-12-20, manuel): we should be able to load default values for these parameters from a config file
	// See: https://github.com/wesen/sqleton/issues/39
	description := s.Description()

	parameters, err := cmds.GatherParameters(cmd, description, args)
	if err != nil {
		return err
	}

	db, err := OpenDatabaseFromViper()
	if err != nil {
		return errors.Wrapf(err, "Could not open database")
	}

	dbContext := context.Background()
	err = db.PingContext(dbContext)
	if err != nil {
		return errors.Wrapf(err, "Could not ping database")
	}

	gp, of, err := cli.SetupProcessor(cmd)
	if err != nil {
		return errors.Wrapf(err, "Could not setup processor")
	}

	// TODO(2022-12-21, manuel): Add explain functionality
	// See: https://github.com/wesen/sqleton/issues/45
	explain, _ := cmd.Flags().GetBool("explain")
	parameters["explain"] = explain
	_ = explain

	printQuery, _ := cmd.Flags().GetBool("print-query")
	if printQuery {
		query, err := s.RenderQuery(parameters)
		if err != nil {
			return errors.Wrapf(err, "Could not generate query")
		}
		fmt.Println(query)
		return nil
	}

	err = s.RunQueryIntoGlaze(dbContext, db, parameters, gp)
	if err != nil {
		return errors.Wrapf(err, "Could not run query")
	}

	output, err := of.Output()
	if err != nil {
		return errors.Wrapf(err, "Could not get output")
	}
	fmt.Print(output)

	return nil
}

func init() {
}
