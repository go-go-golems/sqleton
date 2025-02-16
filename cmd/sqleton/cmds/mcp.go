package cmds

import (
	"context"
	"fmt"

	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/repositories/mcp"
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	cmd_middlewares "github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/go-go-golems/sqleton/pkg/flags"
	"github.com/spf13/cobra"
)

type McpCommands struct {
	repositories []*repositories.Repository
}

func NewMcpCommands(repositories []*repositories.Repository) *McpCommands {
	return &McpCommands{
		repositories: repositories,
	}
}

var McpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "MCP (Machine Control Protocol) related commands",
}

type ListToolsCommand struct {
	*cmds.CommandDescription
	repositories []*repositories.Repository
}

func NewListToolsCommand(repositories []*repositories.Repository) (*ListToolsCommand, error) {
	glazedLayer, err := settings.NewGlazedParameterLayers()
	if err != nil {
		return nil, err
	}

	return &ListToolsCommand{
		CommandDescription: cmds.NewCommandDescription(
			"list",
			cmds.WithShort("List all available tools"),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"repository",
					parameters.ParameterTypeString,
					parameters.WithHelp("Filter tools by repository name"),
					parameters.WithDefault(""),
				),
			),
			cmds.WithLayersList(glazedLayer),
		),
		repositories: repositories,
	}, nil
}

func (c *ListToolsCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedLayers *layers.ParsedLayers,
	gp middlewares.Processor,
) error {
	s := &struct {
		Repository string `glazed.parameter:"repository"`
	}{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	allTools := []mcp.Tool{}
	for _, repo := range c.repositories {
		tools, _, err := repo.ListTools(ctx, s.Repository)
		if err != nil {
			return fmt.Errorf("error listing tools from repository: %w", err)
		}
		allTools = append(allTools, tools...)
	}

	for _, tool := range allTools {
		row := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": tool.InputSchema,
		}
		row_ := types.NewRowFromMap(row)
		if err := gp.AddRow(ctx, row_); err != nil {
			return err
		}
	}

	return nil
}

func (mc *McpCommands) CreateToolsCmd() *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "Tool related commands",
	}

	listCmd, err := NewListToolsCommand(mc.repositories)
	if err != nil {
		panic(err)
	}

	// Create middleware to override output format to YAML
	outputOverride := cmd_middlewares.UpdateFromMap(
		map[string]map[string]interface{}{
			"glazed": {
				"output": "json",
			},
		},
		parameters.WithParseStepSource("output-override"),
	)

	// Build cobra command with custom middlewares
	cobraCmd, err := cli.BuildCobraCommandFromCommand(listCmd,
		cli.WithCobraMiddlewaresFunc(func(
			parsedLayers *layers.ParsedLayers,
			cmd *cobra.Command,
			args []string,
		) ([]cmd_middlewares.Middleware, error) {
			// Start with cobra-specific middlewares
			middlewares_ := []cmd_middlewares.Middleware{
				cmd_middlewares.ParseFromCobraCommand(cmd,
					parameters.WithParseStepSource("cobra"),
				),
				cmd_middlewares.GatherArguments(args,
					parameters.WithParseStepSource("arguments"),
				),
			}

			sqletonMiddlewares, err := sql.GetSqletonMiddlewares(parsedLayers)
			if err != nil {
				return nil, err
			}
			middlewares_ = append(middlewares_, sqletonMiddlewares...)
			// sneak in the output override at the end
			middlewares_ = append(middlewares_, outputOverride)
			return middlewares_, nil
		}),
		cli.WithCobraShortHelpLayers(
			layers.DefaultSlug,
			sql.DbtSlug,
			sql.SqlConnectionSlug,
			flags.SqlHelpersSlug,
		),
		cli.WithProfileSettingsLayer(),
	)
	if err != nil {
		panic(err)
	}

	toolsCmd.AddCommand(cobraCmd)
	return toolsCmd
}

func (mc *McpCommands) AddToRootCommand(rootCmd *cobra.Command) {
	toolsCmd := mc.CreateToolsCmd()
	McpCmd.AddCommand(toolsCmd)
	rootCmd.AddCommand(McpCmd)
}
