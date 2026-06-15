// Package config loads the YAML configuration that defines scheduled scan
// targets and platform settings.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config is the top-level platform configuration.
type Config struct {
	// StoreDir is where drift reports are persisted.
	StoreDir string `yaml:"store_dir"`
	// Addr is the dashboard listen address (used by serve).
	Addr string `yaml:"addr"`
	// Targets are the scan targets, each optionally scheduled.
	Targets []Target `yaml:"targets"`
}

// Target describes one thing to scan: a state source + a provider.
type Target struct {
	Name     string      `yaml:"name"`
	State    StateConfig `yaml:"state"`
	Provider string      `yaml:"provider"`
	// Schedule is a cron expression (e.g. "0 * * * *"). Empty means the target
	// is only scanned on demand, never automatically.
	Schedule string `yaml:"schedule"`
	// Region/Project/Subscription configure region-scoped providers.
	Region       string `yaml:"region"`
	Project      string `yaml:"project"`
	Subscription string `yaml:"subscription"`
	// Options are provider-specific key/values (e.g. mock fixture path).
	Options map[string]string `yaml:"options"`
}

// StateConfig points at a Terraform state source.
type StateConfig struct {
	// Type is "local" or "s3".
	Type string `yaml:"type"`
	// Path is the file path (local) or s3://bucket/key (s3).
	Path string `yaml:"path"`
	// Region is the bucket region for s3 sources.
	Region string `yaml:"region"`
}

// Load reads and validates a YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", path, err)
	}
	if c.StoreDir == "" {
		c.StoreDir = "./reports"
	}
	if c.Addr == "" {
		c.Addr = ":8080"
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	if len(c.Targets) == 0 {
		return fmt.Errorf("config has no targets")
	}
	seen := map[string]bool{}
	for i, t := range c.Targets {
		if t.Name == "" {
			return fmt.Errorf("target %d: name is required", i)
		}
		if seen[t.Name] {
			return fmt.Errorf("duplicate target name %q", t.Name)
		}
		seen[t.Name] = true
		if t.Provider == "" {
			return fmt.Errorf("target %q: provider is required", t.Name)
		}
		if t.State.Path == "" {
			return fmt.Errorf("target %q: state.path is required", t.Name)
		}
	}
	return nil
}
