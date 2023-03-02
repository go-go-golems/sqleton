package pkg

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// Source is the generic structure we use to represent
// a database connection string
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

func (s *Source) ToConnectionString() string {
	switch s.Type {
	case "postgres":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", s.Hostname, s.Port, s.Username, s.Password, s.Database)
	case "mysql":
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", s.Username, s.Password, s.Hostname, s.Port, s.Database)
	case "sqlite":
		fallthrough
	case "sqlite3":
		return s.Database
	default:
		return ""
	}
}

type dbtProfile struct {
	Target  string             `yaml:"target"`
	Outputs map[string]*Source `yaml:"outputs"`
}

type dbtProfiles = map[string]*dbtProfile

// ParseDbtProfiles parses a dbt profiles.yml file and returns a map of sources
func ParseDbtProfiles(profilesPath string) ([]*Source, error) {
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
	for name, profile := range profiles {
		for outputName, source := range profile.Outputs {
			source.Name = fmt.Sprintf("%s.%s", name, outputName)
			ret = append(ret, source)
		}
	}

	return ret, nil
}
