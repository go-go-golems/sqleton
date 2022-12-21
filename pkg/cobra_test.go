package pkg

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

// TODO(2022-12-21, manuel) Things to test:
// - default values for optional arguments
// - addFlags
// - invalid default values
// - parsing dates

func TestAddZeroArguments(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{},
	}
	err := addArguments(cmd, &desc)
	// assert that err is nil
	require.Nil(t, err)
}

func TestAddSingleRequiredArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
				Type:     ParameterTypeString,
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"bar"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"bar", "foo"}))

	values, err := gatherArguments(desc.Arguments, []string{"bar"})
	require.Nil(t, err)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, "bar", values["foo"])

	_, err = gatherArguments(desc.Arguments, []string{})
	assert.Error(t, err)

	_, err = gatherArguments(desc.Arguments, []string{"foo", "bla"})
	assert.Error(t, err)
}

func TestAddTwoRequiredArguments(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
				Type:     ParameterTypeString,
			},
			{
				Name:     "bar",
				Required: true,
				Type:     ParameterTypeString,
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"bar"}))
	assert.Error(t, cmd.Args(cmd, []string{"bar", "foo", "baz"}))

	values, err := gatherArguments(desc.Arguments, []string{"bar", "foo"})
	require.Nil(t, err)
	assert.Equal(t, 2, len(values))
	assert.Equal(t, "bar", values["foo"])
	assert.Equal(t, "foo", values["bar"])

	_, err = gatherArguments(desc.Arguments, []string{})
	assert.Error(t, err)

	_, err = gatherArguments(desc.Arguments, []string{"bar"})
	assert.Error(t, err)

	_, err = gatherArguments(desc.Arguments, []string{"bar", "foo", "baz"})
	assert.Error(t, err)
}

func TestOneRequiredOneOptionalArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
				Type:     ParameterTypeString,
			},
			{
				Name:    "bar",
				Type:    ParameterTypeString,
				Default: "baz",
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"bar", "foo", "baz"}))

	values, err := gatherArguments(desc.Arguments, []string{"bar", "foo"})
	require.Nil(t, err)
	assert.Equal(t, 2, len(values))
	assert.Equal(t, "bar", values["foo"])
	assert.Equal(t, "foo", values["bar"])

	values, err = gatherArguments(desc.Arguments, []string{"foo"})
	require.Nil(t, err)
	assert.Equal(t, 2, len(values))
	assert.Equal(t, "foo", values["foo"])
	assert.Equal(t, "baz", values["bar"])

	_, err = gatherArguments(desc.Arguments, []string{})
	assert.Error(t, err)

	_, err = gatherArguments(desc.Arguments, []string{"bar", "foo", "baz"})
	assert.Error(t, err)
}

func TestOneOptionalArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:    "foo",
				Default: "123",
				Type:    ParameterTypeString,
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Error(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{}))

	values, err := gatherArguments(desc.Arguments, []string{"foo"})
	require.Nil(t, err)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, "foo", values["foo"])

	values, err = gatherArguments(desc.Arguments, []string{})
	require.Nil(t, err)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, "123", values["foo"])
}

func TestDefaultIntValue(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:    "foo",
				Default: 123,
				Type:    ParameterTypeInteger,
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	values, err := gatherArguments(desc.Arguments, []string{})
	require.Nil(t, err)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, 123, values["foo"])

	values, err = gatherArguments(desc.Arguments, []string{"234"})
	require.Nil(t, err)
	assert.Equal(t, 1, len(values))
	assert.Equal(t, 234, values["foo"])

	_, err = gatherArguments(desc.Arguments, []string{"foo"})
	assert.Error(t, err)
}

func TestParseDate(t *testing.T) {
	// set default time for unit tests
	refTime_ := time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)
	refTime = &refTime_

	testCases := []struct {
		Value  string
		Result time.Time
	}{
		{Value: "2018-01-01", Result: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Value: "2018/01/01", Result: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
		//{Value: "January First 2018", Result: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Value: "January 1st 2018", Result: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Value: "2018-01-01T00:00:00+00:00", Result: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Value: "2018-01-01T00:00:00+01:00", Result: time.Date(2018, 1, 1, 0, 0, 0, 0, time.FixedZone("", 3600))},
		{Value: "2018-01-01T00:00:00-01:00", Result: time.Date(2018, 1, 1, 0, 0, 0, 0, time.FixedZone("", -3600))},
		{Value: "2018", Result: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Value: "2018-01", Result: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Value: "last year", Result: time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Value: "last hour", Result: time.Date(2017, 12, 31, 23, 0, 0, 0, time.UTC)},
		{Value: "last month", Result: time.Date(2017, 12, 1, 0, 0, 0, 0, time.UTC)},
		{Value: "last week", Result: time.Date(2017, 12, 25, 0, 0, 0, 0, time.UTC)},
		{Value: "last monday", Result: time.Date(2017, 12, 25, 0, 0, 0, 0, time.UTC)},
		{Value: "10 days ago", Result: time.Date(2017, 12, 22, 0, 0, 0, 0, time.UTC)},
	}

	for _, testCase := range testCases {
		result, err := parseDate(testCase.Value)
		require.Nil(t, err)
		if !result.Equal(testCase.Result) {
			t.Errorf("Expected %s to parse to %s, got %s", testCase.Value, testCase.Result, result)
		}
	}
}

func TestInvalidDefaultValue(t *testing.T) {
	cmd := &cobra.Command{}
	failingTypes := []DefaultTypeTestCase{
		{Type: ParameterTypeString, Value: 123},
		{Type: ParameterTypeString, Value: []string{"foo"}},
		{Type: ParameterTypeInteger, Value: "foo"},
		{Type: ParameterTypeInteger, Value: []int{1}},
		// so oddly enough this is a valid word
		{Type: ParameterTypeDate, Value: "22#@!"},
		{Type: ParameterTypeStringList, Value: "foo"},
		{Type: ParameterTypeIntegerList, Value: "foo"},
		{Type: ParameterTypeStringList, Value: []int{1, 2, 3}},
		{Type: ParameterTypeStringList, Value: []int{}},
		{Type: ParameterTypeIntegerList, Value: []string{"1", "2", "3"}},
		{Type: ParameterTypeIntegerList, Value: []string{}},
	}
	for _, failingType := range failingTypes {
		desc := SqletonCommandDescription{
			Arguments: []*SqlParameter{
				{
					Name:    "foo",
					Default: failingType.Value,
					Type:    failingType.Type,
				},
			},
		}
		err := addArguments(cmd, &desc)
		if err == nil {
			t.Errorf("Expected error for type %s and value %v\n", failingType.Type, failingType.Value)
		}
		assert.Error(t, err)
	}
}

func TestTwoOptionalArguments(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name: "foo",
			},
			{
				Name: "bar",
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Error(t, cmd.Args(cmd, []string{"bar", "foo", "blop"}))
	assert.Nil(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{}))
}

func TestFailAddingRequiredAfterOptional(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name: "foo",
			},
			{
				Name:     "bar",
				Required: true,
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Error(t, err)
}

func TestAddStringListRequiredArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
				Type:     ParameterTypeStringList,
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Nil(t, cmd.Args(cmd, []string{"bar"}))
	assert.Nil(t, cmd.Args(cmd, []string{"bar", "foo", "baz"}))
}

func TestAddStringListOptionalArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:    "foo",
				Type:    ParameterTypeStringList,
				Default: []string{"baz"},
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{}))

	values, err := gatherArguments(desc.Arguments, []string{"bar", "foo"})
	require.Nil(t, err)
	assert.Equal(t, []string{"bar", "foo"}, values["foo"])

	values, err = gatherArguments(desc.Arguments, []string{"foo"})
	require.Nil(t, err)
	assert.Equal(t, []string{"foo"}, values["foo"])

	values, err = gatherArguments(desc.Arguments, []string{})
	require.Nil(t, err)
	assert.Equal(t, []string{"baz"}, values["foo"])
}

func TestFailAddingArgumentAfterStringList(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name: "foo",
				Type: ParameterTypeStringList,
			},
			{
				Name: "bar",
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Error(t, err)
}

func TestAddIntegerListRequiredArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
				Type:     ParameterTypeIntegerList,
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"1", "2"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Nil(t, cmd.Args(cmd, []string{"1"}))
	assert.Nil(t, cmd.Args(cmd, []string{"1", "4", "2"}))
}

func TestAddStringListRequiredAfterRequiredArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
			},
			{
				Name:     "bar",
				Type:     ParameterTypeStringList,
				Required: true,
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"foo", "bar"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"1"}))
	assert.Nil(t, cmd.Args(cmd, []string{"1", "4", "2"}))
}

func TestAddStringListOptionalAfterRequiredArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
			},
			{
				Name:    "bar",
				Type:    ParameterTypeStringList,
				Default: []string{"blop"},
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"foo", "bar", "baz"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo", "bar"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
}

func TestAddStringListOptionalAfterOptionalArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:    "foo",
				Type:    ParameterTypeString,
				Default: "blop",
			},
			{
				Name:    "bar",
				Type:    ParameterTypeStringList,
				Default: []string{"bloppp"},
			},
		},
	}
	err := addArguments(cmd, &desc)
	require.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"foo", "bar", "baz"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo", "bar"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{}))
}

func TestAddStringListRequiredAfterOptionalArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name: "foo",
			},
			{
				Name:     "bar",
				Type:     ParameterTypeStringList,
				Required: true,
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Error(t, err)
}
