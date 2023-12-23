package cmds

import (
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	"strings"
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
	return strings.HasSuffix(fileName, ".yaml") || strings.HasSuffix(fileName, ".yml")
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

	options_ := []cmds.CommandDescriptionOption{
		cmds.WithShort(scd.Short),
		cmds.WithLong(scd.Long),
		cmds.WithFlags(scd.Flags...),
		cmds.WithArguments(scd.Arguments...),
		cmds.WithLayers(scd.Layers...),
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
