package codegen

import (
	"github.com/dave/jennifer/jen"
	cmds2 "github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"github.com/go-go-golems/glazed/pkg/codegen"
	"github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/iancoleman/strcase"
)

type SqlCommandCodeGenerator struct {
	PackageName string
}

const SqletonCmdsPath = "github.com/go-go-golems/sqleton/pkg/cmds"

func (s *SqlCommandCodeGenerator) defineConstants(f *jen.File, cmdName string, cmd *cmds.SqlCommand) {
	// Define the constant for the main query.
	queryConstName := strcase.ToLowerCamel(cmdName) + "CommandQuery"
	f.Const().Id(queryConstName).Op("=").Lit(cmd.Query)

	if len(cmd.SubQueries) > 0 {
		for name, subQuery := range cmd.SubQueries {
			subQueryConstName := strcase.ToLowerCamel(cmdName) + "CommandSubQuery" + name
			f.Const().Id(subQueryConstName).Op("=").Lit(subQuery)
		}
	}
}

func (s *SqlCommandCodeGenerator) defineStruct(f *jen.File, cmdName string) {
	structName := strcase.ToCamel(cmdName) + "Command"
	f.Type().Id(structName).Struct(
		jen.Op("*").Qual(codegen.GlazedCommandsPath, "CommandDescription"),
		jen.Id("Query").String().Tag(map[string]string{"yaml": "query"}),
		jen.Id("SubQueries").Map(jen.String()).String().Tag(map[string]string{"yaml": "subqueries,omitempty"}),
	)
}

func (s *SqlCommandCodeGenerator) defineParametersStruct(
	f *jen.File,
	cmdName string,
	cmd *cmds2.CommandDescription,
) {
	structName := strcase.ToCamel(cmdName) + "CommandParameters"
	f.Type().Id(structName).StructFunc(func(g *jen.Group) {
		cmd.GetDefaultFlags().ForEach(func(flag *parameters.ParameterDefinition) {
			s := g.Id(strcase.ToCamel(flag.Name))
			s = codegen.FlagTypeToGoType(s, flag.Type)
			s.Tag(map[string]string{"glazed.parameter": strcase.ToSnake(flag.Name)})
		})
		cmd.GetDefaultArguments().ForEach(func(arg *parameters.ParameterDefinition) {
			s := g.Id(strcase.ToCamel(arg.Name))
			s = codegen.FlagTypeToGoType(s, arg.Type)
			s.Tag(map[string]string{"glazed.parameter": strcase.ToSnake(arg.Name)})
		})
	})
}

func (s *SqlCommandCodeGenerator) renderQuery() []jen.Code {
	return []jen.Code{
		jen.Id("ps").Op(":=").Qual(codegen.MapsHelpersPath, "StructToMap").Call(jen.Id("params"), jen.Lit(false)),
		jen.List(jen.Id("renderedQuery"), jen.Err()).Op(":=").Qual(codegen.ClaySqlPath, "RenderQuery").Call(
			jen.Id("ctx"), jen.Id("db"), jen.Id("p").Dot("Query"), jen.Id("p").Dot("SubQueries"), jen.Id("ps"),
		),
		jen.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Err())),
		jen.Line(),
	}
}

func (s *SqlCommandCodeGenerator) defineRunIntoGlazedMethod(f *jen.File, cmdName string) {
	methodName := "RunIntoGlazed"
	receiver := strcase.ToCamel(cmdName) + "Command"
	parametersStruct := strcase.ToCamel(cmdName) + "CommandParameters"

	f.Func().
		Params(jen.Id("p").Op("*").Id(receiver)).Id(methodName).
		Params(
			jen.Id("ctx").Qual("context", "Context"),
			jen.Id("db").Op("*").Qual("github.com/jmoiron/sqlx", "DB"),
			jen.Id("params").Op("*").Id(parametersStruct),
			jen.Id("gp").Qual(codegen.GlazedMiddlewaresPath, "Processor"),
		).Error().
		BlockFunc(func(g *jen.Group) {
			for _, c := range s.renderQuery() {
				g.Add(c)
			}
			g.Err().Op("=").Qual(codegen.ClaySqlPath, "RunQueryIntoGlaze").Call(
				jen.Id("ctx"), jen.Id("db"), jen.Id("renderedQuery"), jen.Index().Interface().Values(), jen.Id("gp"),
			)
			g.If(jen.Err().Op("!=").Nil()).Block(jen.Return(jen.Err()))
			g.Return(jen.Nil())
		})
}

func (s *SqlCommandCodeGenerator) defineNewFunction(f *jen.File, cmdName string, cmd *cmds.SqlCommand) error {
	funcName := "New" + strcase.ToCamel(cmdName) + "Command"
	commandStruct := strcase.ToCamel(cmdName) + "Command"
	queryConstName := strcase.ToLowerCamel(cmdName) + "CommandQuery"

	description := cmd.Description()

	var err_ error
	f.Func().Id(funcName).Params().
		Params(jen.Op("*").Id(commandStruct), jen.Error()).
		Block(
			// TODO(manuel, 2023-12-07) Can be refactored since this is duplicated in geppetto/codegen.go
			jen.Var().Id("flagDefs").Op("=").
				Index().Op("*").
				Qual(codegen.GlazedParametersPath, "ParameterDefinition").
				ValuesFunc(func(g *jen.Group) {
					err_ = cmd.GetDefaultFlags().ForEachE(func(flag *parameters.ParameterDefinition) error {
						dict, err := codegen.ParameterDefinitionToDict(flag)
						if err != nil {
							return err
						}
						g.Values(dict)
						return nil
					})
				}),
			jen.Line(),
			jen.Var().Id("argDefs").Op("=").
				Index().Op("*").
				Qual(codegen.GlazedParametersPath, "ParameterDefinition").
				ValuesFunc(func(g *jen.Group) {
					err_ = cmd.GetDefaultArguments().ForEachE(func(arg *parameters.ParameterDefinition) error {
						dict, err := codegen.ParameterDefinitionToDict(arg)
						if err != nil {
							return err
						}
						g.Values(dict)
						return nil
					})
				}),
			jen.Line(),
			jen.Id("cmdDescription").Op(":=").Qual(codegen.GlazedCommandsPath, "NewCommandDescription").
				Call(
					jen.Lit(description.Name),
					jen.Line().Qual(codegen.GlazedCommandsPath, "WithShort").Call(jen.Lit(description.Short)),
					jen.Line().Qual(codegen.GlazedCommandsPath, "WithLong").Call(jen.Lit(description.Long)),
					jen.Line().Qual(codegen.GlazedCommandsPath, "WithFlags").Call(jen.Id("flagDefs").Op("...")),
					jen.Line().Qual(codegen.GlazedCommandsPath, "WithArguments").Call(jen.Id("argDefs").Op("...")),
				),
			jen.Line(),

			jen.Return(jen.Op("&").Id(commandStruct).Values(jen.Dict{
				jen.Id("CommandDescription"): jen.Id("cmdDescription"),
				jen.Id("Query"):              jen.Id(queryConstName),
				jen.Id("SubQueries"): jen.Map(jen.String()).String().Values(jen.DictFunc(func(d jen.Dict) {
					if len(cmd.SubQueries) > 0 {
						for name := range cmd.SubQueries {
							subQueryConstName := strcase.ToLowerCamel(cmdName) + "CommandSubQuery" + name
							d[jen.Lit(name)] = jen.Id(subQueryConstName)
						}
					}
				})),
			}), jen.Nil()),
		)

	return err_
}

func (s *SqlCommandCodeGenerator) GenerateCommandCode(cmd *cmds.SqlCommand) (*jen.File, error) {
	f := jen.NewFile(s.PackageName)
	cmdName := strcase.ToLowerCamel(cmd.Name)

	// Define constants, struct, and methods using helper functions.
	s.defineConstants(f, cmdName, cmd)
	s.defineStruct(f, cmdName)
	f.Line()
	s.defineParametersStruct(f, cmdName, cmd.Description())
	s.defineRunIntoGlazedMethod(f, cmdName)
	f.Line()
	err := s.defineNewFunction(f, cmdName, cmd)
	if err != nil {
		return nil, err
	}

	return f, nil
}
