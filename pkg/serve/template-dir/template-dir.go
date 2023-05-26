package template_dir

import (
	"fmt"
	"github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/render"
	"github.com/go-go-golems/sqleton/pkg/serve/config"
	"io/fs"
	"os"
	"strings"
)

type TemplateDirHandler struct {
	fs                       fs.FS
	LocalDirectory           string
	IndexTemplateName        string
	MarkdownBaseTemplateName string
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

func NewTemplateDirHandler(options ...TemplateDirHandlerOption) *TemplateDirHandler {
	handler := &TemplateDirHandler{}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func NewTemplateDirHandlerFromConfig(td *config.TemplateDir, options ...TemplateDirHandlerOption) (*TemplateDirHandler, error) {
	handler := &TemplateDirHandler{
		IndexTemplateName: td.IndexTemplateName,
	}
	WithLocalDirectory(td.LocalDirectory)(handler)

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
	rendererOptions := append(
		handler.rendererOptions,
		render.WithPrependTemplateLookups(templateLookup),
		render.WithIndexTemplateName(handler.IndexTemplateName),
	)
	r, err := render.NewRenderer(rendererOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to load local template: %w", err)
	}
	handler.renderer = r

	return handler, nil
}

func (td *TemplateDirHandler) Serve(server *pkg.Server, path string) error {
	// TODO(manuel, 2023-05-26) This is a hack because we currently mix and match content with commands.
	server.Router.Use(td.renderer.HandleWithTrimPrefix(path, nil))

	return nil
}
