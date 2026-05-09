package content

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

type scenarioIndex struct {
	DefaultScenarioID string           `yaml:"defaultScenarioId"`
	Records           []scenarioRecord `yaml:"records"`
}

type scenarioRecord struct {
	ID      string `yaml:"id"`
	Enabled bool   `yaml:"enabled"`
	Version int    `yaml:"version"`
	File    string `yaml:"file"`
	Name    string `yaml:"name"`
	Default bool   `yaml:"default"`
	Summary string `yaml:"summary"`
}

// ResolveNightglassScenarioID resolves the configured scenario ID using the
// installed Nightglass index fallback chain described in the README.
func ResolveNightglassScenarioID(repoRoot, configuredID string) (string, error) {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return "", fmt.Errorf("repo root path is required")
	}

	indexPath := filepath.Join(repoRoot, NightglassInstallDir, "scenarios", "index.yaml")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return "", fmt.Errorf("read Nightglass scenario index: %w", err)
	}

	var index scenarioIndex
	if err := yaml.Unmarshal(data, &index); err != nil {
		return "", fmt.Errorf("decode Nightglass scenario index: %w", err)
	}

	selected, ok := selectNightglassScenarioID(index, configuredID)
	if !ok {
		return "", fmt.Errorf("no enabled Nightglass scenarios available in %s", indexPath)
	}
	return selected, nil
}

func selectNightglassScenarioID(index scenarioIndex, configuredID string) (string, bool) {
	if id := strings.TrimSpace(configuredID); id != "" && scenarioEnabled(index.Records, id) {
		return id, true
	}
	if id := strings.TrimSpace(index.DefaultScenarioID); id != "" && scenarioEnabled(index.Records, id) {
		return id, true
	}

	records := make([]scenarioRecord, 0, len(index.Records))
	for _, record := range index.Records {
		if record.Enabled && strings.TrimSpace(record.ID) != "" {
			records = append(records, record)
		}
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ID < records[j].ID })
	if len(records) == 0 {
		return "", false
	}
	return records[0].ID, true
}

func scenarioEnabled(records []scenarioRecord, id string) bool {
	for _, record := range records {
		if record.ID == id && record.Enabled {
			return true
		}
	}
	return false
}
