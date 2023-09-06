package cmds

import (
	"context"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/middlewares"
	"github.com/jmoiron/sqlx"
)

// NOTE(manuel, 2023-09-01) we need the following features
//
// - generate constructor from flags
// - dbConnectionFactory
// - queries and subqueries
//
// The problem is that we kind of have to parse parameters into the hashmap, to then reinterpret them
// back into the standard Run.
// So we can either populate the parsedLayers needed to call the factory, but that's kind of meh,
// or we can require a *sqlx.DB to be passed in directly.
//
// We can then call RunQueryIntoGlaze(ctx, db, ps, gp).
//
// Do we want to specify a list of return columns and make a struct for that too?
// Maybe add an output schema part?
// Then we can have it return either a list of structs, or a function that writes to a channel of structs
//
// How could we expose something similar form a YAML file? Really just load the whole thing and return a straight SqlCommand?
// But for that we can just a loader. Really the codegen is about defined input and output structured types.

// NOTE(manuel, 2023-09-01) This is a manually written generated code out of a yaml file
// This will be transformed to a proper codegen later on.

const psCommandQuery = `
  SELECT 
  Id,User,Host,db,Command,Time,State
  {{ if .short_info -}}
  ,LEFT(info,50) AS info
  {{ end -}}
  {{ if .medium_info -}}
  ,LEFT(info,80) AS info
  {{ end -}}
  {{ if .full_info -}}
  ,info
  {{ end -}}
   FROM information_schema.processlist
  WHERE 1=1
  {{ if .user_like -}}
  AND User LIKE {{ .user_like | sqlLike }}
  {{ end -}}
  {{ if .mysql_user -}}
  AND User IN ({{ .mysql_user | sqlStringIn }})
  {{ end -}}
  {{ if .state -}}
  AND State IN ({{ .state | sqlStringIn }})
  {{ end -}}
  {{ if .db -}}
  AND db = {{ .db | sqlString }}
  {{ end -}}
  {{ if .db_like -}}
  AND db LIKE {{ .db_like | sqlLike }}
  {{ end -}}
  {{ if .hostname -}}
  AND host = {{ .hostname | sqlString }}
  {{ end -}}
  {{ if .info_like -}}
  AND info LIKE {{ .info_like | sqlLike }}
  {{ end -}}
`

type PsCommand struct {
	*SqlCommand
}

type PsCommandParameters struct {
	MysqlUser  []string `glazed.parameter:"mysql_user"`
	UserLike   string   `glazed.parameter:"user_like"`
	Hostname   string   `glazed.parameter:"hostname"`
	Db         string   `glazed.parameter:"db"`
	DbLike     string   `glazed.parameter:"db_like"`
	State      []string `glazed.parameter:"state"`
	InfoLike   string   `glazed.parameter:"info_like"`
	ShortInfo  bool     `glazed.parameter:"short_info"`
	MediumInfo bool     `glazed.parameter:"medium_info"`
	FullInfo   bool     `glazed.parameter:"full_info"`
}

func (p *PsCommand) Run(ctx context.Context, db *sqlx.DB, params *PsCommandParameters, gp middlewares.Processor) error {
	ps, err := parameters.StructToMap(params)
	if err != nil {
		return err
	}

	err = p.SqlCommand.RunQueryIntoGlaze(ctx, db, ps, gp)
	if err != nil {
		return err
	}

	return nil
}

func NewPSCommand() (*PsCommand, error) {

	desc := cmds.NewCommandDescription(
		"ps",
		cmds.WithFlags(
			parameters.NewParameterDefinition(
				"mysql_user",
				parameters.ParameterTypeStringList,
				parameters.WithHelp("Filter by user(s)"),
			),
		),
	)

	psSqlCommand, err := NewSqlCommand(
		desc,
		WithQuery(psCommandQuery),
	)

	if err != nil {
		return nil, err
	}

	return &PsCommand{
		SqlCommand: psSqlCommand,
	}, nil
}
