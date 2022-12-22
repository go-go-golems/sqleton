package pkg

import (
	"context"
	"github.com/jmoiron/sqlx"
	"github.com/wesen/glazed/pkg/cli"
)

type CommandAlias struct {
	AliasedCommand   SqletonCommand
	Name             string                 `yaml:"name"`
	AliasFor         string                 `yaml:"aliasFor"`
	FlagDefaults     map[string]interface{} `yaml:"flagDefaults"`
	ArgumentDefaults map[string]interface{} `yaml:"argumentDefaults"`

	Parents []string
	Source  string
}

func (a *CommandAlias) IsValid() bool {
	return a.Name != "" && a.AliasFor != ""
}

func (a *CommandAlias) RunQueryIntoGlaze(ctx context.Context, db *sqlx.DB, parameters map[string]interface{}, gp *cli.GlazeProcessor) error {
	return a.AliasedCommand.RunQueryIntoGlaze(ctx, db, parameters, gp)
}

func (a *CommandAlias) RenderQuery(parameters map[string]interface{}) (string, error) {
	return a.AliasedCommand.RenderQuery(parameters)
}

// TODO(2022-12-22, manuel) this is actually not enough, because we want aliases to also deal with
// any kind of default value. So this is a good first approach, but not sufficient...
func (a *CommandAlias) Description() SqletonCommandDescription {
	s := a.AliasedCommand.Description()
	ret := SqletonCommandDescription{
		Name:      a.Name,
		Short:     s.Short,
		Long:      s.Long,
		Flags:     []*SqlParameter{},
		Arguments: []*SqlParameter{},
	}

	for _, flag := range s.Flags {
		newFlag := flag.Copy()
		if defaultValue, ok := a.FlagDefaults[flag.Name]; ok {
			newFlag.Default = defaultValue
		}
		newFlag.Required = false
		ret.Flags = append(ret.Flags, newFlag)
	}

	for _, argument := range s.Arguments {
		newArgument := argument.Copy()
		if defaultValue, ok := a.ArgumentDefaults[argument.Name]; ok {
			newArgument.Default = defaultValue
		}
		newArgument.Required = false
		ret.Arguments = append(ret.Arguments, newArgument)
	}

	return ret
}
