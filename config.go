package main

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type DatabaseConfig struct {
	Name       string `toml:"name"`
	Database   string `toml:"database"`
	Port       int    `toml:"port"`
	AWSProfile string `toml:"aws_profile"`
	AWSRegion  string `toml:"aws_region"`
}

type Config struct {
	AWSProfile string           `toml:"aws_profile"`
	AWSRegion  string           `toml:"aws_region"`
	Databases  []DatabaseConfig `toml:"database"`
}

func LoadConfig(path string) (Config, error) {
	var cfg Config

	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("could not read config file: %w", err)
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("could not parse config file: %w", err)
	}

	if len(cfg.Databases) == 0 {
		return cfg, fmt.Errorf("no databases configured in config.toml")
	}

	for i, db := range cfg.Databases {
		if db.Name == "" {
			return cfg, fmt.Errorf("database entry %d: missing required field 'name'", i+1)
		}
		if db.Database == "" {
			return cfg, fmt.Errorf("database entry %d (%s): missing required field 'database'", i+1, db.Name)
		}
		if db.Port == 0 {
			return cfg, fmt.Errorf("database entry %d (%s): missing required field 'port'", i+1, db.Name)
		}

		// Merge global defaults
		if db.AWSProfile == "" {
			cfg.Databases[i].AWSProfile = cfg.AWSProfile
		}
		if db.AWSRegion == "" {
			cfg.Databases[i].AWSRegion = cfg.AWSRegion
		}
	}

	return cfg, nil
}
