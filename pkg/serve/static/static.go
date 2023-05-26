package static

import (
	"github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/sqleton/pkg/serve/config"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

type StaticHandler struct {
	fs        fs.FS
	localPath string
}

type StaticHandlerOption func(handler *StaticHandler)

func WithDefaultFS(fs fs.FS, localPath string) StaticHandlerOption {
	return func(handler *StaticHandler) {
		if handler.fs == nil {
			handler.fs = fs
			handler.localPath = localPath
		}
	}
}

func WithLocalPath(localPath string) StaticHandlerOption {
	return func(handler *StaticHandler) {
		if localPath != "" {
			if localPath[0] == '/' {
				handler.fs = os.DirFS(localPath)
			} else {
				handler.fs = os.DirFS(localPath)
			}
			handler.localPath = strings.TrimPrefix(localPath, "/")
		}
	}
}

func NewStaticHandler(options ...StaticHandlerOption) *StaticHandler {
	handler := &StaticHandler{}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func NewStaticHandlerFromConfig(sh *config.Static, options ...StaticHandlerOption) *StaticHandler {
	handler := &StaticHandler{
		localPath: sh.LocalPath,
	}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func (s *StaticHandler) Serve(server *pkg.Server, path string) error {
	fs := s.fs
	if s.localPath != "" {
		fs = pkg.NewAddPrefixPathFS(s.fs, s.localPath)
	}
	server.Router.StaticFS(path, http.FS(fs))
	return nil
}
