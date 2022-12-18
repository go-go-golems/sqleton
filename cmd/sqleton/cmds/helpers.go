package cmds

import (
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/wesen/sqleton/pkg"
)

func openDatabase(cmd *cobra.Command) (*sqlx.DB, error) {
	useDbtProfiles, err := cmd.Flags().GetBool("use-dbt-profiles")
	cobra.CheckErr(err)

	var source *pkg.Source

	if useDbtProfiles {
		dbtProfilesPath, err := cmd.Flags().GetString("dbt-profiles-path")
		if err != nil {
			return nil, err
		}

		sources, err := pkg.ParseDbtProfiles(dbtProfilesPath)
		if err != nil {
			return nil, err
		}

		sourceName, err := cmd.Flags().GetString("dbt-profile")
		if err != nil {
			return nil, err
		}

		for _, s := range sources {
			if s.Name == sourceName {
				source = s
				break
			}
		}

		if source == nil {
			return nil, errors.Errorf("Source %s not found", sourceName)
		}
	} else {
		source, err = setupSource(cmd)
		if err != nil {
			return nil, err
		}
	}

	db, err := sqlx.Connect(source.Type, source.ToConnectionString())

	// TODO(2022-12-18, manuel): this is where we would add support for a ro connection
	// https://github.com/wesen/sqleton/issues/24

	return db, err
}
