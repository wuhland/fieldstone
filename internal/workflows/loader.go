package workflows

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadWorkflows reads all *.yaml files from dir and returns a map of
// resource_type → WorkflowConfig. Moved from services/workflow/loader.go
// so both the worker and domain service clients can share the types.
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
		slog.Info("loaded workflow", "resource_type", wf.ResourceType,
			"statuses", len(wf.Statuses), "transitions", len(wf.Transitions))
	}
	return configs, nil
}
