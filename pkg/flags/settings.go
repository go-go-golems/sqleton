package flags

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/schema"
	"github.com/pkg/errors"
)

//go:embed "helpers.yaml"
var helpersFlagsYaml []byte

const SqlHelpersSlug = "sql-helpers"

type SqlHelpersSettings struct {
	Explain    bool `glazed:"explain"`
	PrintQuery bool `glazed:"print-query"`
}

func NewSqlHelpersParameterLayer(
	options ...schema.SectionOption,
) (*schema.SectionImpl, error) {
	ret, err := schema.NewSectionFromYAML(helpersFlagsYaml, options...)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize helpers parameter layer")
	}
	return ret, nil
}
