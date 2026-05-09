package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveNightglassScenarioID(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, NightglassInstallDir, "scenarios", "index.yaml")
	if err := os.MkdirAll(filepath.Dir(indexPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	index := strings.Join([]string{
		"defaultScenarioId: \"scn.default\"",
		"records:",
		"  - id: \"scn.alpha\"",
		"    enabled: true",
		"  - id: \"scn.default\"",
		"    enabled: true",
		"  - id: \"scn.zulu\"",
		"    enabled: false",
	}, "\n")
	if err := os.WriteFile(indexPath, []byte(index), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Run("configured id wins when enabled", func(t *testing.T) {
		got, err := ResolveNightglassScenarioID(root, "scn.alpha")
		if err != nil {
			t.Fatalf("ResolveNightglassScenarioID() error = %v", err)
		}
		if got != "scn.alpha" {
			t.Fatalf("expected scn.alpha, got %q", got)
		}
	})

	t.Run("default id used when configured missing", func(t *testing.T) {
		got, err := ResolveNightglassScenarioID(root, "")
		if err != nil {
			t.Fatalf("ResolveNightglassScenarioID() error = %v", err)
		}
		if got != "scn.default" {
			t.Fatalf("expected scn.default, got %q", got)
		}
	})

	t.Run("first enabled id used when default missing", func(t *testing.T) {
		altIndex := strings.Join([]string{
			"records:",
			"  - id: \"scn.bravo\"",
			"    enabled: true",
			"  - id: \"scn.alpha\"",
			"    enabled: true",
		}, "\n")
		if err := os.WriteFile(indexPath, []byte(altIndex), 0o644); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		got, err := ResolveNightglassScenarioID(root, "scn.missing")
		if err != nil {
			t.Fatalf("ResolveNightglassScenarioID() error = %v", err)
		}
		if got != "scn.alpha" {
			t.Fatalf("expected first enabled ID alphabetically, got %q", got)
		}
	})
}
