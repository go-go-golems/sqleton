package pkg

import (
	_ "embed"
	"github.com/go-go-golems/glazed/pkg/cmds/layers"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

//go:embed "flags/connection.yaml"
var connectionFlagsYaml []byte

type ConnectionParameterLayer struct {
	layers.ParameterLayerImpl
}

func NewSqlConnectionParameterLayer() (*ConnectionParameterLayer, error) {
	ret := &ConnectionParameterLayer{}
	err := ret.LoadFromYAML(connectionFlagsYaml)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize connection parameter layer")
	}
	return ret, nil
}

func (cp *ConnectionParameterLayer) ParseFlagsFromCobraCommand(_ *cobra.Command) (map[string]interface{}, error) {
	// actually hijack and load everything from viper instead of cobra...
	ps := make(map[string]interface{})

	for _, f := range cp.Flags {
		switch f.Type {
		case parameters.ParameterTypeString:
			v := viper.GetString(f.Name)
			ps[f.Name] = v
		case parameters.ParameterTypeInteger:
			v := viper.GetInt(f.Name)
			ps[f.Name] = v
		default:
			return nil, errors.Errorf("Unknown DB Connection parameter type %s for flag: %s", f.Type, f.Name)
		}
	}

	return ps, nil
}

//go:embed "flags/helpers.yaml"
var helpersFlagsYaml []byte

func NewSqlHelpersParameterLayer() (*layers.ParameterLayerImpl, error) {
	ret := &layers.ParameterLayerImpl{}
	err := ret.LoadFromYAML(helpersFlagsYaml)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize helpers parameter layer")
	}
	return ret, nil
}

//go:embed "flags/dbt.yaml"
var dbtFlagsYaml []byte

type DbtParameterLayer struct {
	layers.ParameterLayerImpl
}

func NewDbtParameterLayer() (*DbtParameterLayer, error) {
	ret := &DbtParameterLayer{}
	err := ret.LoadFromYAML(dbtFlagsYaml)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize dbt parameter layer")
	}
	return ret, nil
}

func (d *DbtParameterLayer) ParseFlagsFromCobraCommand(_ *cobra.Command) (map[string]interface{}, error) {
	// actually hijack and load everything from viper instead of cobra...
	ps := make(map[string]interface{})

	for _, f := range d.Flags {
		switch f.Type {
		case parameters.ParameterTypeString:
			v := viper.GetString(f.Name)
			ps[f.Name] = v
		case parameters.ParameterTypeInteger:
			v := viper.GetInt(f.Name)
			ps[f.Name] = v
		case parameters.ParameterTypeBool:
			v := viper.GetBool(f.Name)
			ps[f.Name] = v
		default:
			return nil, errors.Errorf("Unknown DBT parameter type %s for flag %s", f.Type, f.Name)
		}
	}

	return ps, nil
}
