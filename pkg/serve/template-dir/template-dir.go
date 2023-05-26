package serve

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/sqleton/pkg/serve/config"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

type TemplateDirHandler struct {
	fs                fs.FS
	LocalDirectory    string
	IndexTemplateName string
	AdditionalData    map[string]interface{}
}

type TemplateDirHandlerOption func(handler *TemplateDirHandler)

func WithDefaultFS(fs fs.FS, localPath string) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) {
		if handler.fs == nil {
			handler.fs = fs
			handler.LocalDirectory = localPath
		}
	}
}

func WithLocalDirectory(localPath string) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) {
		if localPath != "" {
			if localPath[0] == '/' {
				handler.fs = os.DirFS(localPath)
			} else {
				handler.fs = os.DirFS(localPath)
			}
			handler.LocalDirectory = strings.TrimPrefix(localPath, "/")
		}
	}
}

func NewTemplateDirHandler(options ...TemplateDirHandlerOption) *TemplateDirHandler {
	handler := &TemplateDirHandler{}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func NewTemplateDirHandlerFromConfig(td *config.TemplateDir, options ...TemplateDirHandlerOption) *TemplateDirHandler {
	handler := &TemplateDirHandler{
		LocalDirectory:    td.LocalDirectory,
		IndexTemplateName: td.IndexTemplateName,
		AdditionalData:    td.AdditionalData,
	}
	for _, option := range options {
		option(handler)
	}
	return handler
}

// RenderTemplate takes a templateLookup function, a page name and additional data, and
// renders a full resulting HTML page.
//
// If markdown content is found (either a .tmpl.md or a .md file), the markdown is rendered and converted
// to HTML, and then passed to the base.tmpl.html file for final HTML rendering).
//
// TODO(manuel, 2023-05-26) The whole handling of render template lookup can probably be packaged in its own full package
// There is quite a lot of things covered here:
// - mapping URLs to actual templates
// - loading templates from local directories or providing other ways of generating ultimately HTML content
//   - static markdown content
//   - templated markdown content
//   - static HTML content
//   - templated HTML content
//
// - embedding generated content into higher level templates (say, base.tmpl.html to serve markdown page)
// - providing additional data
//
// In a way, this is actually some of what the parka.Server class tries to do, with its list of template lookup functions.
func RenderTemplate(w io.Writer, lookup render.TemplateLookup, page string, data interface{}) error {
	// first, we will check if we have markdown contexnt
	markdown := ""
	// first, check for a tmpl.md template
	t, err := lookup(page + ".tmpl.md")
	if err != nil {
		return err
	}

	if t != nil {
		markdown, err := render.RenderMarkdownTemplateToHTML(t, data)
		if err != nil {
			return fmt.Errorf("failed to render markdown template: %w", err)
		}
	}

}

func (td *TemplateDirHandler) Serve(server *pkg.Server, path string) error {
	templateLookup, err := render.LookupTemplateFromFSReloadable(
		td.fs,
		td.LocalDirectory,
		"**/*.tmpl.md",
		"**/*.md",
		"**/*.tmpl.html",
		"**/*.html")
	if err != nil {
		return fmt.Errorf("failed to load local template: %w", err)
	}
	lookups[i] = templateLookup

	server.Router.GET(path+"/*path", func(c *gin.Context) {
		page := strings.TrimPrefix(c.Param("path"), "/")
		// The following is lifted from ServeMarkdownTemplatePage from parka.Server
		// which can probably be consolidated into this approach and remove the list of template lookup handlers
		// (or merge the template lookup mechanism in the template dir handling mechanism)

		// first, check for a markdown template or markdown file
		t, err := templateLookup(page + ".tmpl.md")
	})

	// NOTE(manuel, 2023-05-26) This needs to be extracted at a higher catch all path level in the config filer
	// This is currently done with a middleware, but it should be a GET route
	// match all remaining paths to the templates
	server.Router.Use(
		func(c *gin.Context) {
			rawPath := c.Request.URL.Path
			if len(rawPath) > 0 && rawPath[0] == '/' {
				trimmedPath := rawPath[1:]
				server.ServeMarkdownTemplatePage(c, trimmedPath, nil)
				return
			}
			c.Next()
		})

	// TODO(manuel, 2023-05-25) Kill the whole staticPaths / templateLookups stuff
	//
	// This is the current way to serve templates, through this convoluted templateLookup mechanism
	// that we might be able to kill entirely.
	//
	// It looks like the new method of declaring the layout in a config struct / config file
	// and then repeatedly applying works much better.
	//
	// This should also allow us to watch and reload the config file,
	//
	// See: https://github.com/go-go-golems/sqleton/issues/164
	lookups := make([]render.TemplateLookup, len(contentDirs))
	for i, contentDir := range contentDirs {
	}
	serverOptions = append(serverOptions, parka.WithAppendTemplateLookups(lookups...))
	server.Router.SetHTMLTemplate(http.FS(options.fs))
	return nil
}
