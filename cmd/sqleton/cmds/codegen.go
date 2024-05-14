package cmds

import (
	"fmt"
	"github.com/go-go-golems/glazed/pkg/cmds"
	"github.com/go-go-golems/glazed/pkg/cmds/alias"
	"github.com/go-go-golems/glazed/pkg/cmds/loaders"
	cmds2 "github.com/go-go-golems/sqleton/pkg/cmds"
	"github.com/go-go-golems/sqleton/pkg/codegen"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
	"path"
	"strings"
)

func NewCodegenCommand() *cobra.Command {
	ret := &cobra.Command{
		Use:   "codegen [file...]",
		Short: "A program to convert Sqleton YAML commands into Go code",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packageName := cmd.Flag("package-name").Value.String()
			outputDir := cmd.Flag("output-dir").Value.String()

			s := &codegen.SqlCommandCodeGenerator{
				PackageName: packageName,
			}
			for _, fileName := range args {
				loader := &cmds2.SqlCommandLoader{
					DBConnectionFactory: nil,
				}

				fs_, fileName, err := loaders.FileNameToFsFilePath(fileName)
				if err != nil {
					return err
				}
				cmds_, err := loader.LoadCommands(fs_, fileName, []cmds.CommandDescriptionOption{}, []alias.Option{})
				if err != nil {
					return err
				}
				if len(cmds_) != 1 {
					return errors.Errorf("expected exactly one command, got %d", len(cmds_))
				}
				cmd := cmds_[0].(*cmds2.SqlCommand)

				f, err := s.GenerateCommandCode(cmd)
				if err != nil {
					return err
				}

				s := f.GoString()
				// store in path.go after removing .yaml
				p, _ := strings.CutSuffix(path.Base(fileName), ".yaml")
				p = p + ".go"
				p = path.Join(outputDir, p)

				fmt.Printf("Converting %s to %s\n", fileName, p)
				_ = os.WriteFile(p, []byte(s), 0644)
			}

			return nil
		},
	}

	ret.PersistentFlags().StringP("output-dir", "o", ".", "Output directory for generated code")
	ret.PersistentFlags().StringP("package-name", "p", "main", "Package name for generated code")
	return ret
}
