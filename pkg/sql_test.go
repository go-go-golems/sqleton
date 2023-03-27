package pkg

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"testing"

	// sqlite
	_ "github.com/mattn/go-sqlite3"
)

// Here we do a bunch of unit tests in a pretty end to end style by using an in memory SQLite database.

func createDB(_ map[string]*layers.ParsedParameterLayer) (*sqlx.DB, error) {
	db, err := sqlx.Connect("sqlite3", ":memory:")
	if err != nil {
		return nil, err
	}

	// create test table
	_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		return nil, err
	}
	// insert test data
	testData := []struct {
		id   int
		name string
	}{
		{1, "test1"},
		{2, "test2"},
		{3, "test3"},
	}
	for _, d := range testData {
		_, err = db.Exec("INSERT INTO test (id, name) VALUES (?, ?)", d.id, d.name)
		if err != nil {
			return nil, err
		}
	}

	// second table for a join
	_, err = db.Exec("CREATE TABLE test2 (id INTEGER PRIMARY KEY, test_id INTEGER, name TEXT)")
	if err != nil {
		return nil, err
	}
	// insert test data
	testData2 := []struct {
		id      int
		test_id int
		name    string
	}{
		{1, 1, "test1_1"},
		{2, 1, "test1_2"},
		{3, 2, "test2_3"},
	}
	for _, d := range testData2 {
		_, err = db.Exec("INSERT INTO test2 (id, test_id, name) VALUES (?, ?, ?)", d.id, d.test_id, d.name)
		if err != nil {
			return nil, err
		}
	}

	return db, nil
}

func TestSimpleRender(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery("SELECT * FROM test"),
	)
	require.NoError(t, err)

	query, err := s.RenderQuery(nil)
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM test", query)
}

func TestSimpleTemplateRender(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery("SELECT * FROM {{.table}}"),
	)
	require.NoError(t, err)

	query, err := s.RenderQuery(map[string]interface{}{
		"table": "test",
	})
	require.NoError(t, err)
	require.Equal(t, "SELECT * FROM test", query)
}

func TestSimpleRun(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery("SELECT * FROM test"),
	)
	require.NoError(t, err)

	gp := cmds.NewSimpleGlazeProcessor()
	err = s.Run(context.Background(), map[string]*layers.ParsedParameterLayer{}, map[string]interface{}{}, gp)
	require.NoError(t, err)

	rows := gp.GetTable().Rows
	require.Equal(t, 3, len(rows))
	row := rows[0].GetValues()
	require.Equal(t, int64(1), row["id"])
	require.Equal(t, "test1", row["name"])
	row = rows[1].GetValues()
	require.Equal(t, int64(2), row["id"])
	require.Equal(t, "test2", row["name"])
	row = rows[2].GetValues()
	require.Equal(t, int64(3), row["id"])
	require.Equal(t, "test3", row["name"])
}
