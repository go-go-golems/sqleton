package pkg

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type DefaultTypeTestCase struct {
	Type  ParameterType
	Value interface{}
	Args  []string
}

func TestValidDefaultValue(t *testing.T) {
	testCases := []DefaultTypeTestCase{
		{Type: ParameterTypeString, Value: "foo"},
		{Type: ParameterTypeInteger, Value: 123},
		{Type: ParameterTypeBool, Value: false},
		{Type: ParameterTypeDate, Value: "2018-01-01"},
		{Type: ParameterTypeStringList, Value: []string{"foo", "bar"}},
		{Type: ParameterTypeIntegerList, Value: []int{1, 2, 3}},
		{Type: ParameterTypeStringList, Value: []string{}},
		{Type: ParameterTypeIntegerList, Value: []int{}},
	}
	for _, testCase := range testCases {
		param := &SqlParameter{
			Name:    "foo",
			Default: testCase.Value,
			Type:    testCase.Type,
		}
		err := param.CheckParameterDefaultValueValidity()
		assert.Nil(t, err)
	}
}

func TestValidChoiceDefaultValue(t *testing.T) {
	param := &SqlParameter{
		Name:    "foo",
		Default: "bar",
		Type:    ParameterTypeChoice,
		Choices: []string{"foo", "bar"},
	}
	err := param.CheckParameterDefaultValueValidity()
	assert.Nil(t, err)
}

func TestInvalidChoiceDefaultValue(t *testing.T) {
	testCases := []interface{}{
		"baz",
		123,
		"flop",
	}
	for _, testCase := range testCases {
		param := &SqlParameter{
			Name:    "foo",
			Default: testCase,
			Type:    ParameterTypeChoice,
			Choices: []string{"foo", "bar"},
		}
		err := param.CheckParameterDefaultValueValidity()
		assert.Error(t, err)
	}
}
