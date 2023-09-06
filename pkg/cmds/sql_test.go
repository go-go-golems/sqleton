package cmds

import (
	"context"
	"github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	assert2 "github.com/go-go-golems/glazed/pkg/helpers/assert"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/go-go-golems/glazed/pkg/middlewares/table"
	"github.com/go-go-golems/glazed/pkg/types"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
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

	query, err := s.RenderQuery(context.Background(), nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM test", query)
}

func TestSimpleTemplateRender(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery("SELECT * FROM {{.table}}"),
	)
	require.NoError(t, err)

	query, err := s.RenderQuery(context.Background(),
		map[string]interface{}{
			"table": "test",
		}, nil)
	require.NoError(t, err)
	assert.Equal(t, "SELECT * FROM test", query)
}

func TestSimpleRun(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery("SELECT * FROM test"),
	)
	require.NoError(t, err)

	gp := middlewares.NewTableProcessor()
	require.NoError(t, err)
	gp.AddTableMiddleware(&table.NullTableMiddleware{})
	ctx := context.Background()
	err = s.Run(ctx, map[string]*layers.ParsedParameterLayer{}, map[string]interface{}{}, gp)
	require.NoError(t, err)

	err = gp.Close(ctx)
	require.NoError(t, err)
	table_ := gp.GetTable()
	require.NoError(t, err)

	expected := []types.Row{
		types.NewRow(
			types.MRP("id", int64(1)),
			types.MRP("name", "test1"),
		),
		types.NewRow(
			types.MRP("id", int64(2)),
			types.MRP("name", "test2"),
		),
		types.NewRow(
			types.MRP("id", int64(3)),
			types.MRP("name", "test3"),
		),
	}

	assert2.EqualRows(t, expected, table_.Rows)
}

func TestSimpleSubQuery(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery(`
	SELECT * FROM test
	WHERE id IN (
	    {{ sqlColumn "SELECT test_id FROM test2 WHERE name = {{.name | sqlString }}" | sqlIntIn }}
	)
`,
		),
	)
	require.NoError(t, err)
	db, err := createDB(nil)
	require.NoError(t, err)
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	ps := map[string]interface{}{
		"name": "test2_3",
	}

	ctx := context.Background()
	s_, err := s.RenderQuery(ctx, ps, db)
	require.NoError(t, err)
	assert.Equal(t, sql.CleanQuery(`
	SELECT * FROM test
	WHERE id IN (
	    2
	)
`), s_)

	gp := middlewares.NewTableProcessor()
	gp.AddTableMiddleware(&table.NullTableMiddleware{})
	err = s.Run(ctx, map[string]*layers.ParsedParameterLayer{}, ps, gp)
	require.NoError(t, err)

	err = gp.Close(ctx)
	require.NoError(t, err)
	table_ := gp.GetTable()
	require.NoError(t, err)

	assert2.EqualRows(t, []types.Row{
		types.NewRow(types.MRP("id", int64(2)), types.MRP("name", "test2")),
	}, table_.Rows)
}

func TestSimpleSubQuerySingle(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery(`
	SELECT * FROM test
	WHERE id = {{ sqlSingle "SELECT test_id FROM test2 WHERE name = {{.name | sqlString }} LIMIT 1" }}
`,
		),
	)
	require.NoError(t, err)
	db, err := createDB(nil)
	require.NoError(t, err)
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	ps := map[string]interface{}{
		"name": "test1_1",
	}

	s_, err := s.RenderQuery(context.Background(), ps, db)
	require.NoError(t, err)
	assert.Equal(t, sql.CleanQuery(`
	SELECT * FROM test
	WHERE id = 1
`), s_)

	// fail if we return more than 1
	s, err = NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery(`
	SELECT * FROM test
	WHERE id = {{ sqlSingle "SELECT test_id FROM test2" | sqlIntIn }}
`,
		),
	)
	require.NoError(t, err)

	_, err = s.RenderQuery(context.Background(), ps, db)
	assert.Error(t, err)

	// fail if there are more than 2 fields
	s, err = NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery(`
	SELECT * FROM test
	WHERE id = {{ sqlSingle "SELECT test_id, name FROM test2 WHERE name = {{.name | sqlString }} LIMIT 1" }}
`,
		),
	)
	require.NoError(t, err)

	_, err = s.RenderQuery(context.Background(), ps, db)
	assert.Error(t, err)
}

func TestSimpleSubQueryWithArguments(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery(`
	SELECT * FROM test
	WHERE id IN (
	    {{ sqlColumn "SELECT test_id FROM test2 WHERE name = {{.name | sqlString }} AND id = {{.test2_id}}" "test2_id" 2 | sqlIntIn }}
	)
`,
		),
	)
	require.NoError(t, err)
	db, err := createDB(nil)
	require.NoError(t, err)
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	ps := map[string]interface{}{
		"name": "test1_2",
	}

	ctx := context.Background()
	s_, err := s.RenderQuery(ctx, ps, db)
	require.NoError(t, err)
	assert.Equal(t, sql.CleanQuery(`
	SELECT * FROM test
	WHERE id IN (
	    1
	)
`), s_)

	gp := middlewares.NewTableProcessor()
	gp.AddTableMiddleware(&table.NullTableMiddleware{})
	err = s.Run(ctx, map[string]*layers.ParsedParameterLayer{}, ps, gp)
	require.NoError(t, err)

	err = gp.Close(ctx)
	require.NoError(t, err)
	table_ := gp.GetTable()
	require.NoError(t, err)

	assert2.EqualRows(t, []types.Row{
		types.NewRow(
			types.MRP("id", int64(1)),
			types.MRP("name", "test1"),
		),
	}, table_.Rows)

	s, err = NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery(`
	SELECT * FROM test
	WHERE id IN (
	    {{ sqlColumn "SELECT test_id, id FROM test2 WHERE name = {{.name | sqlString }} AND id = {{.test2_id}}" "test2_id" 2 | sqlIntIn }}
	)
`,
		),
	)
	require.NoError(t, err)

	_, err = s.RenderQuery(ctx, ps, db)
	require.Error(t, err)
}

func TestSliceSubQueryWithArguments(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery(`
	SELECT * FROM test
	WHERE id IN (
	    {{ range sqlSlice "SELECT id, test_id FROM test2 ORDER BY id" }}{{- index . 1 -}} +{{ end }}0
	)
`,
		),
	)
	require.NoError(t, err)
	db, err := createDB(nil)
	require.NoError(t, err)
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	ps := map[string]interface{}{
		"name": "test1_2",
	}

	s_, err := s.RenderQuery(context.Background(), ps, db)
	require.NoError(t, err)
	assert.Equal(t, sql.CleanQuery(`
	SELECT * FROM test
	WHERE id IN (
	    1+1+2+0
	)
`), s_)

}
func TestMapSubQueryWithArguments(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery(`
	SELECT * FROM test
	WHERE id IN (
	    {{ range sqlMap "SELECT id, test_id FROM test2 ORDER BY id" }}{{- index . "id" -}} +{{ end }}0
	)
`,
		),
	)
	require.NoError(t, err)
	db, err := createDB(nil)
	require.NoError(t, err)
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	ps := map[string]interface{}{
		"name": "test1_2",
	}

	s_, err := s.RenderQuery(context.Background(), ps, db)
	require.NoError(t, err)
	assert.Equal(t, sql.CleanQuery(`
	SELECT * FROM test
	WHERE id IN (
	    1+2+3+0
	)
`), s_)

}

func TestMapSubQuery(t *testing.T) {
	s, err := NewSqlCommand(
		cmds.NewCommandDescription("test"),
		WithDbConnectionFactory(createDB),
		WithQuery(`
	SELECT * FROM test
	WHERE id IN (
	    {{ sqlColumn (subQuery "test2_id") "test2_id" 2 | sqlIntIn }}
	)
`,
		),
		WithSubQueries(map[string]string{
			"test2_id": "SELECT test_id FROM test2 WHERE name = {{.name | sqlString }} AND id = {{.test2_id}}",
		}),
	)
	require.NoError(t, err)
	db, err := createDB(nil)
	require.NoError(t, err)
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)

	ps := map[string]interface{}{
		"name": "test1_2",
	}

	ctx := context.Background()
	s_, err := s.RenderQuery(ctx, ps, db)
	require.NoError(t, err)
	assert.Equal(t, sql.CleanQuery(`
	SELECT * FROM test
	WHERE id IN (
	    1
	)
`), s_)

	gp := middlewares.NewTableProcessor()
	gp.AddTableMiddleware(&table.NullTableMiddleware{})
	err = s.Run(ctx, map[string]*layers.ParsedParameterLayer{}, ps, gp)
	require.NoError(t, err)

	err = gp.Close(ctx)
	require.NoError(t, err)
	table_ := gp.GetTable()
	require.NoError(t, err)
	rows := table_.Rows
	assert.Equal(t, 1, len(rows))
	row := rows[0]
	id, ok := row.Get("id")
	assert.True(t, ok)
	assert.Equal(t, int64(1), id)
	name, ok := row.Get("name")
	assert.True(t, ok)
	assert.Equal(t, "test1", name)

}
