package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/repositories/mcp"
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	cmd_middlewares "github.com/go-go-golems/glazed/pkg/cmds/middlewares"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/cmds/runner"
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
		var prettySchema bytes.Buffer
		err := json.Indent(&prettySchema, tool.InputSchema, "", "  ")
		if err != nil {
			return fmt.Errorf("error formatting input schema: %w", err)
		}

		var inputSchema_ interface{}

		output, _ := parsedLayers.GetParameter(settings.GlazedSlug, "output")
		if output.Value == "json" {
			inputSchema_ = tool.InputSchema
		} else {
			inputSchema_ = prettySchema.String()
		}

		row := map[string]interface{}{
			"name":        tool.Name,
			"description": tool.Description,
			"inputSchema": inputSchema_,
		}
		row_ := types.NewRowFromMap(row)
		if err := gp.AddRow(ctx, row_); err != nil {
			return err
		}
	}

	return nil
}

// createCommandMiddlewares creates the common middleware chain used by MCP commands
func createCommandMiddlewares(
	parsedLayers *layers.ParsedLayers,
	cmd *cobra.Command,
	args []string,
	outputOverride cmd_middlewares.Middleware,
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

	// Add output override if provided
	if outputOverride != nil {
		middlewares_ = append(middlewares_, outputOverride)
	}

	return middlewares_, nil
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
			return createCommandMiddlewares(parsedLayers, cmd, args, outputOverride)
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

// RunCommandSettings holds the parameters for the run command
type RunCommandSettings struct {
	Name         string                 `glazed.parameter:"name"`
	Args         string                 `glazed.parameter:"args"`
	ArgsFromFile map[string]interface{} `glazed.parameter:"args-from-file"`
}

type RunCommand struct {
	*cmds.CommandDescription
	repositories []*repositories.Repository
}

func NewRunCommand(repositories []*repositories.Repository) (*RunCommand, error) {
	return &RunCommand{
		CommandDescription: cmds.NewCommandDescription(
			"run",
			cmds.WithShort("Run a tool by name"),
			cmds.WithArguments(
				parameters.NewParameterDefinition(
					"name",
					parameters.ParameterTypeString,
					parameters.WithHelp("Name of the tool to run"),
				),
			),
			cmds.WithFlags(
				parameters.NewParameterDefinition(
					"args",
					parameters.ParameterTypeString,
					parameters.WithHelp("Arguments as JSON string"),
					parameters.WithDefault("{}"),
				),
				parameters.NewParameterDefinition(
					"args-from-file",
					parameters.ParameterTypeObjectFromFile,
					parameters.WithHelp("Load arguments from JSON/YAML file"),
				),
			),
		),
		repositories: repositories,
	}, nil
}

func (c *RunCommand) Run(ctx context.Context, parsedLayers *layers.ParsedLayers) error {
	// Parse settings
	s := &RunCommandSettings{}
	if err := parsedLayers.InitializeStruct(layers.DefaultSlug, s); err != nil {
		return err
	}

	// Find tool in repositories
	var foundCmd cmds.Command
	for _, repo := range c.repositories {
		cmd, ok := repo.GetCommand(s.Name)
		if ok {
			foundCmd = cmd
			break
		}
	}
	if foundCmd == nil {
		return fmt.Errorf("command %s not found", s.Name)
	}

	// Parse args string into map
	var argsMap map[string]interface{}
	if err := json.Unmarshal([]byte(s.Args), &argsMap); err != nil {
		return fmt.Errorf("failed to parse args JSON: %w", err)
	}

	// Merge with args from file if provided
	if s.ArgsFromFile != nil {
		for k, v := range s.ArgsFromFile {
			argsMap[k] = v
		}
	}

	sqletonMiddlewares, err := sql.GetSqletonMiddlewares(parsedLayers)
	if err != nil {
		return fmt.Errorf("failed to get sqleton middlewares: %w", err)
	}

	// Parse parameters using runner
	parsedToolLayers, err := runner.ParseCommandParameters(
		foundCmd,
		runner.WithValuesForLayers(map[string]map[string]interface{}{
			layers.DefaultSlug: argsMap,
		}),
		runner.WithAdditionalMiddlewares(sqletonMiddlewares...),
	)
	if err != nil {
		return fmt.Errorf("failed to parse tool parameters: %w", err)
	}

	// Run the command using the runner
	err = runner.RunCommand(
		ctx,
		foundCmd,
		parsedToolLayers,
		runner.WithWriter(os.Stdout), // For WriterCommand
	)
	if err != nil {
		return fmt.Errorf("failed to run tool: %w", err)
	}

	return nil
}

func (mc *McpCommands) CreateRunCmd() *cobra.Command {
	runCmd, err := NewRunCommand(mc.repositories)
	if err != nil {
		panic(err)
	}

	// Build cobra command with custom middlewares
	cobraCmd, err := cli.BuildCobraCommandFromCommand(runCmd,
		cli.WithCobraMiddlewaresFunc(func(
			parsedLayers *layers.ParsedLayers,
			cmd *cobra.Command,
			args []string,
		) ([]cmd_middlewares.Middleware, error) {
			return createCommandMiddlewares(parsedLayers, cmd, args, nil)
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

	return cobraCmd
}

func (mc *McpCommands) AddToRootCommand(rootCmd *cobra.Command) {
	toolsCmd := mc.CreateToolsCmd()
	runCmd := mc.CreateRunCmd()
	McpCmd.AddCommand(toolsCmd, runCmd)
	rootCmd.AddCommand(McpCmd)
}
