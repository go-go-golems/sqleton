package cmds

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/spf13/cobra"
)

const SqletonAppName = "sqleton"

func NewSqletonParserConfig() cli.CobraParserConfig {
	return cli.CobraParserConfig{
		AppName:         SqletonAppName,
		ConfigFilesFunc: resolveSqletonCommandConfigFiles,
	}
}

func resolveSqletonCommandConfigFiles(
	parsedCommandSections *values.Values,
	_ *cobra.Command,
	_ []string,
) ([]string, error) {
	commandSettings := &cli.CommandSettings{}
	if err := parsedCommandSections.DecodeSectionInto(cli.CommandSettingsSlug, commandSettings); err != nil {
		return nil, err
	}

	if commandSettings.ConfigFile == "" {
		return nil, nil
	}

	return []string{commandSettings.ConfigFile}, nil
}
