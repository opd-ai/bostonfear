#!/bin/bash
# Setup script for asset-generator integration
# Helps new team members configure asset generation

set -e

echo "🔧 Setting up asset-generator for BostonFear..."
echo ""

# Check if asset-generator is installed
if ! command -v asset-generator &> /dev/null; then
    echo "❌ asset-generator not found"
    echo ""
    echo "📥 Installation options:"
    echo ""
    echo "  Linux (amd64):"
    echo "    curl -sSL https://github.com/opd-ai/asset-generator/releases/latest/download/asset-generator-linux-amd64 -o asset-generator"
    echo "    chmod +x asset-generator"
    echo "    sudo mv asset-generator /usr/local/bin/"
    echo ""
    echo "  macOS (amd64):"
    echo "    curl -sSL https://github.com/opd-ai/asset-generator/releases/latest/download/asset-generator-darwin-amd64 -o asset-generator"
    echo "    chmod +x asset-generator"
    echo "    sudo mv asset-generator /usr/local/bin/"
    echo ""
    echo "  macOS (arm64 / Apple Silicon):"
    echo "    curl -sSL https://github.com/opd-ai/asset-generator/releases/latest/download/asset-generator-darwin-arm64 -o asset-generator"
    echo "    chmod +x asset-generator"
    echo "    sudo mv asset-generator /usr/local/bin/"
    echo ""
    echo "  Or download from: https://github.com/opd-ai/asset-generator/releases"
    exit 1
fi

echo "✅ asset-generator found: $(asset-generator --version)"
echo ""

# Initialize user config if it doesn't exist
if [ ! -f ~/.asset-generator/config.yaml ]; then
    echo "📝 Initializing user configuration..."
    asset-generator config init
    echo ""
fi

# Prompt for API URL
echo "🌐 SwarmUI API Configuration"
echo "------------------------------"
read -p "Enter SwarmUI API URL [http://localhost:7801]: " api_url
api_url=${api_url:-http://localhost:7801}
asset-generator config set api-url "$api_url"
echo ""

# Optional API key
read -p "Enter API key (leave blank if none): " api_key
if [ -n "$api_key" ]; then
    asset-generator config set api-key "$api_key"
    echo ""
fi

# Create output directory
mkdir -p output

echo "✅ Setup complete!"
echo ""
echo "📋 Next steps:"
echo "   1. Ensure SwarmUI is running at: $api_url"
echo "   2. Preview what will be generated:"
echo "      make assets-preview"
echo "   3. Generate all assets:"
echo "      make assets"
echo "   4. Or generate specific asset types:"
echo "      make assets-investigators"
echo "      make assets-locations"
echo "      make assets-tokens"
echo "      make assets-ui"
echo ""
echo "📖 For more information, see docs/ASSET_GENERATION.md"
