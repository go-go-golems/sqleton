package codegen

import (
	"github.com/dave/jennifer/jen"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/iancoleman/strcase"
)

type SqlCommandCodeGenerator struct {
	PackageName string
}

const SqletonCmdsPath = "github.com/go-go-golems/sqleton/pkg/cmds"
const GlazedCommandsPath = "github.com/go-go-golems/glazed/pkg/cmds"
const GlazedMiddlewaresPath = "github.com/go-go-golems/glazed/pkg/middlewares"
const GlazedParametersPath = "github.com/go-go-golems/glazed/pkg/cmds/parameters"
const ClaySqlPath = "github.com/go-go-golems/clay/pkg/sql"
const MapsHelpersPath = "github.com/go-go-golems/glazed/pkg/helpers/maps"

func (s *SqlCommandCodeGenerator) defineRunMethod(f *jen.File, cmdName string) {
	methodName := "Run"
	receiver := strcase.ToCamel(cmdName) + "Command"
	parametersStruct := strcase.ToCamel(cmdName) + "CommandParameters"

	f.Func().Params(jen.Id("p").Op("*").Id(receiver)).Id(methodName).
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("db").Op("*").Qual("github.com/jmoiron/sqlx", "DB"),
			jen.Id("params").Op("*").Id(parametersStruct),
			jen.Id("gp").Qual(GlazedMiddlewaresPath, "Processor"),
		).Error().
		Block(
			jen.Id("ps").Op(":=").Qual(MapsHelpersPath, "StructToMap").Call(jen.Id("params"), jen.Lit(false)),
			jen.List(jen.Id("renderedQuery"), jen.Err()).Op(":=").Qual(ClaySqlPath, "RenderQuery").Call(
				jen.Id("ctx"), jen.Id("db"), jen.Id("p").Dot("Query"), jen.Id("p").Dot("SubQueries"), jen.Id("ps"),
			),
			jen.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Err())),
			jen.Err().Op("=").Qual(ClaySqlPath, "RunQueryIntoGlaze").Call(
				jen.Id("ctx"), jen.Id("db"), jen.Id("renderedQuery"), jen.Index().Interface().Values(), jen.Id("gp"),
			),
			jen.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Err())),
			jen.Return(jen.Nil()),
		)
}

func (s *SqlCommandCodeGenerator) defineNewFunction(f *jen.File, cmdName string, cmd *cmds.SqlCommand) error {
	funcName := "New" + strcase.ToCamel(cmdName) + "Command"
	commandStruct := strcase.ToCamel(cmdName) + "Command"
	queryConstName := strcase.ToCamel(cmdName) + "CommandQuery"

	description := cmd.Description()

	var dicts []jen.Code
	for _, flag := range cmd.Flags {
		dict, err := ParameterDefinitionToDict(flag)
		if err != nil {
			return err
		}
		dicts = append(dicts, dict)
	}

	var err_ error
	f.Func().Id(funcName).Params().
		Params(jen.Op("*").Id(commandStruct), jen.Error()).
		Block(
			jen.Var().Id("flagDefs").Op("=").
				Index().Op("*").
				Qual(GlazedParametersPath, "ParameterDefinition").
				ValuesFunc(func(g *jen.Group) {
					for _, flag := range cmd.Flags {
						dict, err := ParameterDefinitionToDict(flag)
						if err != nil {
							err_ = err
							return
						}
						g.Values(dict)
					}
				}),
			jen.Id("cmdDescription").Op(":=").Qual(GlazedCommandsPath, "NewCommandDescription").
				Call(
					jen.Lit(description.Name),
					jen.Qual(GlazedCommandsPath, "WithShort").Call(jen.Lit(description.Short)),
					jen.Qual(GlazedCommandsPath, "WithLong").Call(jen.Lit(description.Long)),
					jen.Qual(GlazedCommandsPath, "WithFlags").Call(jen.Id("flagDefs").Op("...")),
				),
			jen.Return(jen.Op("&").Id(commandStruct).Values(jen.Dict{
				jen.Id("CommandDescription"): jen.Id("cmdDescription"),
				jen.Id("Query"):              jen.Id(queryConstName),
				jen.Id("SubQueries"):         jen.Map(jen.String()).String().Values(),
			}), jen.Nil()),
		)

	return err_
}

func (s *SqlCommandCodeGenerator) defineConstants(f *jen.File, cmdName string, cmd *cmds.SqlCommand) {
	// Define the constant for the main query.
	queryConstName := strcase.ToCamel(cmdName) + "CommandQuery"
	f.Const().Id(queryConstName).Op("=").Lit(cmd.Query)
	// Define constants for subqueries if they exist.
	if len(cmd.SubQueries) > 0 {
		for name, subQuery := range cmd.SubQueries {
			subQueryConstName := strcase.ToCamel(cmdName) + "CommandSubQuery" + name
			f.Const().Id(subQueryConstName).Op("=").Lit(subQuery)
		}
	}
}

func (s *SqlCommandCodeGenerator) defineStruct(f *jen.File, cmdName string) {
	structName := strcase.ToCamel(cmdName) + "Command"
	f.Type().Id(structName).Struct(
		jen.Op("*").Qual(GlazedCommandsPath, "CommandDescription"),
		jen.Id("Query").String().Tag(map[string]string{"yaml": "query"}),
		jen.Id("SubQueries").Map(jen.String()).String().Tag(map[string]string{"yaml": "subqueries,omitempty"}),
	)
}

func (s *SqlCommandCodeGenerator) defineParametersStruct(f *jen.File, cmdName string, flags []*parameters.ParameterDefinition) {
	structName := strcase.ToCamel(cmdName) + "CommandParameters"
	f.Type().Id(structName).StructFunc(func(g *jen.Group) {
		for _, flag := range flags {
			s := g.Id(strcase.ToCamel(flag.Name))
			s = FlagTypeToGoType(s, flag.Type)
			s.Tag(map[string]string{"glazed.parameter": strcase.ToSnake(flag.Name)})
		}
	})
}

func (s *SqlCommandCodeGenerator) GenerateCommandCode(cmd *cmds.SqlCommand) (*jen.File, error) {
	f := jen.NewFile(s.PackageName)
	cmdName := strcase.ToLowerCamel(cmd.Name)

	// Define constants, struct, and methods using helper functions.
	s.defineConstants(f, cmdName, cmd)
	s.defineStruct(f, cmdName)
	s.defineParametersStruct(f, cmdName, cmd.Flags)
	s.defineRunMethod(f, cmdName)
	err := s.defineNewFunction(f, cmdName, cmd)
	if err != nil {
		return nil, err
	}

	return f, nil
}
