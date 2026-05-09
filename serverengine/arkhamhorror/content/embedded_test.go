package content

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureNightglassContentInstalled_CopiesMissingFiles(t *testing.T) {
	repoRoot := t.TempDir()

	if err := EnsureNightglassContentInstalled(repoRoot); err != nil {
		t.Fatalf("EnsureNightglassContentInstalled() error = %v", err)
	}

	manifestPath := filepath.Join(repoRoot, NightglassInstallDir, "manifest.yaml")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("expected manifest at %q: %v", manifestPath, err)
	}

	scenarioPath := filepath.Join(repoRoot, NightglassInstallDir, "scenarios", "nightglass-harbor-signal.yaml")
	if _, err := os.Stat(scenarioPath); err != nil {
		t.Fatalf("expected scenario at %q: %v", scenarioPath, err)
	}
}

func TestEnsureNightglassContentInstalled_PreservesExistingFiles(t *testing.T) {
	repoRoot := t.TempDir()
	manifestPath := filepath.Join(repoRoot, NightglassInstallDir, "manifest.yaml")

	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	custom := []byte("custom-manifest\n")
	if err := os.WriteFile(manifestPath, custom, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := EnsureNightglassContentInstalled(repoRoot); err != nil {
		t.Fatalf("EnsureNightglassContentInstalled() error = %v", err)
	}

	got, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	if string(got) != string(custom) {
		t.Fatalf("expected existing file to be preserved, got %q", string(got))
	}
}
