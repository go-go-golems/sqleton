package codegen

import (
	"github.com/dave/jennifer/jen"
	"github.com/go-go-golems/glazed/pkg/cmds/parameters"
	"reflect"
)

func ParameterDefinitionToDict(p *parameters.ParameterDefinition) (jen.Code, error) {
	ret := jen.Dict{
		jen.Id("Name"): jen.Lit(p.Name),
		jen.Id("Type"): jen.Lit(string(p.Type)),
		jen.Id("Help"): jen.Lit(p.Help),
	}

	if p.Default != nil {
		var err error
		ret[jen.Id("Default")], err = FlagValueToJen(p)
		if err != nil {
			return nil, err
		}
	}
	if p.Choices != nil {
		ret[jen.Id("Choices")] = jen.Index().String().ValuesFunc(func(g *jen.Group) {
			for _, c := range p.Choices {
				g.Lit(c)
			}
		})
	}

	return ret, nil
}

func FlagValueToJen(p *parameters.ParameterDefinition) (jen.Code, error) {
	err := p.CheckParameterDefaultValueValidity()
	if err != nil {
		return nil, err
	}

	d := p.Default
	return LiteralToJen(reflect.ValueOf(d))
}

func FlagTypeToGoType(s *jen.Statement, parameterType parameters.ParameterType) *jen.Statement {
	switch parameterType {
	case parameters.ParameterTypeFloat:
		return s.Id("float64")
	case parameters.ParameterTypeFloatList:
		return s.Index().Id("float64")
	case parameters.ParameterTypeInteger:
		return s.Id("int")
	case parameters.ParameterTypeIntegerList:
		return s.Index().Id("int")
	case parameters.ParameterTypeBool:
		return s.Id("bool")
	case parameters.ParameterTypeDate:
		return s.Qual("time", "Time")
	case parameters.ParameterTypeStringFromFile,
		parameters.ParameterTypeStringFromFiles,
		parameters.ParameterTypeChoice,
		parameters.ParameterTypeString:
		return s.Id("string")
	case parameters.ParameterTypeStringList,
		parameters.ParameterTypeStringListFromFile,
		parameters.ParameterTypeStringListFromFiles,
		parameters.ParameterTypeChoiceList:
		return s.Index().Id("string")
	case parameters.ParameterTypeFile:
		return s.Qual(GlazedParametersPath, "FileData")
	case parameters.ParameterTypeFileList:
		return s.Index().Qual(GlazedParametersPath, "FileData")
	case parameters.ParameterTypeObjectFromFile:
		return s.Map(jen.Id("string")).Id("interface{}")
	case parameters.ParameterTypeObjectListFromFile, parameters.ParameterTypeObjectListFromFiles:
		return s.Index().Map(jen.Id("string")).Id("interface{}")
	case parameters.ParameterTypeKeyValue:
		return s.Map(jen.Id("string")).Id("string")
	default:
		return s.Id(string(parameterType))
	}
}
