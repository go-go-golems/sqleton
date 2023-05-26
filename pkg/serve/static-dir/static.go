package static_dir

import (
	"github.com/go-go-golems/parka/pkg"
	"github.com/go-go-golems/sqleton/pkg/serve/config"
	"io/fs"
	"net/http"
	"os"
	"strings"
)

type StaticDirHandler struct {
	fs        fs.FS
	localPath string
}

type StaticDirHandlerOption func(handler *StaticDirHandler)

func WithDefaultFS(fs fs.FS, localPath string) StaticDirHandlerOption {
	return func(handler *StaticDirHandler) {
		if handler.fs == nil {
			handler.fs = fs
			handler.localPath = localPath
		}
	}
}

func WithLocalPath(localPath string) StaticDirHandlerOption {
	return func(handler *StaticDirHandler) {
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

func NewStaticDirHandler(options ...StaticDirHandlerOption) *StaticDirHandler {
	handler := &StaticDirHandler{}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func NewStaticDirHandlerFromConfig(sh *config.Static, options ...StaticDirHandlerOption) *StaticDirHandler {
	handler := &StaticDirHandler{
		localPath: sh.LocalPath,
	}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func (s *StaticDirHandler) Serve(server *pkg.Server, path string) error {
	fs := s.fs
	if s.localPath != "" {
		fs = pkg.NewAddPrefixPathFS(s.fs, s.localPath)
	}
	server.Router.StaticFS(path, http.FS(fs))
	return nil
}
