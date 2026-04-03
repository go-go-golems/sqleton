package cmds

import (
	"bytes"
	"io"
	"strings"

	clay_sql "github.com/go-go-golems/clay/pkg/sql"
	"github.com/go-go-golems/glazed/pkg/cmds"
	fields "github.com/go-go-golems/glazed/pkg/cmds/fields"
	"github.com/go-go-golems/glazed/pkg/cmds/layout"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type SourceKind int

const (
	SourceUnknown SourceKind = iota
	SourceSQLCommand
	SourceYAMLAlias
)

func DetectSourceKind(path string) SourceKind {
	lower := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lower, ".alias.yaml"), strings.HasSuffix(lower, ".alias.yml"):
		return SourceYAMLAlias
	case strings.HasSuffix(lower, ".sql"):
		return SourceSQLCommand
	default:
		return SourceUnknown
	}
}

type SqlCommandSpec struct {
	Name       string                 `yaml:"name"`
	Short      string                 `yaml:"short"`
	Long       string                 `yaml:"long,omitempty"`
	Layout     []*layout.Section      `yaml:"layout,omitempty"`
	Flags      []*fields.Definition   `yaml:"flags,omitempty"`
	Arguments  []*fields.Definition   `yaml:"arguments,omitempty"`
	Tags       []string               `yaml:"tags,omitempty"`
	Metadata   map[string]interface{} `yaml:"metadata,omitempty"`
	Query      string                 `yaml:"query"`
	SubQueries map[string]string      `yaml:"subqueries,omitempty"`
}

func (s *SqlCommandSpec) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("sql command spec is missing name")
	}
	if strings.TrimSpace(s.Short) == "" {
		return errors.Errorf("sql command spec %q is missing short description", s.Name)
	}
	if strings.TrimSpace(s.Query) == "" {
		return errors.Errorf("sql command spec %q is missing query body", s.Name)
	}
	return nil
}

type SqlCommandCompiler struct {
	DBConnectionFactory clay_sql.DBConnectionFactory
}

func (c *SqlCommandCompiler) Compile(
	spec *SqlCommandSpec,
	options ...cmds.CommandDescriptionOption,
) (*SqlCommand, error) {
	if spec == nil {
		return nil, errors.New("sql command spec is nil")
	}
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	cmd, err := NewSqlCommand(
		cmds.NewCommandDescription(spec.Name),
		WithDbConnectionFactory(c.DBConnectionFactory),
		WithQuery(spec.Query),
		WithSubQueries(spec.SubQueries),
	)
	if err != nil {
		return nil, err
	}

	normalizedFlags := normalizeOptionalBoolFlags(spec.Flags)

	options_ := []cmds.CommandDescriptionOption{
		cmds.WithShort(spec.Short),
		cmds.WithLong(spec.Long),
		cmds.WithFlags(normalizedFlags...),
		cmds.WithArguments(spec.Arguments...),
		cmds.WithTags(spec.Tags...),
		cmds.WithMetadata(spec.Metadata),
		cmds.WithLayout(&layout.Layout{
			Sections: spec.Layout,
		}),
	}
	options_ = append(options_, options...)

	for _, option := range options_ {
		option(cmd.Description())
	}

	if !cmd.IsValid() {
		return nil, errors.New("invalid sql command")
	}

	return cmd, nil
}

func normalizeOptionalBoolFlags(flags []*fields.Definition) []*fields.Definition {
	if len(flags) == 0 {
		return nil
	}

	ret := make([]*fields.Definition, 0, len(flags))
	for _, flag := range flags {
		if flag == nil {
			ret = append(ret, nil)
			continue
		}

		cloned := flag.Clone()
		if cloned.Type == fields.TypeBool && !cloned.Required && cloned.Default == nil {
			defaultValue := interface{}(false)
			cloned.Default = &defaultValue
		}
		ret = append(ret, cloned)
	}

	return ret
}

func ParseSQLFileSpec(path string, contents []byte) (*SqlCommandSpec, error) {
	metadataText, body, err := splitSqletonSQLPreamble(contents)
	if err != nil {
		return nil, errors.Wrapf(err, "parse sqleton sql preamble: %s", path)
	}

	spec := &SqlCommandSpec{}
	decoder := yaml.NewDecoder(strings.NewReader(metadataText))
	if err := decoder.Decode(spec); err != nil {
		return nil, errors.Wrapf(err, "decode sqleton sql metadata: %s", path)
	}

	if len(spec.SubQueries) > 0 {
		return nil, errors.Errorf("sql command spec %q uses subqueries in metadata; inline them with SQL or CTEs instead", path)
	}

	spec.Query = strings.TrimSpace(body)
	if err := spec.Validate(); err != nil {
		return nil, errors.Wrapf(err, "validate sqleton sql command: %s", path)
	}
	return spec, nil
}

func splitSqletonSQLPreamble(contents []byte) (string, string, error) {
	s := strings.TrimLeft(string(contents), "\ufeff\r\n\t ")
	if !strings.HasPrefix(s, "/*") {
		return "", "", errors.New("missing sqleton sql preamble")
	}

	end := strings.Index(s, "*/")
	if end == -1 {
		return "", "", errors.New("unterminated sqleton sql preamble")
	}

	raw := strings.TrimSpace(s[2:end])
	if !strings.HasPrefix(raw, "sqleton") {
		return "", "", errors.New("invalid sqleton sql preamble marker")
	}

	metadata := strings.TrimSpace(strings.TrimPrefix(raw, "sqleton"))
	body := strings.TrimSpace(s[end+2:])
	if metadata == "" {
		return "", "", errors.New("empty sqleton sql preamble metadata")
	}
	if body == "" {
		return "", "", errors.New("empty sqleton sql query body")
	}

	return metadata, body, nil
}

func LooksLikeSqletonSQLCommand(contents []byte) bool {
	s := strings.TrimLeft(string(contents), "\ufeff\r\n\t ")
	if !strings.HasPrefix(s, "/*") {
		return false
	}

	end := strings.Index(s, "*/")
	if end == -1 {
		return false
	}

	raw := strings.TrimSpace(s[2:end])
	return strings.HasPrefix(raw, "sqleton")
}

func MarshalSpecToSQLFile(spec *SqlCommandSpec) (string, error) {
	if spec == nil {
		return "", errors.New("sql command spec is nil")
	}
	if err := spec.Validate(); err != nil {
		return "", err
	}
	if len(spec.SubQueries) > 0 {
		return "", errors.New("sql command spec with subqueries cannot be marshaled to a .sql file")
	}

	metadata := &SqlCommandSpec{
		Name:      spec.Name,
		Short:     spec.Short,
		Long:      spec.Long,
		Layout:    spec.Layout,
		Flags:     spec.Flags,
		Arguments: spec.Arguments,
		Tags:      spec.Tags,
		Metadata:  spec.Metadata,
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(metadata); err != nil {
		return "", errors.Wrap(err, "encode sqleton sql metadata")
	}
	if err := encoder.Close(); err != nil {
		return "", errors.Wrap(err, "close sqleton sql metadata encoder")
	}

	metadataText := strings.TrimSpace(buf.String())
	query := strings.TrimSpace(spec.Query)

	var ret strings.Builder
	ret.WriteString("/* sqleton\n")
	ret.WriteString(metadataText)
	ret.WriteString("\n*/\n")
	ret.WriteString(query)
	ret.WriteString("\n")
	return ret.String(), nil
}

func ParseSQLFileSpecFromReader(path string, r io.Reader) (*SqlCommandSpec, error) {
	contents, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrapf(err, "read sqleton sql command: %s", path)
	}
	return ParseSQLFileSpec(path, contents)
}
