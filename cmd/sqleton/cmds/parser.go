package cmds

import (
	"github.com/go-go-golems/glazed/pkg/cli"
	sqleton_cmds "github.com/go-go-golems/sqleton/pkg/cmds"
)

const SqletonAppName = "sqleton"

func NewSqletonParserConfig() cli.CobraParserConfig {
	return cli.CobraParserConfig{
		MiddlewaresFunc: sqleton_cmds.GetCobraCommandSqletonMiddlewares,
	}
}
