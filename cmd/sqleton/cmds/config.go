package cmds

import (
	"github.com/go-go-golems/clay/pkg/cmds/repositories"
	"github.com/go-go-golems/glazed/pkg/config"
	"github.com/go-go-golems/glazed/pkg/help"
	"github.com/spf13/cobra"
)

func NewConfigGroupCommand(helpSystem *help.HelpSystem) (*cobra.Command, error) {
	configCmd, err := config.NewConfigCommand("sqleton")
	if err != nil {
		return nil, err
	}

	configCmd.AddCommand(repositories.NewRepositoriesGroupCommand())
	err = repositories.AddDocToHelpSystem(helpSystem)
	if err != nil {
		return nil, err
	}

	return configCmd.Command, nil
}
