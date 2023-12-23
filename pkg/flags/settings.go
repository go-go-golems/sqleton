package flags

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/pkg/errors"
)

//go:embed "helpers.yaml"
var helpersFlagsYaml []byte

const SqlHelpersSlug = "sql-helpers"

type SqlHelpersSettings struct {
	Explain    bool `glazed.parameter:"explain"`
	PrintQuery bool `glazed.parameter:"print-query"`
}

func NewSqlHelpersParameterLayer(
	options ...layers.ParameterLayerOptions,
) (*layers.ParameterLayerImpl, error) {
	ret, err := layers.NewParameterLayerFromYAML(helpersFlagsYaml, options...)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize helpers parameter layer")
	}
	return ret, nil
}
