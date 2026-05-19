# Asset Generation Guide for BostonFear

This guide explains how to generate game assets for BostonFear using the integrated asset-generator pipeline.

## Table of Contents

- [Quick Start](#quick-start)
- [Setup](#setup)
- [Asset Pipeline Structure](#asset-pipeline-structure)
- [Generating Assets](#generating-assets)
- [Customizing Assets](#customizing-assets)
- [Integration with Game Client](#integration-with-game-client)
- [Troubleshooting](#troubleshooting)

## Quick Start

```bash
# 1. Setup (first time only)
./scripts/setup-asset-gen.sh

# 2. Generate all assets
make assets

# Your assets are now in ./output/
```

## Setup

### Prerequisites

1. **SwarmUI** running locally or remotely
   - Default: `http://localhost:7801`
   - Supports WebSocket-based image generation
   - Compatible with Stable Diffusion XL models

2. **asset-generator CLI** installed
   - Download from: https://github.com/opd-ai/asset-generator/releases
   - Or run setup script: `./scripts/setup-asset-gen.sh`

### First-Time Configuration

Run the setup script to configure your environment:

```bash
./scripts/setup-asset-gen.sh
```

This will:
- Check if asset-generator is installed
- Initialize user configuration
- Prompt for SwarmUI API URL
- Create necessary directories

Manual configuration:

```bash
# Initialize config
asset-generator config init

# Set API endpoint
asset-generator config set api-url http://localhost:7801

# Optional: Set API key if required
asset-generator config set api-key your-api-key
```

## Asset Pipeline Structure

BostonFear uses four main asset pipeline files:

```
assets/
├── investigators.yaml    # Character portraits (512x512)
├── locations.yaml        # Location backgrounds (1024x768)
├── tokens.yaml          # Game tokens (256x256)
└── ui-elements.yaml     # UI components (various sizes)
```

### Pipeline File Format

Each pipeline file defines asset groups with consistent styling:

```yaml
assets:
  - name: Asset Group Name
    output_dir: output/path/
    seed_offset: 1000
    width: 512
    length: 512
    steps: 40
    
    metadata:
      style: "art style description"
      quality: "quality modifiers"
      negative: "things to avoid"
    
    assets:
      - id: unique_id
        name: asset-name
        prompt: "detailed generation prompt"
        filename: "output_file.png"
```

## Generating Assets

### Generate All Assets

```bash
# Using Makefile (recommended)
make assets

# Or directly
./scripts/generate-assets.sh
```

This generates:
- All investigator portraits
- All location backgrounds
- All game tokens
- All UI elements

Output: `./output/` directory

### Generate Specific Asset Types

```bash
# Investigators only
make assets-investigators

# Locations only
make assets-locations

# Tokens only
make assets-tokens

# UI elements only
make assets-ui
```

### Preview Generation (Dry Run)

See what will be generated without creating images:

```bash
make assets-preview

# Or for specific type
asset-generator pipeline --file assets/investigators.yaml --dry-run
```

### Custom Parameters

Override default parameters:

```bash
# Custom seed for reproducibility
BASE_SEED=100 ./scripts/generate-assets.sh

# Custom output directory
OUTPUT_DIR=./game-assets make assets

# Custom generation steps (quality vs. speed)
STEPS=50 ./scripts/generate-assets.sh
```

## Customizing Assets

### Modifying Existing Assets

Edit the pipeline files to customize prompts:

```yaml
# In assets/investigators.yaml
assets:
  - name: Arkham Horror - Core Investigators
    assets:
      - id: inv_roland_banks
        name: roland-banks
        # Customize this prompt
        prompt: "your custom description here"
        filename: "roland_banks.png"
```

Regenerate after changes:

```bash
make assets-investigators
```

### Adding New Assets

1. Open the appropriate pipeline file
2. Add a new asset entry:

```yaml
- id: inv_custom_investigator
  name: custom-investigator
  prompt: "detailed character description, 1920s investigator"
  filename: "custom_investigator.png"
```

3. Regenerate:

```bash
asset-generator pipeline --file assets/investigators.yaml
```

### Adjusting Generation Parameters

Global parameters (affects all assets in group):

```yaml
assets:
  - name: My Asset Group
    width: 512          # Image width
    length: 512         # Image height
    steps: 40           # Generation steps (20-50 typical)
    seed_offset: 1000   # Base seed for reproducibility
```

Per-asset parameters (overrides group settings):

```yaml
assets:
  - id: special_asset
    name: special
    prompt: "prompt here"
    width: 1024         # Override group width
    steps: 50           # More steps for higher quality
    seed: 42            # Explicit seed
```

### Style Metadata

Consistent styling across asset groups:

```yaml
metadata:
  style: "1920s art style, detailed, professional"
  era: "1920s period styling"
  mood: "mysterious and atmospheric"
  quality: "high detail, sharp focus"
  negative: "blurry, low quality, modern elements"
```

Metadata is automatically appended to all asset prompts in the group.

## Integration with Game Client

### Directory Structure

Generated assets should be organized:

```
output/
├── investigators/
│   ├── arkhamhorror/core/
│   ├── eldersign/core/
│   ├── eldritchhorror/core/
│   └── finalhour/core/
├── locations/
│   ├── arkhamhorror/
│   ├── eldersign/
│   └── eldritchhorror/
├── tokens/
│   ├── resources/
│   ├── actions/
│   └── dice/
└── ui/
    ├── buttons/
    ├── panels/
    └── icons/
```

### Copying to Game Client

After generation and review:

```bash
# Copy investigators
cp output/investigators/**/*.png client/ebiten/assets/investigators/

# Copy locations
cp output/locations/**/*.png client/ebiten/assets/locations/

# Copy tokens
cp output/tokens/**/*.png client/ebiten/assets/tokens/

# Copy UI elements
cp output/ui/**/*.png client/ebiten/assets/ui/
```

Or create a deployment script:

```bash
#!/bin/bash
# scripts/deploy-assets.sh
rsync -av --delete output/ client/ebiten/assets/
```

### Loading in Ebitengine

Update sprite loading code to use generated assets:

```go
// client/ebiten/render/atlas.go

// Load investigator portraits
investigatorImage, _, err := ebitenutil.NewImageFromFile(
    "assets/investigators/arkhamhorror/core/detective.png",
)

// Load location backgrounds
locationImage, _, err := ebitenutil.NewImageFromFile(
    "assets/locations/arkhamhorror/downtown.png",
)

// Load game tokens
healthToken, _, err := ebitenutil.NewImageFromFile(
    "assets/tokens/resources/health.png",
)
```

## Advanced Usage

### Reproducible Generation

Use consistent seeds for reproducible results:

```bash
# Always use same seed
BASE_SEED=42 make assets
```

Document the seed in project README for team consistency.

### Post-Processing

Built-in post-processing options:

```bash
# Auto-crop whitespace
asset-generator pipeline --file assets/investigators.yaml --auto-crop

# Downscale to specific width
asset-generator pipeline --file assets/locations.yaml --downscale-width 1024

# Strip metadata
asset-generator pipeline --file assets/tokens.yaml --strip-metadata

# Combine multiple
make assets  # Uses defaults: auto-crop, downscale, strip-metadata
```

### Converting to SVG

For scalable UI elements:

```bash
# Convert PNG to SVG
asset-generator convert svg \
  --input output/ui/icons/icon_settings.png \
  --output output/ui/icons/icon_settings.svg \
  --mode shapes \
  --num-shapes 100
```

### Batch Regeneration

Regenerate specific assets by ID:

```bash
# Find asset by ID in pipeline file
grep -n "inv_roland_banks" assets/investigators.yaml

# Regenerate with same parameters
asset-generator generate image \
  --prompt "male federal agent, stern expression..." \
  --seed 1000 \
  --width 512 \
  --length 512 \
  --steps 40 \
  --save-images \
  --output-dir output/investigators/arkhamhorror/core \
  --filename roland_banks.png
```

## Troubleshooting

### "Connection refused" errors

**Cause**: SwarmUI not running or wrong URL

**Solution**:
```bash
# Check SwarmUI status
curl http://localhost:7801/api/status

# Verify config
asset-generator config get api-url

# Update if needed
asset-generator config set api-url http://localhost:7801
```

### Generated assets don't match expectations

**Cause**: Prompt quality or generation parameters

**Solution**:
1. Review prompts in pipeline file
2. Test with single generation:
   ```bash
   asset-generator generate image \
     --prompt "test prompt here" \
     --steps 40 \
     --save-images
   ```
3. Adjust metadata and prompts
4. Regenerate

### Slow generation

**Cause**: Too many steps or large images

**Solution**:
```bash
# Use fewer steps for faster generation
STEPS=25 make assets-tokens

# Or reduce image size in pipeline file
width: 512
length: 512
steps: 30
```

### Out of disk space

**Solution**:
```bash
# Clean old generated assets
make assets-clean

# Or selective cleanup
rm -rf output/
```

### Pipeline YAML errors

**Cause**: Syntax errors in pipeline files

**Solution**:
```bash
# Validate YAML syntax
yamllint assets/investigators.yaml

# Test with dry run
asset-generator pipeline --file assets/investigators.yaml --dry-run
```

## Best Practices

1. **Use consistent seeds** for reproducible results across team
2. **Preview before generating** with `--dry-run` flag
3. **Start with small batches** to test prompts and quality
4. **Document custom parameters** in pipeline file comments
5. **Version control pipeline files** but not generated assets
6. **Review generated assets** before copying to game client
7. **Use descriptive filenames** for easy identification

## See Also

- [ROADMAP.md](../ROADMAP.md) - Asset generation roadmap and priorities
- [README.md](../README.md) - Project overview
- [asset-generator documentation](https://github.com/opd-ai/asset-generator) - Full tool documentation

## Support

For issues with asset generation:
1. Check this troubleshooting section
2. Review asset-generator documentation
3. Open issue on GitHub: https://github.com/opd-ai/bostonfear/issues
