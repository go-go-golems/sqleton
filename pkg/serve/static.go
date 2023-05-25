package serve

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/parka/pkg/render"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

type StaticOptions struct {
	fs fs.FS
}

func (s *StaticFile) Serve(server *pkg.Server, options *StaticOptions, path string) error {
	server.Router.StaticFileFS(
		path,
		s.LocalPath,
		http.FS(options.fs),
	)
	return nil
}

func (s *Static) Serve(server *pkg.Server, options *StaticOptions, path string) error {
	fs := options.fs
	if s.LocalPath != "" {
		fs = pkg.NewAddPrefixPathFS(options.fs, s.LocalPath)
	}
	server.Router.StaticFS(path, http.FS(fs))
	return nil
}

type TemplateOptions struct {
	fs fs.FS
}

func (td *TemplateDir) Serve(server *pkg.Server, options *TemplateOptions, path string) error {
	// this is from server.Run(), where it does the catch all

	// match all remaining paths to the templates
	s.Router.Use(
		func(c *gin.Context) {
			rawPath := c.Request.URL.Path
			if len(rawPath) > 0 && rawPath[0] == '/' {
				trimmedPath := rawPath[1:]
				s.serveMarkdownTemplatePage(c, trimmedPath, nil)
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
		isAbsoluteDir := strings.HasPrefix(contentDir, "/")

		// resolve base dir
		if !isAbsoluteDir {
			baseDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get working directory: %w", err)
			}
			contentDir = baseDir + "/" + contentDir
		}

		localTemplateLookup, err := render.LookupTemplateFromFSReloadable(
			os.DirFS(contentDir),
			"",
			"**/*.tmpl.md",
			"**/*.md",
			"**/*.tmpl.html",
			"**/*.html")
		if err != nil {
			return fmt.Errorf("failed to load local template: %w", err)
		}
		lookups[i] = localTemplateLookup
	}
	serverOptions = append(serverOptions, parka.WithAppendTemplateLookups(lookups...))
	server.Router.SetHTMLTemplate(http.FS(options.fs))
	return nil
}
