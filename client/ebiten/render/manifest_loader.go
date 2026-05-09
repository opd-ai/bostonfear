package render

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"
)

const manifestSchemaVersion = 1

// VisualManifest is the root YAML contract for client-side visual asset mapping.
type VisualManifest struct {
	Content struct {
		Visuals VisualConfig `yaml:"visuals"`
	} `yaml:"content"`
}

// VisualConfig contains all versioned visual-asset configuration fields.
type VisualConfig struct {
	Version      int                       `yaml:"version"`
	BasePath     string                    `yaml:"basePath"`
	Placeholders PlaceholderConfig         `yaml:"placeholders"`
	Components   map[string]ComponentAsset `yaml:"components"`
}

// PlaceholderConfig defines fallback assets used during runtime recovery.
type PlaceholderConfig struct {
	Missing string `yaml:"missing"`
}

// ComponentAsset maps one logical UI/render component to one or more PNG files.
type ComponentAsset struct {
	File      string `yaml:"file"`
	Hover     string `yaml:"hover,omitempty"`
	Pressed   string `yaml:"pressed,omitempty"`
	Disabled  string `yaml:"disabled,omitempty"`
	X         int    `yaml:"x,omitempty"`
	Y         int    `yaml:"y,omitempty"`
	W         int    `yaml:"w,omitempty"`
	H         int    `yaml:"h,omitempty"`
	ScaleMode string `yaml:"scaleMode,omitempty"`
	Anchor    string `yaml:"anchor,omitempty"`
}

// ManifestValidationError reports one or more schema violations.
type ManifestValidationError struct {
	Issues []string
}

func (e *ManifestValidationError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "manifest validation failed"
	}
	return "manifest validation failed: " + strings.Join(e.Issues, "; ")
}

// LoadVisualManifestFile loads and validates a visual manifest from disk.
func LoadVisualManifestFile(path string) (*VisualManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read visual manifest: %w", err)
	}
	return ParseVisualManifest(data)
}

// ParseVisualManifest decodes and validates a visual manifest document.
func ParseVisualManifest(data []byte) (*VisualManifest, error) {
	if len(data) == 0 {
		return nil, errors.New("visual manifest is empty")
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("decode visual manifest yaml: %w", err)
	}

	duplicates := findDuplicateComponentKeys(&root)

	var manifest VisualManifest
	if err := root.Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode visual manifest structure: %w", err)
	}

	issues := validateVisualManifest(&manifest, duplicates)
	if len(issues) > 0 {
		return nil, &ManifestValidationError{Issues: issues}
	}

	return &manifest, nil
}

func validateVisualManifest(manifest *VisualManifest, duplicates []string) []string {
	issues := make([]string, 0)
	v := manifest.Content.Visuals

	if v.Version != manifestSchemaVersion {
		issues = append(issues, fmt.Sprintf("content.visuals.version must be %d", manifestSchemaVersion))
	}
	if strings.TrimSpace(v.BasePath) == "" {
		issues = append(issues, "content.visuals.basePath is required")
	}
	if strings.TrimSpace(v.Placeholders.Missing) == "" {
		issues = append(issues, "content.visuals.placeholders.missing is required")
	} else if err := validateAssetPath(v.Placeholders.Missing); err != nil {
		issues = append(issues, fmt.Sprintf("content.visuals.placeholders.missing: %v", err))
	}

	if len(v.Components) == 0 {
		issues = append(issues, "content.visuals.components must include at least one entry")
	}

	for _, key := range duplicates {
		issues = append(issues, fmt.Sprintf("content.visuals.components contains duplicate key %q", key))
	}

	for componentID, asset := range v.Components {
		if strings.TrimSpace(componentID) == "" {
			issues = append(issues, "content.visuals.components contains empty component key")
			continue
		}
		if strings.TrimSpace(asset.File) == "" {
			issues = append(issues, fmt.Sprintf("components.%s.file is required", componentID))
		} else if err := validateAssetPath(asset.File); err != nil {
			issues = append(issues, fmt.Sprintf("components.%s.file: %v", componentID, err))
		}

		if err := validateOptionalPath(componentID, "hover", asset.Hover); err != nil {
			issues = append(issues, err.Error())
		}
		if err := validateOptionalPath(componentID, "pressed", asset.Pressed); err != nil {
			issues = append(issues, err.Error())
		}
		if err := validateOptionalPath(componentID, "disabled", asset.Disabled); err != nil {
			issues = append(issues, err.Error())
		}

		if err := validateCropRect(componentID, asset); err != nil {
			issues = append(issues, err.Error())
		}
	}

	return issues
}

func validateCropRect(componentID string, asset ComponentAsset) error {
	// No crop rectangle specified.
	if asset.X == 0 && asset.Y == 0 && asset.W == 0 && asset.H == 0 {
		return nil
	}
	if asset.X < 0 || asset.Y < 0 {
		return fmt.Errorf("components.%s crop x/y must be non-negative", componentID)
	}
	if asset.W <= 0 || asset.H <= 0 {
		return fmt.Errorf("components.%s crop w/h must be positive when crop is specified", componentID)
	}
	return nil
}

func validateOptionalPath(componentID, field, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	if err := validateAssetPath(value); err != nil {
		return fmt.Errorf("components.%s.%s: %v", componentID, field, err)
	}
	return nil
}

func validateAssetPath(path string) error {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return errors.New("path is empty")
	}
	if filepath.IsAbs(trimmed) {
		return errors.New("path must be relative")
	}
	cleaned := filepath.Clean(trimmed)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return errors.New("path traversal is not allowed")
	}
	if strings.ToLower(filepath.Ext(cleaned)) != ".png" {
		return errors.New("only .png assets are supported")
	}
	return nil
}

func findDuplicateComponentKeys(root *yaml.Node) []string {
	duplicates := make(map[string]struct{})

	contentMap := getMappingValue(root, "content")
	visualsMap := getMappingValue(contentMap, "visuals")
	componentsMap := getMappingValue(visualsMap, "components")
	if componentsMap == nil || componentsMap.Kind != yaml.MappingNode {
		return nil
	}

	seen := make(map[string]struct{})
	for i := 0; i+1 < len(componentsMap.Content); i += 2 {
		key := componentsMap.Content[i].Value
		if _, ok := seen[key]; ok {
			duplicates[key] = struct{}{}
			continue
		}
		seen[key] = struct{}{}
	}

	if len(duplicates) == 0 {
		return nil
	}

	out := make([]string, 0, len(duplicates))
	for key := range duplicates {
		out = append(out, key)
	}
	return out
}

func getMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}

	mapping := node
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		mapping = node.Content[0]
	}
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}
