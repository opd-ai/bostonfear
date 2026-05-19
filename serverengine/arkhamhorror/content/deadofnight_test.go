package content

import (
	"os"
	"path/filepath"
	"testing"

	"go.yaml.in/yaml/v3"
)

// TestDeadOfNightManifest verifies Dead of Night expansion manifest structure.
func TestDeadOfNightManifest(t *testing.T) {
	manifestPath := filepath.Join("deadofnight", "manifest.yaml")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read Dead of Night manifest: %v", err)
	}

	var manifest struct {
		SchemaVersion   string   `yaml:"schemaVersion"`
		ContentPackID   string   `yaml:"contentPackId"`
		DefaultScenario string   `yaml:"defaultScenarioId"`
		Dependencies    []string `yaml:"dependencies"`
	}
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("decode manifest: %v", err)
	}

	if manifest.SchemaVersion != "1.0.0" {
		t.Errorf("expected schemaVersion 1.0.0, got %q", manifest.SchemaVersion)
	}
	if manifest.ContentPackID != "deadofnight.expansion" {
		t.Errorf("expected contentPackId deadofnight.expansion, got %q", manifest.ContentPackID)
	}
	if manifest.DefaultScenario != "scn.deadofnight.museum-awakening" {
		t.Errorf("expected default scenario scn.deadofnight.museum-awakening, got %q", manifest.DefaultScenario)
	}
	if len(manifest.Dependencies) == 0 {
		t.Error("expansion should declare nightglass.core dependency")
	}
}

// TestDeadOfNightInvestigators verifies investigators are properly defined.
func TestDeadOfNightInvestigators(t *testing.T) {
	invPath := filepath.Join("deadofnight", "base-set", "investigators.yaml")
	data, err := os.ReadFile(invPath)
	if err != nil {
		t.Fatalf("read investigators: %v", err)
	}

	var doc struct {
		Records []struct {
			ID      string `yaml:"id"`
			Name    string `yaml:"name"`
			Enabled bool   `yaml:"enabled"`
		} `yaml:"records"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode investigators: %v", err)
	}

	expectedInvestigators := []string{
		"inv.evelyn-cross",
		"inv.marcus-graves",
		"inv.vera-night",
		"inv.silas-thorne",
	}

	if len(doc.Records) != len(expectedInvestigators) {
		t.Errorf("expected %d investigators, got %d", len(expectedInvestigators), len(doc.Records))
	}

	for i, inv := range doc.Records {
		if i >= len(expectedInvestigators) {
			break
		}
		if inv.ID != expectedInvestigators[i] {
			t.Errorf("investigator %d: expected ID %q, got %q", i, expectedInvestigators[i], inv.ID)
		}
		if !inv.Enabled {
			t.Errorf("investigator %q should be enabled", inv.ID)
		}
		if inv.Name == "" {
			t.Errorf("investigator %q missing name", inv.ID)
		}
	}
}

// TestDeadOfNightScenarios verifies scenario files exist and are valid.
func TestDeadOfNightScenarios(t *testing.T) {
	indexPath := filepath.Join("deadofnight", "scenarios", "index.yaml")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("read scenario index: %v", err)
	}

	var index struct {
		DefaultScenarioID string `yaml:"defaultScenarioId"`
		Records           []struct {
			ID      string `yaml:"id"`
			Enabled bool   `yaml:"enabled"`
			File    string `yaml:"file"`
		} `yaml:"records"`
	}
	if err := yaml.Unmarshal(data, &index); err != nil {
		t.Fatalf("decode scenario index: %v", err)
	}

	if index.DefaultScenarioID != "scn.deadofnight.museum-awakening" {
		t.Errorf("expected default scenario scn.deadofnight.museum-awakening, got %q", index.DefaultScenarioID)
	}

	if len(index.Records) != 2 {
		t.Errorf("expected 2 scenarios, got %d", len(index.Records))
	}

	for _, record := range index.Records {
		if !record.Enabled {
			t.Errorf("scenario %q should be enabled", record.ID)
		}
		if record.File == "" {
			t.Errorf("scenario %q missing file path", record.ID)
		}
	}
}

// TestDeadOfNightEncounters verifies museum and graveyard encounter decks.
func TestDeadOfNightEncounters(t *testing.T) {
	encounterPath := filepath.Join("deadofnight", "base-set", "encounters.yaml")
	data, err := os.ReadFile(encounterPath)
	if err != nil {
		t.Fatalf("read encounters: %v", err)
	}

	var doc struct {
		Records []struct {
			ID            string `yaml:"id"`
			EncounterDeck string `yaml:"encounterDeck"`
			Enabled       bool   `yaml:"enabled"`
		} `yaml:"records"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("decode encounters: %v", err)
	}

	museumCount := 0
	graveyardCount := 0
	for _, enc := range doc.Records {
		if !enc.Enabled {
			t.Errorf("encounter %q should be enabled", enc.ID)
		}
		switch enc.EncounterDeck {
		case "museum":
			museumCount++
		case "graveyard":
			graveyardCount++
		case "nocturnal":
			// nocturnal encounters are shared
		default:
			if enc.EncounterDeck != "" {
				t.Errorf("unexpected encounter deck %q for %q", enc.EncounterDeck, enc.ID)
			}
		}
	}

	if museumCount < 2 {
		t.Errorf("expected at least 2 museum encounters, got %d", museumCount)
	}
	if graveyardCount < 2 {
		t.Errorf("expected at least 2 graveyard encounters, got %d", graveyardCount)
	}
}
