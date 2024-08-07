package cmds

import (
	"github.com/go-go-golems/clay/pkg/cmds/repositories"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/spf13/cobra"
)

func NewConfigGroupCommand(helpSystem *help.HelpSystem) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Commands for manipulating configuration and profiles",
	}

	cmd.AddCommand(repositories.NewRepositoriesGroupCommand())
	err := repositories.AddDocToHelpSystem(helpSystem)
	if err != nil {
		return nil, err
	}

	return cmd, nil
}
