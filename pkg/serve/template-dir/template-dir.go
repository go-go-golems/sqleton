package template_dir

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/sqleton/pkg/serve/config"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

type TemplateDirHandler struct {
	fs                       fs.FS
	LocalDirectory           string
	IndexTemplateName        string
	MarkdownBaseTemplateName string
	AdditionalData           map[string]interface{}
	templateLookups          []render.TemplateLookup
	rendererOptions          []render.RendererOption
	renderer                 *render.Renderer
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

func WithAppendRendererOptions(rendererOptions ...render.RendererOption) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) {
		handler.rendererOptions = append(handler.rendererOptions, rendererOptions...)
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

func WithAppendTemplateLookups(templateLookups ...render.TemplateLookup) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) {
		handler.templateLookups = append(handler.templateLookups, templateLookups...)
	}
}

func WithPrependTemplateLookups(templateLookups ...render.TemplateLookup) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) {
		handler.templateLookups = append(templateLookups, handler.templateLookups...)
	}
}

func WithReplaceTemplateLookups(templateLookups ...render.TemplateLookup) TemplateDirHandlerOption {
	return func(handler *TemplateDirHandler) {
		handler.templateLookups = templateLookups
	}
}

func NewTemplateDirHandler(options ...TemplateDirHandlerOption) *TemplateDirHandler {
	handler := &TemplateDirHandler{}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func NewTemplateDirHandlerFromConfig(td *config.TemplateDir, options ...TemplateDirHandlerOption) (*TemplateDirHandler, error) {
	handler := &TemplateDirHandler{
		LocalDirectory:    td.LocalDirectory,
		IndexTemplateName: td.IndexTemplateName,
		AdditionalData:    td.AdditionalData,
	}

	for _, option := range options {
		option(handler)
	}
	templateLookup, err := render.LookupTemplateFromFSReloadable(
		handler.fs,
		handler.LocalDirectory,
		"**/*.tmpl.md",
		"**/*.md",
		"**/*.tmpl.html",
		"**/*.html")

	if err != nil {
		return nil, fmt.Errorf("failed to load local template: %w", err)
	}
	r, err := render.NewRenderer(
		render.WithPrependTemplateLookups(templateLookup),
		render.WithAppendTemplateLookups(handler.templateLookups...))
	if err != nil {
		return nil, fmt.Errorf("failed to load local template: %w", err)
	}
	handler.renderer = r

	return handler, nil
}

func (td *TemplateDirHandler) Serve(server *pkg.Server, path string) error {
	server.Router.GET(path+"/*path", func(c *gin.Context) {
		page := strings.TrimPrefix(c.Param("path"), "/")
		if page == "" {
			page = td.IndexTemplateName
		} else if strings.HasSuffix(page, "/") {
			page = page + td.IndexTemplateName
		}

		err := td.renderer.Render(c, c.Writer, page, td.AdditionalData)
		if err != nil {
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	})

	return nil
}
