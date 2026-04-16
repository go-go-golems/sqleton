package cmds

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-go-golems/glazed/pkg/cli"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/go-go-golems/glazed/pkg/cmds/sources"
	"github.com/go-go-golems/glazed/pkg/cmds/values"
	"github.com/stretchr/testify/require"
)

func TestBuildSqletonCommandConfigPlanUsesExplicitFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "command-config.yaml")
	require.NoError(t, os.WriteFile(configPath, []byte("sql-connection:\n  db-type: sqlite\n"), 0o644))

	parsed := newParsedCommandSettings(t, map[string]interface{}{
		"config-file": configPath,
	})

	plan, err := BuildSqletonCommandConfigPlan(parsed)
	require.NoError(t, err)

	files, _, err := plan.Resolve(context.Background())
	require.NoError(t, err)
	require.Len(t, files, 1)
	require.Equal(t, filepath.Clean(configPath), files[0].Path)
	require.Equal(t, "explicit-command-config", files[0].SourceName)
	require.Equal(t, "command-config", files[0].SourceKind)
}

func TestBuildSqletonCommandConfigPlanEmptyPathSkips(t *testing.T) {
	parsed := newParsedCommandSettings(t, map[string]interface{}{})

	plan, err := BuildSqletonCommandConfigPlan(parsed)
	require.NoError(t, err)

	files, _, err := plan.Resolve(context.Background())
	require.NoError(t, err)
	require.Empty(t, files)
}

func newParsedCommandSettings(t *testing.T, commandSettings map[string]interface{}) *values.Values {
	t.Helper()

	section, err := cli.NewCommandSettingsSection()
	require.NoError(t, err)

	parsed := values.New()
	err = sources.Execute(
		schema.NewSchema(schema.WithSections(section)),
		parsed,
		sources.FromMap(map[string]map[string]interface{}{
			cli.CommandSettingsSlug: commandSettings,
		}),
	)
	require.NoError(t, err)

	return parsed
}
