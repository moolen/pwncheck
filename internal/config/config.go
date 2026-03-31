package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Release      ReleaseConfig      `yaml:"release"`
	Repositories []RepositoryConfig `yaml:"repositories"`
}

type ReleaseConfig struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
	Tag   string `yaml:"tag"`
	Name  string `yaml:"name"`
}

type RepositoryConfig struct {
	Name    string           `yaml:"name"`
	Package string           `yaml:"package"`
	Policy  ProvenancePolicy `yaml:"policy"`
}

type ProvenancePolicy struct {
	Issuer       string `yaml:"issuer"`
	Repository   string `yaml:"repository"`
	WorkflowPath string `yaml:"workflowPath"`
	Ref          string `yaml:"ref"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config %q: %w", path, err)
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c Config) Validate() error {
	if c.Release.Owner == "" || c.Release.Repo == "" || c.Release.Tag == "" || c.Release.Name == "" {
		return errors.New("release.owner, release.repo, release.tag, and release.name are required")
	}

	if len(c.Repositories) == 0 {
		return errors.New("at least one repository must be configured")
	}

	for i, repo := range c.Repositories {
		if repo.Name == "" {
			return fmt.Errorf("repositories[%d].name is required", i)
		}
		if repo.Package == "" {
			return fmt.Errorf("repositories[%d].package is required", i)
		}

		policy := repo.Policy
		switch {
		case policy.Issuer == "":
			return fmt.Errorf("repositories[%d].policy.issuer is required", i)
		case policy.Repository == "":
			return fmt.Errorf("repositories[%d].policy.repository is required", i)
		case policy.WorkflowPath == "":
			return fmt.Errorf("repositories[%d].policy.workflowPath is required", i)
		case policy.Ref == "":
			return fmt.Errorf("repositories[%d].policy.ref is required", i)
		}
	}

	return nil
}
