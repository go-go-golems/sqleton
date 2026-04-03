package cmds

import (
	"testing"
	"testing/fstest"

	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	"github.com/stretchr/testify/require"
)

func TestSqlCommandLoaderIsFileSupportedRequiresSqletonPreamble(t *testing.T) {
	fsys := fstest.MapFS{
		"queries/command.sql": {
			Data: []byte("/* sqleton\nname: command\nshort: Command\n*/\nSELECT 1;\n"),
		},
		"queries/plain.sql": {
			Data: []byte("SELECT 1;\n"),
		},
		"queries/command.alias.yaml": {
			Data: []byte("name: short\naliasFor: command\n"),
		},
	}

	loader := &SqlCommandLoader{}

	require.True(t, loader.IsFileSupported(fsys, "queries/command.sql"))
	require.False(t, loader.IsFileSupported(fsys, "queries/plain.sql"))
	require.True(t, loader.IsFileSupported(fsys, "queries/command.alias.yaml"))
}

func TestSqlCommandLoaderSkipsPlainSQLDuringRepositoryDiscovery(t *testing.T) {
	fsys := fstest.MapFS{
		"queries/command.sql": {
			Data: []byte("/* sqleton\nname: command\nshort: Command\n*/\nSELECT 1;\n"),
		},
		"queries/plain.sql": {
			Data: []byte("SELECT 1;\n"),
		},
	}

	loader := &SqlCommandLoader{}

	loaded, err := loaders.LoadCommandsFromFS(
		fsys,
		"queries",
		"test",
		loader,
		[]cmds.CommandDescriptionOption{},
		[]alias.Option{},
	)
	require.NoError(t, err)
	require.Len(t, loaded, 1)
	require.Equal(t, "command", loaded[0].Description().Name)
}
