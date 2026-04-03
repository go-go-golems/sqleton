package cmds

import (
	"testing"

	"github.com/go-go-golems/glazed/pkg/cmds"
	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/stretchr/testify/require"
)

func TestCompileSetsOptionalBoolFlagDefaultFalse(t *testing.T) {
	compiler := &SqlCommandCompiler{}
	spec := &SqlCommandSpec{
		Name:  "test",
		Short: "test",
		Flags: []*fields.Definition{
			fields.New("only_active", fields.TypeBool),
		},
		Query: "SELECT 1",
	}

	cmd, err := compiler.Compile(spec)
	require.NoError(t, err)

	flag, ok := cmd.Description().GetDefaultFlags().Get("only_active")
	require.True(t, ok)
	require.NotNil(t, flag.Default)
	require.Equal(t, false, *flag.Default)

	require.Nil(t, spec.Flags[0].Default)
}

func TestCompilePreservesExplicitBoolFlagDefault(t *testing.T) {
	compiler := &SqlCommandCompiler{}
	spec := &SqlCommandSpec{
		Name:  "test",
		Short: "test",
		Flags: []*fields.Definition{
			fields.New("enabled", fields.TypeBool, fields.WithDefault(true)),
		},
		Query: "SELECT 1",
	}

	cmd, err := compiler.Compile(spec)
	require.NoError(t, err)

	flag, ok := cmd.Description().GetDefaultFlags().Get("enabled")
	require.True(t, ok)
	require.NotNil(t, flag.Default)
	require.Equal(t, true, *flag.Default)
}

func TestCompileDoesNotDefaultRequiredBoolFlag(t *testing.T) {
	compiler := &SqlCommandCompiler{}
	spec := &SqlCommandSpec{
		Name:  "test",
		Short: "test",
		Flags: []*fields.Definition{
			fields.New("enabled", fields.TypeBool, fields.WithRequired(true)),
		},
		Query: "SELECT 1",
	}

	cmd, err := compiler.Compile(spec, cmds.WithType("sql"))
	require.NoError(t, err)

	flag, ok := cmd.Description().GetDefaultFlags().Get("enabled")
	require.True(t, ok)
	require.Nil(t, flag.Default)
}
