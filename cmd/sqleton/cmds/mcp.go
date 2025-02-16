package cmds

import (
	"encoding/json"
	"fmt"

	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/repositories/mcp"
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

func (mc *McpCommands) CreateToolsCmd() *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "Tool related commands",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all available tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			allTools := []mcp.Tool{}

			for _, repo := range mc.repositories {
				tools, _, err := repo.ListTools(cmd.Context(), "")
				if err != nil {
					return fmt.Errorf("error listing tools from repository: %w", err)
				}
				allTools = append(allTools, tools...)

			}

			jsonBytes, err := json.MarshalIndent(allTools, "", "  ")
			if err != nil {
				return fmt.Errorf("error marshaling tools to JSON: %w", err)
			}

			fmt.Println(string(jsonBytes))
			return nil
		},
	}

	toolsCmd.AddCommand(listCmd)
	return toolsCmd
}

func (mc *McpCommands) AddToRootCommand(rootCmd *cobra.Command) {
	toolsCmd := mc.CreateToolsCmd()
	McpCmd.AddCommand(toolsCmd)
	rootCmd.AddCommand(McpCmd)
}
