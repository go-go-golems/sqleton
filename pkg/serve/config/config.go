package config

import "gopkg.in/yaml.v3"

type Route struct {
	Path              string       `yaml:"path"`
	CommandDirectory  *CommandDir  `yaml:"commandDirectory,omitempty"`
	Command           *Command     `yaml:"command,omitempty"`
	Static            *Static      `yaml:"static,omitempty"`
	StaticFile        *StaticFile  `yaml:"staticFile,omitempty"`
	TemplateDirectory *TemplateDir `yaml:"templateDirectory,omitempty"`
	Template          *Template    `yaml:"template,omitempty"`
}

type CommandDir struct {
	Repositories      []string          `yaml:"repositories"`
	TemplateDirectory string            `yaml:"templateDirectory,omitempty"`
	TemplateName      string            `yaml:"templateName,omitempty"`
	IndexTemplateName string            `yaml:"indexTemplateName,omitempty"`
	AdditionalData    map[string]string `yaml:"additionalData,omitempty"`
	Defaults          *LayerParams      `yaml:"defaults,omitempty"`
	Overrides         *LayerParams      `yaml:"overrides,omitempty"`
}

type Command struct {
	File           string            `yaml:"file"`
	TemplateName   string            `yaml:"templateName"`
	AdditionalData map[string]string `yaml:"additionalData,omitempty"`
	Defaults       *LayerParams      `yaml:"defaults,omitempty"`
	Overrides      *LayerParams      `yaml:"overrides,omitempty"`
}

type Static struct {
	LocalPath string `yaml:"localPath"`
}

type StaticFile struct {
	LocalPath string `yaml:"localPath"`
}

type TemplateDir struct {
	LocalDirectory    string                 `yaml:"localDirectory"`
	IndexTemplateName string                 `yaml:"indexTemplateName,omitempty"`
	AdditionalData    map[string]interface{} `yaml:"additionalData,omitempty"`
}

type Template struct {
	TemplateFile string `yaml:"templateFile"`
}

type LayerParams struct {
	Layers    map[string]map[string]interface{} `yaml:"layers,omitempty"`
	Flags     map[string]interface{}            `yaml:"flags,omitempty"`
	Arguments map[string]interface{}            `yaml:"arguments,omitempty"`
}

type Config struct {
	Routes []Route `yaml:"routes"`
}

func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	err := yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
