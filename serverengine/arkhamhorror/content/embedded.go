package content

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	// NightglassEmbeddedRoot is the embedded content subtree rooted in this package.
	NightglassEmbeddedRoot = "nightglass"
	// NightglassInstallDir is the repository-relative install path used at runtime.
	NightglassInstallDir = "serverengine/arkhamhorror/content/nightglass"
)

//go:embed nightglass/**
var nightglassEmbeddedFS embed.FS

// EnsureNightglassContentInstalled copies embedded Nightglass content files into
// the module-scoped runtime content directory when files are missing.
//
// Existing files are preserved and never overwritten.
func EnsureNightglassContentInstalled(repoRoot string) error {
	repoRoot = strings.TrimSpace(repoRoot)
	if repoRoot == "" {
		return fmt.Errorf("repo root path is required")
	}

	installBase := filepath.Join(repoRoot, NightglassInstallDir)

	err := fs.WalkDir(nightglassEmbeddedFS, NightglassEmbeddedRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk embedded content %q: %w", path, walkErr)
		}

		rel, err := filepath.Rel(NightglassEmbeddedRoot, path)
		if err != nil {
			return fmt.Errorf("resolve relative path for %q: %w", path, err)
		}

		target := installBase
		if rel != "." {
			target = filepath.Join(installBase, rel)
		}

		if d.IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create content directory %q: %w", target, err)
			}
			return nil
		}

		if _, err := os.Stat(target); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("stat target file %q: %w", target, err)
		}

		payload, err := nightglassEmbeddedFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read embedded file %q: %w", path, err)
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create parent directory for %q: %w", target, err)
		}

		if err := os.WriteFile(target, payload, 0o644); err != nil {
			return fmt.Errorf("write embedded content to %q: %w", target, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("install embedded Nightglass content: %w", err)
	}

	return nil
}
