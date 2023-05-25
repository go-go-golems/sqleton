package serve

import (
	"fmt"
	"github.com/gin-gonic/gin"
	cmds2 "github.com/go-go-golems/clay/pkg/cmds"
	"github.com/go-go-golems/clay/pkg/repositories"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/go-go-golems/glazed/pkg/help"
	pkg2 "github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/glazed"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/sqleton/pkg"
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

func (cd *CommandDir) GetRepository() (*repositories.Repository, error) {
	if len(cd.Repositories) == 0 {
		return nil, errors.New("no repositories defined")
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
	)

	locations := cmds2.CommandLocations{
		Repositories: cd.Repositories,
	}

	yamlLoader := loaders.NewYAMLFSCommandLoader(&pkg.SqlCommandLoader{
		DBConnectionFactory: pkg.OpenDatabaseFromSqletonConnectionLayer,
	})
	commandLoader := cmds2.NewCommandLoader[cmds.Command](&locations)
	// TODO(manuel, 2023-05-25) Add a way to configure serving the help of commands in a CommandDir
	// See https://github.com/go-go-golems/sqleton/issues/163
	helpSystem := help.NewHelpSystem()
	commands, aliases, err := commandLoader.LoadCommands(yamlLoader, helpSystem)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error initializing commands: %s\n", err)
		os.Exit(1)
	}
	r.Add(commands...)
	for _, alias := range aliases {
		r.Add(alias)
	}

	return r, nil
}

type ServeOptions struct {
	DevMode                  bool
	DefaultTemplateName      string
	DefaultIndexTemplateName string
	DefaultTemplateFS        fs.FS
	DefaultTemplateDirectory string
	DbtConnectionLayer       *layers.ParsedParameterLayer
	SqletonConnectionLayer   *layers.ParsedParameterLayer
}

func (cd *CommandDir) Serve(server *pkg2.Server, options *ServeOptions, path string) error {
	repository, err := cd.GetRepository()
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
			glazed.WithReplaceStaticLayer("sqleton-connection", options.SqletonConnectionLayer.Parameters),
			glazed.WithReplaceStaticLayer("dbt", options.DbtConnectionLayer.Parameters),
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

			var templateFS fs.FS = options.DefaultTemplateFS
			templateDirectory := options.DefaultTemplateDirectory
			templateName := options.DefaultTemplateName
			indexTemplateName := options.DefaultIndexTemplateName

			if cd.TemplateDirectory != "" {
				if cd.TemplateDirectory[0] == '/' {
					templateFS = os.DirFS("/")
				} else {
					templateFS = os.DirFS(".")
				}
			}
			if cd.TemplateName != "" {
				templateName = cd.TemplateName
			}
			if cd.IndexTemplateName != "" {
				indexTemplateName = cd.IndexTemplateName
			}

			// TODO(manuel, 2023-05-25) Ignore indexTemplateName for now
			// See https://github.com/go-go-golems/sqleton/issues/162
			_ = indexTemplateName

			localTemplateLookup, err = render.LookupTemplateFromFSReloadable(
				templateFS,
				templateDirectory,
				templateDirectory+"/**/*.tmpl.html",
			)

			if err != nil {
				c.JSON(500, gin.H{"error": "could not create template lookup"})
				return
			}

			dataTablesProcessorFunc := render.NewHTMLTemplateLookupCreateProcessorFunc(
				localTemplateLookup,
				templateName,
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
				glazed.WithReplaceStaticLayer("sqleton-connection", options.SqletonConnectionLayer.Parameters),
				glazed.WithReplaceStaticLayer("dbt", options.DbtConnectionLayer.Parameters),
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
			glazed.WithReplaceStaticLayer("sqleton-connection", options.SqletonConnectionLayer.Parameters),
			glazed.WithReplaceStaticLayer("dbt", options.DbtConnectionLayer.Parameters),
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
