package pkg

import (
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAddZeroArguments(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{},
	}
	err := addArguments(cmd, &desc)
	// assert that err is nil
	assert.Nil(t, err)
}

func TestAddSingleRequiredArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"bar"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"bar", "foo"}))

	// test gather
	_, _ = gatherArguments(cmd, &desc, []string{"bar"})
}

func TestAddTwoRequiredArguments(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
			},
			{
				Name:     "bar",
				Required: true,
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"bar"}))
	assert.Error(t, cmd.Args(cmd, []string{"bar", "foo", "baz"}))
}

func TestOneRequiredOneOptionalArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name:     "foo",
				Required: true,
			},
			{
				Name: "bar",
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo"}))
	assert.Error(t, cmd.Args(cmd, []string{}))
	assert.Error(t, cmd.Args(cmd, []string{"bar", "foo", "baz"}))
}

func TestOneOptionalArgument(t *testing.T) {
	cmd := &cobra.Command{}
	desc := SqletonCommandDescription{
		Arguments: []*SqlParameter{
			{
				Name: "foo",
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Nil(t, err)
	assert.Error(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{}))
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
	assert.Nil(t, err)
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
	assert.Nil(t, err)
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
				Name: "foo",
				Type: ParameterTypeStringList,
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Nil(t, err)
	assert.Nil(t, cmd.Args(cmd, []string{"bar", "foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{"foo"}))
	assert.Nil(t, cmd.Args(cmd, []string{}))
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
	assert.Nil(t, err)
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
	assert.Nil(t, err)
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
				Name: "bar",
				Type: ParameterTypeStringList,
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Nil(t, err)
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
				Name: "foo",
			},
			{
				Name: "bar",
				Type: ParameterTypeStringList,
			},
		},
	}
	err := addArguments(cmd, &desc)
	assert.Nil(t, err)
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
