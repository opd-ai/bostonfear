// Package scenarios defines Arkham Horror scenario templates: starting investigator count,
// initial location distribution, difficulty thresholds, act/agenda decks, and win conditions.
//
// Scenarios are immutable content that GameServer loads on startup. Each scenario
// governs the flow from initial setup through act/agenda progression to game end.
package scenarios

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ScenarioTemplate holds the metadata and setup parameters for a single scenario.
// Fields map directly to the YAML schema used in the Nightglass content pack.
type ScenarioTemplate struct {
	ID          string // Unique scenario identifier (e.g. "scn.nightglass.harbor-signal")
	Name        string // Display name
	Summary     string // Short narrative summary
	Enabled     bool   // Whether the scenario is selectable
	Version     int    // Content version for cache-busting
	InitialDoom int    // Starting doom value (overrides game constant when > 0)
}

// indexRecord is the internal YAML representation from index.yaml.
type indexRecord struct {
	ID      string `yaml:"id"`
	Name    string `yaml:"name"`
	Summary string `yaml:"summary"`
	Enabled bool   `yaml:"enabled"`
	Version int    `yaml:"version"`
}

// indexFile is the top-level YAML structure for scenarios/index.yaml.
type indexFile struct {
	DefaultScenarioID string        `yaml:"defaultScenarioId"`
	Records           []indexRecord `yaml:"records"`
}

// Index is an ordered collection of ScenarioTemplates with a default scenario ID.
type Index struct {
	DefaultScenarioID string
	Templates         []ScenarioTemplate
}

// LoadIndex reads the scenario index from the given filesystem at the given path.
// It returns a populated Index or an error if the file cannot be parsed.
func LoadIndex(fsys fs.FS, indexPath string) (*Index, error) {
	data, err := fs.ReadFile(fsys, indexPath)
	if err != nil {
		return nil, fmt.Errorf("read scenario index %q: %w", indexPath, err)
	}

	var raw indexFile
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse scenario index %q: %w", indexPath, err)
	}

	templates := make([]ScenarioTemplate, 0, len(raw.Records))
	for _, r := range raw.Records {
		if strings.TrimSpace(r.ID) == "" {
			continue
		}
		templates = append(templates, ScenarioTemplate{
			ID:      r.ID,
			Name:    r.Name,
			Summary: r.Summary,
			Enabled: r.Enabled,
			Version: r.Version,
		})
	}

	return &Index{
		DefaultScenarioID: raw.DefaultScenarioID,
		Templates:         templates,
	}, nil
}

// Resolve returns the ScenarioTemplate matching the selection priority:
//  1. configuredID if non-empty and enabled
//  2. DefaultScenarioID if non-empty and enabled
//  3. First enabled scenario sorted by ID
//  4. Error if none available
func (idx *Index) Resolve(configuredID string) (ScenarioTemplate, error) {
	// Step 1: configured ID
	if id := strings.TrimSpace(configuredID); id != "" {
		if tmpl, ok := idx.find(id); ok && tmpl.Enabled {
			return tmpl, nil
		}
	}

	// Step 2: index default
	if id := strings.TrimSpace(idx.DefaultScenarioID); id != "" {
		if tmpl, ok := idx.find(id); ok && tmpl.Enabled {
			return tmpl, nil
		}
	}

	// Step 3: first enabled sorted by ID
	enabled := make([]ScenarioTemplate, 0, len(idx.Templates))
	for _, t := range idx.Templates {
		if t.Enabled {
			enabled = append(enabled, t)
		}
	}
	sort.Slice(enabled, func(i, j int) bool { return enabled[i].ID < enabled[j].ID })
	if len(enabled) > 0 {
		return enabled[0], nil
	}

	return ScenarioTemplate{}, fmt.Errorf("no enabled scenarios found in index")
}

// FindByID looks up a scenario by exact ID. Returns an error if not found.
func (idx *Index) FindByID(id string) (ScenarioTemplate, error) {
	if tmpl, ok := idx.find(id); ok {
		return tmpl, nil
	}
	return ScenarioTemplate{}, fmt.Errorf("scenario %q not found in index", id)
}

// find looks up a scenario by ID. Returns the template and true if found.
func (idx *Index) find(id string) (ScenarioTemplate, bool) {
	for _, t := range idx.Templates {
		if t.ID == id {
			return t, true
		}
	}
	return ScenarioTemplate{}, false
}
