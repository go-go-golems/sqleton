package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/clay/pkg/repositories/mcp"
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/runner"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/settings"
	"github.com/go-go-golems/glazed/pkg/types"
	sqleton_cmds "github.com/go-go-golems/sqleton/pkg/cmds"
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
	glazedSection, err := settings.NewGlazedSection()
	if err != nil {
		return nil, err
	}

	return &ListToolsCommand{
		CommandDescription: cmds.NewCommandDescription(
			"list",
			cmds.WithShort("List all available tools"),
			cmds.WithFlags(
				fields.New(
					"repository",
					fields.TypeString,
					fields.WithHelp("Filter tools by repository name"),
					fields.WithDefault(""),
				),
			),
			cmds.WithSections(glazedSection),
		),
		repositories: repositories,
	}, nil
}

func (c *ListToolsCommand) RunIntoGlazeProcessor(
	ctx context.Context,
	parsedValues *values.Values,
	gp middlewares.Processor,
) error {
	s := &struct {
		Repository string `glazed:"repository"`
	}{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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

		outputValue := ""
		if outputField, ok := parsedValues.GetField(settings.GlazedSlug, "output"); ok {
			if outputString, ok := outputField.Value.(string); ok {
				outputValue = outputString
			}
		}
		if outputValue == "json" {
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
	parsedValues *values.Values,
	cmd *cobra.Command,
	args []string,
	outputOverride sources.Middleware,
) ([]sources.Middleware, error) {
	// Start with cobra-specific middlewares
	middlewares_ := []sources.Middleware{
		sources.FromCobra(cmd,
			fields.WithSource("cobra"),
		),
		sources.FromArgs(args,
			fields.WithSource("arguments"),
		),
	}

	sqletonMiddlewares, err := sqleton_cmds.GetSqletonMiddlewares(parsedValues)
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

	// Create middleware to override output format to JSON
	outputOverride := sources.FromMap(
		map[string]map[string]interface{}{
			settings.GlazedSlug: {
				"output": "json",
			},
		},
		fields.WithSource("output-override"),
	)

	// Build cobra command with custom middlewares
	cobraCmd, err := cli.BuildCobraCommandFromCommand(listCmd,
		cli.WithCobraMiddlewaresFunc(func(
			parsedValues *values.Values,
			cmd *cobra.Command,
			args []string,
		) ([]sources.Middleware, error) {
			return createCommandMiddlewares(parsedValues, cmd, args, outputOverride)
		}),
		cli.WithCobraShortHelpSections(
			schema.DefaultSlug,
			sql.DbtSlug,
			sql.SqlConnectionSlug,
			flags.SqlHelpersSlug,
		),
		cli.WithProfileSettingsSection(),
	)
	if err != nil {
		panic(err)
	}

	toolsCmd.AddCommand(cobraCmd)

	runCmd := mc.CreateRunCmd()
	toolsCmd.AddCommand(runCmd)

	schemaCmd, err := NewSchemaCommand(mc.repositories)
	if err != nil {
		panic(err)
	}

	cobraSchemaCmd, err := cli.BuildCobraCommandFromCommand(schemaCmd,
		cli.WithCobraMiddlewaresFunc(func(
			parsedValues *values.Values,
			cmd *cobra.Command,
			args []string,
		) ([]sources.Middleware, error) {
			return createCommandMiddlewares(parsedValues, cmd, args, nil)
		}),
		cli.WithCobraShortHelpSections(
			schema.DefaultSlug,
			sql.DbtSlug,
			sql.SqlConnectionSlug,
			flags.SqlHelpersSlug,
		),
		cli.WithProfileSettingsSection(),
	)
	if err != nil {
		panic(err)
	}

	toolsCmd.AddCommand(cobraSchemaCmd)

	return toolsCmd
}

// RunCommandSettings holds the parameters for the run command
type RunCommandSettings struct {
	Name         string                 `glazed:"name"`
	Args         string                 `glazed:"args"`
	ArgsFromFile map[string]interface{} `glazed:"args-from-file"`
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
				fields.New(
					"name",
					fields.TypeString,
					fields.WithHelp("Name of the tool to run"),
					fields.WithRequired(true),
				),
			),
			cmds.WithFlags(
				fields.New(
					"args",
					fields.TypeString,
					fields.WithHelp("Arguments as JSON string"),
					fields.WithDefault("{}"),
				),
				fields.New(
					"args-from-file",
					fields.TypeObjectFromFile,
					fields.WithHelp("Load arguments from JSON/YAML file"),
				),
			),
		),
		repositories: repositories,
	}, nil
}

func (c *RunCommand) Run(ctx context.Context, parsedValues *values.Values) error {
	// Parse settings
	s := &RunCommandSettings{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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

	sqletonMiddlewares, err := sqleton_cmds.GetSqletonMiddlewares(parsedValues)
	if err != nil {
		return fmt.Errorf("failed to get sqleton middlewares: %w", err)
	}

	// Parse parameters using runner
	parsedToolValues, err := runner.ParseCommandValues(
		foundCmd,
		runner.WithValuesForSections(map[string]map[string]interface{}{
			schema.DefaultSlug: argsMap,
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
		parsedToolValues,
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
			parsedValues *values.Values,
			cmd *cobra.Command,
			args []string,
		) ([]sources.Middleware, error) {
			return createCommandMiddlewares(parsedValues, cmd, args, nil)
		}),
		cli.WithCobraShortHelpSections(
			schema.DefaultSlug,
			sql.DbtSlug,
			sql.SqlConnectionSlug,
			flags.SqlHelpersSlug,
		),
		cli.WithProfileSettingsSection(),
	)
	if err != nil {
		panic(err)
	}

	return cobraCmd
}

type SchemaCommand struct {
	*cmds.CommandDescription
	repositories []*repositories.Repository
}

func NewSchemaCommand(repositories []*repositories.Repository) (*SchemaCommand, error) {
	return &SchemaCommand{
		CommandDescription: cmds.NewCommandDescription(
			"schema",
			cmds.WithShort("Get JSON schema for a tool"),
			cmds.WithArguments(
				fields.New(
					"name",
					fields.TypeString,
					fields.WithHelp("Name of the tool to get schema for"),
				),
			),
		),
		repositories: repositories,
	}, nil
}

func (c *SchemaCommand) RunIntoWriter(
	ctx context.Context,
	parsedValues *values.Values,
	w io.Writer,
) error {
	s := &struct {
		Name string `glazed:"name"`
	}{}
	if err := parsedValues.DecodeSectionInto(schema.DefaultSlug, s); err != nil {
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

	// Get JSON schema from command description
	schema, err := foundCmd.Description().ToJsonSchema()
	if err != nil {
		return fmt.Errorf("failed to get schema: %w", err)
	}

	// Pretty print the schema
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(schema); err != nil {
		return fmt.Errorf("failed to encode schema: %w", err)
	}

	return nil
}

func (mc *McpCommands) AddToRootCommand(rootCmd *cobra.Command) {
	toolsCmd := mc.CreateToolsCmd()
	McpCmd.AddCommand(toolsCmd)
	rootCmd.AddCommand(McpCmd)
}
