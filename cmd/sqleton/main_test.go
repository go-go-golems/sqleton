package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	sqleton_cmds "github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"
)

func TestSQLiteSmoke(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "smoke.db")
	commandPath := filepath.Join(tmpDir, "active-widgets.sql")

	createSmokeSQLiteDB(t, dbPath)
	writeSmokeCommandFile(t, commandPath)

	t.Run("query", func(t *testing.T) {
		rows := runSqletonJSON(t, tmpDir,
			"query",
			"--db-type", "sqlite",
			"--database", dbPath,
			"--output", "json",
			"SELECT id, name FROM widgets ORDER BY id",
		)

		require.Len(t, rows, 3)
		require.Equal(t, float64(1), rows[0]["id"])
		require.Equal(t, "alpha", rows[0]["name"])
		require.Equal(t, float64(3), rows[2]["id"])
		require.Equal(t, "gamma", rows[2]["name"])
	})

	t.Run("run-command", func(t *testing.T) {
		rows := runSqletonJSON(t, tmpDir,
			"run-command",
			commandPath,
			"--",
			"--db-type", "sqlite",
			"--database", dbPath,
			"--output", "json",
			"--only-active",
		)

		require.Len(t, rows, 2)
		require.Equal(t, "alpha", rows[0]["name"])
		require.Equal(t, "gamma", rows[1]["name"])
	})
}

func TestConfiguredRepositoryDiscoverySmoke(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	repoDir := filepath.Join(tmpDir, "repo")
	dbPath := filepath.Join(tmpDir, "discovery.db")
	commandPath := filepath.Join(repoDir, "smoke-widgets.sql")
	aliasPath := filepath.Join(repoDir, "smoke-active-widgets.alias.yaml")

	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))
	require.NoError(t, os.MkdirAll(repoDir, 0o755))

	createSmokeSQLiteDB(t, dbPath)
	writeSmokeCommandFile(t, commandPath)
	writeSmokeAliasFile(t, aliasPath)

	t.Run("discovered-sql-command", func(t *testing.T) {
		rows := runSqletonJSONWithEnv(t, homeDir, map[string]string{
			"SQLETON_REPOSITORIES": repoDir,
		},
			"smoke-widgets",
			"--db-type", "sqlite",
			"--database", dbPath,
			"--only-active=false",
			"--output", "json",
		)

		require.Len(t, rows, 3)
		require.Equal(t, "alpha", rows[0]["name"])
		require.Equal(t, "beta", rows[1]["name"])
		require.Equal(t, "gamma", rows[2]["name"])
	})

	t.Run("discovered-alias-command", func(t *testing.T) {
		rows := runSqletonJSONWithEnv(t, homeDir, map[string]string{
			"SQLETON_REPOSITORIES": repoDir,
		},
			"smoke-active-widgets",
			"--db-type", "sqlite",
			"--database", dbPath,
			"--output", "json",
		)

		require.Len(t, rows, 2)
		require.Equal(t, "alpha", rows[0]["name"])
		require.Equal(t, "gamma", rows[1]["name"])
	})
}

func TestCLIHelperProcess(t *testing.T) {
	if os.Getenv("SQLETON_TEST_SUBPROCESS") != "1" {
		return
	}

	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	require.NotEqual(t, -1, separator, "missing subprocess argument separator")

	os.Args = append([]string{"sqleton"}, os.Args[separator+1:]...)
	main()
	os.Exit(0)
}

func createSmokeSQLiteDB(t *testing.T, dbPath string) {
	t.Helper()

	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec(`
CREATE TABLE widgets (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    active INTEGER NOT NULL
);

INSERT INTO widgets (name, active)
VALUES
    ('alpha', 1),
    ('beta', 0),
    ('gamma', 1);
`)
	require.NoError(t, err)
}

func writeSmokeCommandFile(t *testing.T, commandPath string) {
	t.Helper()

	spec := &sqleton_cmds.SqlCommandSpec{
		Name:  "smoke-widgets",
		Short: "List widgets",
		Flags: []*fields.Definition{
			fields.New("only_active", fields.TypeBool),
		},
		Query: `
SELECT id, name
FROM widgets
WHERE active = 1 OR NOT {{ .only_active }}
ORDER BY id
`,
	}

	contents, err := sqleton_cmds.MarshalSpecToSQLFile(spec)
	require.NoError(t, err)

	err = os.WriteFile(commandPath, []byte(contents), 0o644)
	require.NoError(t, err)
}

func writeSmokeAliasFile(t *testing.T, aliasPath string) {
	t.Helper()

	contents := `name: smoke-active-widgets
aliasFor: smoke-widgets
flags:
  only-active: "true"
`

	err := os.WriteFile(aliasPath, []byte(contents), 0o644)
	require.NoError(t, err)
}

func runSqletonJSON(t *testing.T, homeDir string, args ...string) []map[string]interface{} {
	t.Helper()

	return runSqletonJSONWithEnv(t, homeDir, nil, args...)
}

func runSqletonJSONWithEnv(t *testing.T, homeDir string, extraEnv map[string]string, args ...string) []map[string]interface{} {
	t.Helper()

	packageDir, err := os.Getwd()
	require.NoError(t, err)

	cmdArgs := append([]string{"-test.run=TestCLIHelperProcess", "--"}, args...)
	cmd := exec.Command(os.Args[0], cmdArgs...)
	cmd.Dir = packageDir
	env := append(
		os.Environ(),
		"SQLETON_TEST_SUBPROCESS=1",
		"HOME="+homeDir,
		"XDG_CONFIG_HOME="+filepath.Join(homeDir, ".config"),
		"NO_COLOR=1",
		"CLICOLOR=0",
	)
	for k, v := range extraEnv {
		env = append(env, k+"="+v)
	}
	cmd.Env = env

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	require.NoError(t, err, "stderr:\n%s", stderr.String())

	var rows []map[string]interface{}
	err = json.Unmarshal(stdout.Bytes(), &rows)
	require.NoError(t, err, "stdout:\n%s\nstderr:\n%s", stdout.String(), stderr.String())

	return rows
}
