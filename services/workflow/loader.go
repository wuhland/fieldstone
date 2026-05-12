package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type WorkflowConfig struct {
	ResourceType  string       `yaml:"resource_type"`
	InitialStatus string       `yaml:"initial_status"`
	Statuses      []Status     `yaml:"statuses"`
	Transitions   []Transition `yaml:"transitions"`
}

type Status struct {
	Name     string `yaml:"name"`
	Label    string `yaml:"label"`
	Terminal bool   `yaml:"terminal"`
}

type Transition struct {
	From        string   `yaml:"from"`
	To          string   `yaml:"to"`
	Roles       []string `yaml:"roles"`
	NotifyEvent string   `yaml:"notify_event,omitempty"`
}

// LoadWorkflows reads all *.yaml files from dir and returns a map of resource_type → config.
func LoadWorkflows(dir string) (map[string]*WorkflowConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read workflows dir: %w", err)
	}

	configs := make(map[string]*WorkflowConfig)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var wf WorkflowConfig
		if err := yaml.Unmarshal(data, &wf); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		configs[wf.ResourceType] = &wf
		slog.Info("loaded workflow", "resource_type", wf.ResourceType, "statuses", len(wf.Statuses), "transitions", len(wf.Transitions))
	}
	return configs, nil
}
