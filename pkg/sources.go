package pkg

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Source struct {
	Name     string
	Type     string `yaml:"type"`
	Hostname string `yaml:"server"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Schema   string `yaml:"schema"`
	Database string `yaml:"database"`
}

type dbtProfile struct {
	Target  string             `yaml:"target"`
	Outputs map[string]*Source `yaml:"outputs"`
}

type dbtProfiles = map[string]*dbtProfile

func ParseDbtProfiles(profilesPath string) ([]*Source, error) {
	// Parse dbt profiles.yml
	// Return []*Source

	// parse YAML from ~/.dbt/profiles.yml

	// Read the file contents
	if profilesPath == "" {
		// replace ~ with $HOME in the path
		profilesPath = os.ExpandEnv("$HOME/.dbt/profiles.yml")
	}

	data, err := os.ReadFile(profilesPath)
	if err != nil {
		return nil, err
	}

	var profiles dbtProfiles

	// Parse the YAML file
	err = yaml.Unmarshal(data, &profiles)
	if err != nil {
		return nil, err
	}

	var ret []*Source

	// Print the profiles
	for _, profile := range profiles {
		for outputName, source := range profile.Outputs {
			source.Name = outputName
			ret = append(ret, source)
		}
	}

	return ret, nil
}
