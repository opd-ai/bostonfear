#!/bin/bash
# Master asset generation script for BostonFear
# Generates all game assets using asset-generator

set -e

# Configuration
ASSET_GEN="${ASSET_GEN:-asset-generator}"
BASE_SEED="${BASE_SEED:-42}"
OUTPUT_DIR="${OUTPUT_DIR:-./output}"
STEPS="${STEPS:-40}"

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  BostonFear Asset Generation Pipeline ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════╝${NC}"
echo ""
echo "Configuration:"
echo "  Output Dir: $OUTPUT_DIR"
echo "  Base Seed:  $BASE_SEED"
echo "  Steps:      $STEPS"
echo ""

# Check if asset-generator is available
if ! command -v "$ASSET_GEN" &> /dev/null; then
    echo -e "${RED}❌ asset-generator not found${NC}"
    echo "Run: ./scripts/setup-asset-gen.sh"
    exit 1
fi

# Function to generate assets with error handling
generate_pipeline() {
    local pipeline_file=$1
    local pipeline_name=$2
    
    echo -e "${BLUE}🎨 Generating ${pipeline_name}...${NC}"
    
    if $ASSET_GEN pipeline \
        --file "$pipeline_file" \
        --output-dir "$OUTPUT_DIR" \
        --base-seed "$BASE_SEED" \
        --steps "$STEPS" \
        --auto-crop \
        --downscale-width 1024; then
        echo -e "${GREEN}✅ ${pipeline_name} complete${NC}"
        echo ""
    else
        echo -e "${RED}❌ ${pipeline_name} failed${NC}"
        echo ""
        return 1
    fi
}

# Function to count generated files
count_files() {
    local dir=$1
    if [ -d "$dir" ]; then
        find "$dir" -type f -name "*.png" | wc -l
    else
        echo "0"
    fi
}

# Main generation sequence
echo -e "${YELLOW}Starting asset generation...${NC}"
echo ""

START_TIME=$(date +%s)

# Generate each asset type
generate_pipeline "assets/investigators.yaml" "Investigators"
generate_pipeline "assets/locations.yaml" "Locations"
generate_pipeline "assets/tokens.yaml" "Tokens"
generate_pipeline "assets/ui-elements.yaml" "UI Elements"

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

# Summary
echo -e "${GREEN}╔════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Asset Generation Complete!        ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════╝${NC}"
echo ""
echo "Summary:"
echo "  Investigators: $(count_files "$OUTPUT_DIR/investigators") files"
echo "  Locations:     $(count_files "$OUTPUT_DIR/locations") files"
echo "  Tokens:        $(count_files "$OUTPUT_DIR/tokens") files"
echo "  UI Elements:   $(count_files "$OUTPUT_DIR/ui") files"
echo ""
echo "  Total Time:    ${DURATION}s"
echo "  Output:        $OUTPUT_DIR"
echo ""
echo "Next steps:"
echo "  1. Review generated assets in $OUTPUT_DIR"
echo "  2. Copy to client/ebiten/assets/ when ready"
echo "  3. Update sprite loading code to use new assets"
echo ""
