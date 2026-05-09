package render

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseVisualManifest_Valid(t *testing.T) {
	input := []byte(`
content:
  visuals:
    version: 1
    basePath: assets/png
    placeholders:
      missing: ui/missing.png
    components:
      board.background:
        file: board/board_main.png
      ui.action.button.default:
        file: ui/actions/default.png
        hover: ui/actions/hover.png
        pressed: ui/actions/pressed.png
        disabled: ui/actions/disabled.png
`)

	manifest, err := ParseVisualManifest(input)
	if err != nil {
		t.Fatalf("ParseVisualManifest returned error: %v", err)
	}
	if manifest == nil {
		t.Fatal("ParseVisualManifest returned nil manifest")
	}
	if got := manifest.Content.Visuals.Version; got != 1 {
		t.Fatalf("visuals.version = %d, want 1", got)
	}
	if got := len(manifest.Content.Visuals.Components); got != 2 {
		t.Fatalf("len(components) = %d, want 2", got)
	}
}

func TestParseVisualManifest_FailsOnMissingRequiredFields(t *testing.T) {
	input := []byte(`
content:
  visuals:
    version: 1
    placeholders:
      missing: ui/missing.png
    components:
      board.background:
        file: board/board_main.png
`)

	_, err := ParseVisualManifest(input)
	assertValidationErrorContains(t, err, "content.visuals.basePath is required")
}

func TestParseVisualManifest_FailsOnDuplicateComponentKeys(t *testing.T) {
	input := []byte(`
content:
  visuals:
    version: 1
    basePath: assets/png
    placeholders:
      missing: ui/missing.png
    components:
      board.background:
        file: board/a.png
      board.background:
        file: board/b.png
`)

	_, err := ParseVisualManifest(input)
	if err == nil {
		t.Fatal("expected duplicate key parse error, got nil")
	}
	if !strings.Contains(err.Error(), "already defined") {
		t.Fatalf("error %q does not contain duplicate key decode failure", err.Error())
	}
}

func TestParseVisualManifest_FailsOnInvalidPathAndFormat(t *testing.T) {
	input := []byte(`
content:
  visuals:
    version: 1
    basePath: assets/png
    placeholders:
      missing: ui/missing.jpg
    components:
      board.background:
        file: board/background.jpg
`)

	_, err := ParseVisualManifest(input)
	assertValidationErrorContains(t, err, "only .png assets are supported")
}

func TestParseVisualManifest_FailsOnWrongVersion(t *testing.T) {
	input := []byte(`
content:
  visuals:
    version: 2
    basePath: assets/png
    placeholders:
      missing: ui/missing.png
    components:
      board.background:
        file: board/board_main.png
`)

	_, err := ParseVisualManifest(input)
	assertValidationErrorContains(t, err, "content.visuals.version must be 1")
}

func TestParseVisualManifest_FailsOnEmptyInput(t *testing.T) {
	_, err := ParseVisualManifest(nil)
	if err == nil {
		t.Fatal("expected error for empty visual manifest input")
	}
	if !strings.Contains(err.Error(), "visual manifest is empty") {
		t.Fatalf("error %q does not contain %q", err.Error(), "visual manifest is empty")
	}
}

func TestLoadVisualManifestFile_Valid(t *testing.T) {
	manifestPath := filepath.Join(t.TempDir(), "visuals.yaml")
	input := []byte(`
content:
  visuals:
    version: 1
    basePath: assets/png
    placeholders:
      missing: ui/missing.png
    components:
      board.background:
        file: board/board_main.png
`)

	if err := os.WriteFile(manifestPath, input, 0o600); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	manifest, err := LoadVisualManifestFile(manifestPath)
	if err != nil {
		t.Fatalf("LoadVisualManifestFile returned error: %v", err)
	}
	if manifest == nil {
		t.Fatal("LoadVisualManifestFile returned nil manifest")
	}
	if got := manifest.Content.Visuals.BasePath; got != "assets/png" {
		t.Fatalf("visuals.basePath = %q, want %q", got, "assets/png")
	}
}

func TestLoadVisualManifestFile_FailsOnMissingFile(t *testing.T) {
	_, err := LoadVisualManifestFile(filepath.Join(t.TempDir(), "missing.yaml"))
	if err == nil {
		t.Fatal("expected read error for missing manifest file")
	}
	if !strings.Contains(err.Error(), "read visual manifest") {
		t.Fatalf("error %q does not contain %q", err.Error(), "read visual manifest")
	}
}

func assertValidationErrorContains(t *testing.T, err error, wantSubstr string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", wantSubstr)
	}
	var validationErr *ManifestValidationError
	if !errors.As(err, &validationErr) {
		t.Fatalf("expected ManifestValidationError, got %T (%v)", err, err)
	}
	if !strings.Contains(err.Error(), wantSubstr) {
		t.Fatalf("error %q does not contain %q", err.Error(), wantSubstr)
	}
}
