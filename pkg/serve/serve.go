package serve

import (
	"context"
	parka "github.com/go-go-golems/parka/pkg"
	command_dir "github.com/go-go-golems/sqleton/pkg/serve/command-dir"
	"github.com/go-go-golems/sqleton/pkg/serve/config"
	static_dir "github.com/go-go-golems/sqleton/pkg/serve/static-dir"
	static_file "github.com/go-go-golems/sqleton/pkg/serve/static-file"
	template_dir "github.com/go-go-golems/sqleton/pkg/serve/template-dir"
	"golang.org/x/sync/errgroup"
)

// ConfigFileHandler contains everything needed to serve a config file
type ConfigFileHandler struct {
	Config *config.Config

	CommandDirectoryOptions  []command_dir.CommandDirHandlerOption
	TemplateDirectoryOptions []template_dir.TemplateDirHandlerOption

	// ConfigFileLocation is an optional path to the config file on disk in case it needs to be reloaded
	ConfigFileLocation        string
	commandDirectoryHandlers  []*command_dir.CommandDirHandler
	templateDirectoryHandlers []*template_dir.TemplateDirHandler
}

type ConfigFileHandlerOption func(*ConfigFileHandler)

func WithAppendCommandDirHandlerOptions(options ...command_dir.CommandDirHandlerOption) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.CommandDirectoryOptions = append(handler.CommandDirectoryOptions, options...)
	}
}

func WithAppendTemplateDirHandlerOptions(options ...template_dir.TemplateDirHandlerOption) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.TemplateDirectoryOptions = append(handler.TemplateDirectoryOptions, options...)
	}
}

func WithConfigFileLocation(location string) ConfigFileHandlerOption {
	return func(handler *ConfigFileHandler) {
		handler.ConfigFileLocation = location
	}
}

func NewConfigFileHandler(config *config.Config, options ...ConfigFileHandlerOption) *ConfigFileHandler {
	handler := &ConfigFileHandler{
		Config: config,
	}

	for _, option := range options {
		option(handler)
	}

	return handler
}

func (cfh *ConfigFileHandler) Serve(server *parka.Server) error {
	// NOTE(manuel, 2023-05-26)
	// This could be extracted to a "parseConfigFile", so that we can easily add preconfigured handlers that
	// can deal with embeddedFS

	for _, route := range cfh.Config.Routes {
		if route.CommandDirectory != nil {
			cdh, err := command_dir.NewCommandDirHandlerFromConfig(route.CommandDirectory, cfh.CommandDirectoryOptions...)
			if err != nil {
				return err
			}

			cfh.commandDirectoryHandlers = append(cfh.commandDirectoryHandlers, cdh)

			err = cdh.Serve(server, route.Path)
			if err != nil {
				return err
			}

			continue
		}

		if route.TemplateDirectory != nil {
			tdh, err := template_dir.NewTemplateDirHandlerFromConfig(route.TemplateDirectory, cfh.TemplateDirectoryOptions...)
			if err != nil {
				return err
			}

			cfh.templateDirectoryHandlers = append(cfh.templateDirectoryHandlers, tdh)

			err = tdh.Serve(server, route.Path)
			if err != nil {
				return err
			}

			continue
		}

		if route.StaticFile != nil {
			sfh := static_file.NewStaticFileHandlerFromConfig(route.StaticFile)
			err := sfh.Serve(server, route.Path)
			if err != nil {
				return err
			}

			continue
		}

		if route.Static != nil {
			sdh := static_dir.NewStaticDirHandlerFromConfig(route.Static)
			err := sdh.Serve(server, route.Path)
			if err != nil {
				return err
			}

			continue
		}
	}

	return nil
}

// Watch watches the config for changes and updates the server accordingly.
// Because this will register / unregister routes, this will probably need to be handled
// at a level where we can restart the gin server altogether.
func (cfh *ConfigFileHandler) Watch(ctx context.Context) error {
	errGroup, ctx2 := errgroup.WithContext(ctx)
	for _, cdh := range cfh.commandDirectoryHandlers {
		r, err := cdh.GetRepository()
		if err != nil {
			return err
		}
		errGroup.Go(func() error {
			return r.Watch(ctx2)
		})
	}

	return errGroup.Wait()
}
