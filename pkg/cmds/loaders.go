package cmds

import (
	"io/fs"

	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/pkg/errors"
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

	sourceKind := DetectSourceKind(entryName)
	switch sourceKind {
	case SourceSQLCommand:
		spec, err := ParseSQLFileSpecFromReader(entryName, r)
		if err != nil {
			return nil, err
		}

		compiler := &SqlCommandCompiler{
			DBConnectionFactory: scl.DBConnectionFactory,
		}
		cmd, err := compiler.Compile(spec, options...)
		if err != nil {
			return nil, err
		}
		return []cmds.Command{cmd}, nil

	case SourceYAMLAlias:
		a, err := alias.NewCommandAliasFromYAML(r, aliasOptions...)
		if err != nil {
			return nil, err
		}
		return []cmds.Command{a}, nil

	case SourceUnknown:
		return nil, errors.Errorf("unsupported sqleton source kind for %s", entryName)
	}

	return nil, errors.Errorf("unsupported sqleton source kind for %s", entryName)
}

func (scl *SqlCommandLoader) IsFileSupported(f fs.FS, fileName string) bool {
	switch DetectSourceKind(fileName) {
	case SourceSQLCommand, SourceYAMLAlias:
		return true
	case SourceUnknown:
		return false
	}

	return false
}
