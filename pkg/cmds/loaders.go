package cmds

import (
	"fmt"
	"io"
	"io/fs"

	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type SqlCommandLoader struct {
	DBConnectionFactory sql.DBConnectionFactory
}

var _ loaders.CommandLoader = (*SqlCommandLoader)(nil)

func (scl *SqlCommandLoader) LoadCommands(
	f fs.FS, entryName string,
	options []cmds.CommandDescriptionOption,
	aliasOptions []alias.Option,
) ([]cmds.Command, error) {
	r, err := f.Open(entryName)
	if err != nil {
		return nil, err
	}
	defer func(r fs.File) {
		_ = r.Close()
	}(r)

	return loaders.LoadCommandOrAliasFromReader(
		r,
		scl.loadSqlCommandFromReader,
		options,
		aliasOptions)
}

func (scl *SqlCommandLoader) IsFileSupported(f fs.FS, fileName string) bool {
	return loaders.CheckYamlFileType(f, fileName, "sqleton")
}

func (scl *SqlCommandLoader) loadSqlCommandFromReader(
	s io.Reader,
	options []cmds.CommandDescriptionOption,
	_ []alias.Option,
) ([]cmds.Command, error) {
	scd := &SqlCommandDescription{}
	err := yaml.NewDecoder(s).Decode(scd)
	if err != nil {
		return nil, err
	}

	if scd.Type == "" {
		scd.Type = "sqleton"
	} else if scd.Type != "sqleton" {
		return nil, fmt.Errorf("invalid type: %s", scd.Type)
	}

	options_ := []cmds.CommandDescriptionOption{
		cmds.WithShort(scd.Short),
		cmds.WithLong(scd.Long),
		cmds.WithFlags(scd.Flags...),
		cmds.WithArguments(scd.Arguments...),
		cmds.WithLayersList(scd.Layers...),
		cmds.WithType(scd.Type),
		cmds.WithTags(scd.Tags...),
		cmds.WithMetadata(scd.Metadata),
		cmds.WithLayout(&layout.Layout{
			Sections: scd.Layout,
		}),
	}
	options_ = append(options_, options...)

	sq, err := NewSqlCommand(
		cmds.NewCommandDescription(
			scd.Name,
		),
		WithDbConnectionFactory(scl.DBConnectionFactory),
		WithQuery(scd.Query),
		WithSubQueries(scd.SubQueries),
	)
	if err != nil {
		return nil, err
	}

	for _, option := range options_ {
		option(sq.Description())
	}

	if !sq.IsValid() {
		return nil, errors.New("Invalid command")
	}

	return []cmds.Command{sq}, nil
}
