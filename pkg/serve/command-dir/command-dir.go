package command_dir

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	parka "github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/sqleton/pkg"
	"github.com/go-go-golems/sqleton/pkg/serve/config"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"io/fs"
	"os"
	"strings"
	"time"
)

type Link struct {
	Href  string
	Text  string
	Class string
}

type CommandDirHandler struct {
	DevMode bool

	TemplateName      string
	IndexTemplateName string
	TemplateDirectory string
	TemplateFS        fs.FS

	Repositories []string
	Overrides    *config.LayerParams
	Defaults     *config.LayerParams

	DbtConnectionLayer     *layers.ParsedParameterLayer
	SqletonConnectionLayer *layers.ParsedParameterLayer

	repository *repositories.Repository

	// NOTE(manuel, 2023-05-26) This is probably the right location to add configurable
	// templateLookup chains.
}

func GetRepositoryCommand(c *gin.Context, r *repositories.Repository, commandPath string) (cmds.GlazeCommand, bool) {
	path := strings.Split(commandPath, "/")
	commands := r.Root.CollectCommands(path, false)
	if len(commands) == 0 {
		c.JSON(404, gin.H{"error": "command not found"})
		return nil, false
	}

	if len(commands) > 1 {
		c.JSON(404, gin.H{"error": "ambiguous command"})
		return nil, false
	}

	// NOTE(manuel, 2023-05-15) Check if this is actually an alias, and populate the defaults from the alias flags
	// This could potentially be moved to the repository code itself

	sqlCommand, ok := commands[0].(cmds.GlazeCommand)
	if !ok || sqlCommand == nil {
		c.JSON(500, gin.H{"error": "command is not a sql command"})
	}
	return sqlCommand, true
}

type CommandDirHandlerOption func(handler *CommandDirHandler)

func WithTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.TemplateName = name
	}
}

func WithDefaultTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.TemplateName == "" {
			handler.TemplateName = name
		}
	}
}

func WithIndexTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.IndexTemplateName = name
	}
}

func WithDefaultIndexTemplateName(name string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.IndexTemplateName == "" {
			handler.IndexTemplateName = name
		}
	}
}

func WithTemplateFS(fs fs.FS) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.TemplateFS = fs
	}
}

func WithDefaultTemplateFS(fs fs.FS, templateDirectory string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if handler.TemplateFS == nil {
			handler.TemplateFS = fs
			handler.TemplateDirectory = templateDirectory
		}
	}
}

func WithTemplateDirectory(directory string) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		if directory != "" {
			if directory[0] == '/' {
				handler.TemplateFS = os.DirFS(directory)
			} else {
				handler.TemplateFS = os.DirFS(directory)
			}
			handler.TemplateDirectory = strings.TrimPrefix(directory, "/")
		}
	}
}

// TODO(manuel, 2023-05-26) This should be replaced by a generic ParsedParameterLayer overload thing
func WithDbtConnectionLayer(layer *layers.ParsedParameterLayer) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.DbtConnectionLayer = layer
	}
}

func WithSqletonConnectionLayer(layer *layers.ParsedParameterLayer) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.SqletonConnectionLayer = layer
	}
}

func WithDevMode(devMode bool) CommandDirHandlerOption {
	return func(handler *CommandDirHandler) {
		handler.DevMode = devMode
	}
}

func NewCommandDirHandlerFromConfig(
	config *config.CommandDir,
	options ...CommandDirHandlerOption,
) (*CommandDirHandler, error) {
	cd := &CommandDirHandler{
		TemplateName:      config.TemplateName,
		IndexTemplateName: config.IndexTemplateName,
		Repositories:      config.Repositories,
		Overrides:         config.Overrides,
		Defaults:          config.Defaults,
	}

	WithTemplateDirectory(config.TemplateDirectory)(cd)

	for _, option := range options {
		option(cd)
	}

	return cd, nil
}

// GetRepository uses the configured repositories to load a single repository watcher, and load all
// the necessary commands and aliases at startup.
//
// NOTE(manuel, 2023-05-26) This could probably be extracted out of the CommandHandler and maybe submitted as
// a utility, as this currently ties the YAML load and the whole sqleton thing directly into the CommandDirHandler.
func (cd *CommandDirHandler) GetRepository() (*repositories.Repository, error) {
	if cd.repository != nil {
		return cd.repository, nil
	}

	if len(cd.Repositories) == 0 {
		return nil, errors.New("no repositories defined")
	}

	yamlFSLoader := loaders.NewYAMLFSCommandLoader(&pkg.SqlCommandLoader{
		DBConnectionFactory: pkg.OpenDatabaseFromSqletonConnectionLayer,
	})
	yamlLoader := &loaders.YAMLReaderCommandLoader{
		YAMLCommandLoader: &pkg.SqlCommandLoader{
			DBConnectionFactory: pkg.OpenDatabaseFromSqletonConnectionLayer,
		},
	}

	r := repositories.NewRepository(
		repositories.WithDirectories(cd.Repositories),
		repositories.WithUpdateCallback(func(cmd cmds.Command) error {
			description := cmd.Description()
			log.Info().Str("name", description.Name).
				Str("source", description.Source).
				Msg("Updating cmd")
			// TODO(manuel, 2023-04-19) This is where we would recompute the HandlerFunc used below in GET and POST
			return nil
		}),
		repositories.WithRemoveCallback(func(cmd cmds.Command) error {
			description := cmd.Description()
			log.Info().Str("name", description.Name).
				Str("source", description.Source).
				Msg("Removing cmd")
			// TODO(manuel, 2023-04-19) This is where we would recompute the HandlerFunc used below in GET and POST
			// NOTE(manuel, 2023-05-25) Regarding the above TODO, why?
			// We don't need to recompute the func, since it fetches the command at runtime.
			return nil
		}),
		repositories.WithCommandLoader(yamlLoader),
		repositories.WithFSLoader(yamlFSLoader),
	)

	err := r.LoadCommands()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}

	cd.repository = r

	return r, nil
}

func (cd *CommandDirHandler) Serve(server *parka.Server, path string) error {
	repository, err := cd.GetRepository()
	if err != nil {
		return err
	}
	path = strings.TrimSuffix(path, "/")

	server.Router.GET(path+"/data/*path", func(c *gin.Context) {
		commandPath := c.Param("CommandPath")
		commandPath = strings.TrimPrefix(commandPath, "/")
		sqlCommand, ok := GetRepositoryCommand(c, repository, commandPath)
		if !ok {
			c.JSON(404, gin.H{"error": "command not found"})
			return
		}

		jsonProcessorFunc := glazed.CreateJSONProcessor

		// TODO(manuel, 2023-05-25) We can't currently override defaults, since they are parsed up front.
		// For that we would need https://github.com/go-go-golems/glazed/issues/239
		// So for now, we only deal with overrides.

		parserOptions := []glazed.ParserOption{
			glazed.WithReplaceStaticLayer("sqleton-connection", cd.SqletonConnectionLayer.Parameters),
			glazed.WithReplaceStaticLayer("dbt", cd.DbtConnectionLayer.Parameters),
		}

		if cd.Overrides != nil {
			for slug, layer := range cd.Overrides.Layers {
				parserOptions = append(parserOptions, glazed.WithAppendOverrideLayer(slug, layer))
			}
		}
		handle := server.HandleSimpleQueryCommand(sqlCommand,
			glazed.WithCreateProcessor(jsonProcessorFunc),
			glazed.WithParserOptions(parserOptions...),
		)

		handle(c)
	})

	server.Router.GET(path+"/sqleton/*path",
		func(c *gin.Context) {
			// Get command path from the route
			commandPath := strings.TrimPrefix(c.Param("path"), "/")

			// Get repository command
			sqlCommand, ok := GetRepositoryCommand(c, repository, commandPath)
			if !ok {
				c.JSON(404, gin.H{"error": "command not found"})
				return
			}

			name := sqlCommand.Description().Name
			dateTime := time.Now().Format("2006-01-02--15-04-05")
			links := []Link{
				{
					Href:  fmt.Sprintf("/download/%s/%s-%s.csv", commandPath, dateTime, name),
					Text:  "Download CSV",
					Class: "download",
				},
				{
					Href:  fmt.Sprintf("/download/%s/%s-%s.json", commandPath, dateTime, name),
					Text:  "Download JSON",
					Class: "download",
				},
				{
					Href:  fmt.Sprintf("/download/%s/%s-%s.xlsx", commandPath, dateTime, name),
					Text:  "Download Excel",
					Class: "download",
				},
				{
					Href:  fmt.Sprintf("/download/%s/%s-%s.md", commandPath, dateTime, name),
					Text:  "Download Markdown",
					Class: "download",
				},
				{
					Href:  fmt.Sprintf("/download/%s/%s-%s.html", commandPath, dateTime, name),
					Text:  "Download HTML",
					Class: "download",
				},
				{
					Href:  fmt.Sprintf("/download/%s/%s-%s.txt", commandPath, dateTime, name),
					Text:  "Download Text",
					Class: "download",
				},
			}

			var localTemplateLookup render.TemplateLookup

			// TODO(manuel, 2023-05-25) Ignore indexTemplateName for now
			// See https://github.com/go-go-golems/sqleton/issues/162
			_ = cd.IndexTemplateName

			// TODO(manuel, 2023-05-28) How does this link up with CreateProcessorFunc?
			localTemplateLookup = render.NewLookupTemplateFromFS(
				render.WithFS(cd.TemplateFS),
				render.WithBaseDir(cd.TemplateDirectory),
				render.WithPatterns(cd.TemplateDirectory+"/**/*.tmpl.html"),
				render.WithAlwaysReload(cd.DevMode),
			)
			err := localTemplateLookup.Reload()

			if err != nil {
				c.JSON(500, gin.H{"error": "could not create template lookup"})
				return
			}

			dataTablesProcessorFunc := render.NewHTMLTemplateLookupCreateProcessorFunc(
				localTemplateLookup,
				cd.TemplateName,
				render.WithHTMLTemplateOutputFormatterData(
					map[string]interface{}{
						"Links": links,
					},
				),
				render.WithJavascriptRendering(),
			)

			// TODO(manuel, 2023-05-25) We can't currently override defaults, since they are parsed up front.
			// For that we would need https://github.com/go-go-golems/glazed/issues/239
			// So for now, we only deal with overrides.

			parserOptions := []glazed.ParserOption{
				glazed.WithReplaceStaticLayer("sqleton-connection", cd.SqletonConnectionLayer.Parameters),
				glazed.WithReplaceStaticLayer("dbt", cd.DbtConnectionLayer.Parameters),
			}

			if cd.Overrides != nil {
				for slug, layer := range cd.Overrides.Layers {
					parserOptions = append(parserOptions, glazed.WithAppendOverrideLayer(slug, layer))
				}
			}

			handle := server.HandleSimpleQueryCommand(
				sqlCommand,
				glazed.WithCreateProcessor(
					dataTablesProcessorFunc,
				),
				glazed.WithParserOptions(parserOptions...),
			)

			handle(c)
		})

	server.Router.GET(path+"/download/*path", func(c *gin.Context) {
		// get file name at end of path
		index := strings.LastIndex(path, "/")
		if index == -1 {
			c.JSON(500, gin.H{"error": "could not find file name"})
			return
		}
		if index >= len(path)-1 {
			c.JSON(500, gin.H{"error": "could not find file name"})
			return
		}
		fileName := path[index+1:]

		commandPath := strings.TrimPrefix(path[:index], "/")
		sqlCommand, ok := GetRepositoryCommand(c, repository, commandPath)
		if !ok {
			c.JSON(404, gin.H{"error": "command not found"})
			return
		}

		// create a temporary file for glazed output
		tmpFile, err := os.CreateTemp("/tmp", fmt.Sprintf("glazed-output-*.%s", fileName))
		if err != nil {
			c.JSON(500, gin.H{"error": "could not create temporary file"})
			return
		}
		defer os.Remove(tmpFile.Name())

		// now check file suffix for content-type
		glazedOverrides := map[string]interface{}{
			"output-file": tmpFile.Name(),
		}
		if strings.HasSuffix(fileName, ".csv") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "csv"
		} else if strings.HasSuffix(fileName, ".tsv") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "tsv"
		} else if strings.HasSuffix(fileName, ".md") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "markdown"
		} else if strings.HasSuffix(fileName, ".html") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "html"
		} else if strings.HasSuffix(fileName, ".json") {
			glazedOverrides["output"] = "json"
		} else if strings.HasSuffix(fileName, ".yaml") {
			glazedOverrides["yaml"] = "yaml"
		} else if strings.HasSuffix(fileName, ".xlsx") {
			glazedOverrides["output"] = "excel"
		} else if strings.HasSuffix(fileName, ".txt") {
			glazedOverrides["output"] = "table"
			glazedOverrides["table-format"] = "ascii"
		} else {
			c.JSON(500, gin.H{"error": "could not determine output format"})
			return
		}

		parserOptions := []glazed.ParserOption{
			glazed.WithReplaceStaticLayer("sqleton-connection", cd.SqletonConnectionLayer.Parameters),
			glazed.WithReplaceStaticLayer("dbt", cd.DbtConnectionLayer.Parameters),
		}

		if cd.Overrides != nil {
			for slug, layer := range cd.Overrides.Layers {
				parserOptions = append(parserOptions, glazed.WithAppendOverrideLayer(slug, layer))
			}
		}

		// override parameter layers at the end
		parserOptions = append(parserOptions, glazed.WithAppendOverrideLayer("glazed", glazedOverrides))

		handle := server.HandleSimpleQueryOutputFileCommand(
			sqlCommand,
			tmpFile.Name(),
			fileName,
			glazed.WithParserOptions(parserOptions...),
		)

		handle(c)
	})

	return nil
}
