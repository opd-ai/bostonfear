# Makefile — build, test, and lint targets for the bostonfear project.

.PHONY: all build test test-display vet clean rebuild-wasm
.PHONY: assets assets-preview assets-investigators assets-locations assets-tokens assets-ui assets-clean assets-deploy

# Configuration
ASSET_GEN := asset-generator
ASSET_DIR := assets
OUTPUT_DIR := output

## all: build the server and all clients.
all: build

## build: compile all packages.
build: rebuild-wasm
	go build ./...

## test: run the standard test suite (no display required; CI-safe).
test:
	go test -race ./...

## test-display: run Ebitengine tests that require a local or virtual display.
## Set DISPLAY before calling if no physical display is available:
##   Xvfb :99 -screen 0 1024x768x24 & DISPLAY=:99 make test-display
test-display:
	go test -race -tags=requires_display ./client/ebiten/app/... ./client/ebiten/render/...

## vet: run go vet across all packages.
vet:
	go vet ./...

## clean: remove build artifacts.
clean:
	go clean ./...

## rebuild-wasm: force rebuild the browser WASM binary.
rebuild-wasm:
	rm -f client/wasm/game.wasm
	GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web

# ============================================================================
# Asset Generation Targets
# ============================================================================

## assets: Generate all game assets (investigators, locations, tokens, UI)
assets: assets-investigators assets-locations assets-tokens assets-ui
	@echo "✅ All assets generated in $(OUTPUT_DIR)/"

## assets-preview: Preview what assets would be generated (dry run)
assets-preview:
	@echo "📋 Preview: Investigators"
	@$(ASSET_GEN) pipeline --file $(ASSET_DIR)/investigators.yaml --dry-run
	@echo "\n📋 Preview: Locations"
	@$(ASSET_GEN) pipeline --file $(ASSET_DIR)/locations.yaml --dry-run
	@echo "\n📋 Preview: Tokens"
	@$(ASSET_GEN) pipeline --file $(ASSET_DIR)/tokens.yaml --dry-run
	@echo "\n📋 Preview: UI Elements"
	@$(ASSET_GEN) pipeline --file $(ASSET_DIR)/ui-elements.yaml --dry-run

## assets-investigators: Generate investigator portrait sprites
assets-investigators:
	@echo "🎨 Generating investigator portraits..."
	@$(ASSET_GEN) pipeline --file $(ASSET_DIR)/investigators.yaml \
		--output-dir $(OUTPUT_DIR) \
		--auto-crop \
		--downscale-width 512
	@echo "✅ Investigators generated"

## assets-locations: Generate location background sprites
assets-locations:
	@echo "🎨 Generating location backgrounds..."
	@$(ASSET_GEN) pipeline --file $(ASSET_DIR)/locations.yaml \
		--output-dir $(OUTPUT_DIR) \
		--auto-crop \
		--downscale-width 1024
	@echo "✅ Locations generated"

## assets-tokens: Generate game token sprites
assets-tokens:
	@echo "🎨 Generating game tokens..."
	@$(ASSET_GEN) pipeline --file $(ASSET_DIR)/tokens.yaml \
		--output-dir $(OUTPUT_DIR) \
		--auto-crop \
		--downscale-width 256
	@echo "✅ Tokens generated"

## assets-ui: Generate UI element sprites
assets-ui:
	@echo "🎨 Generating UI elements..."
	@$(ASSET_GEN) pipeline --file $(ASSET_DIR)/ui-elements.yaml \
		--output-dir $(OUTPUT_DIR) \
		--auto-crop
	@echo "✅ UI elements generated"

## assets-clean: Remove all generated assets
assets-clean:
	@echo "🧹 Cleaning generated assets..."
	@rm -rf $(OUTPUT_DIR)
	@mkdir -p $(OUTPUT_DIR)
	@echo "✅ Assets cleaned"

## assets-deploy: Copy generated assets to client directory
assets-deploy:
	@echo "📦 Deploying assets to client..."
	@mkdir -p client/ebiten/assets/investigators
	@mkdir -p client/ebiten/assets/locations
	@mkdir -p client/ebiten/assets/tokens
	@mkdir -p client/ebiten/assets/ui
	@find $(OUTPUT_DIR)/investigators -name "*.png" -exec cp {} client/ebiten/assets/investigators/ \;
	@find $(OUTPUT_DIR)/locations -name "*.png" -exec cp {} client/ebiten/assets/locations/ \;
	@find $(OUTPUT_DIR)/tokens -name "*.png" -exec cp {} client/ebiten/assets/tokens/ \;
	@find $(OUTPUT_DIR)/ui -name "*.png" -exec cp {} client/ebiten/assets/ui/ \;
	@echo "✅ Assets deployed to client/ebiten/assets/"

## check-asset-gen: Verify asset-generator is installed
check-asset-gen:
	@command -v $(ASSET_GEN) >/dev/null 2>&1 || \
		(echo "❌ asset-generator not found. Run: ./scripts/setup-asset-gen.sh" && exit 1)
	@echo "✅ asset-generator found: $$($(ASSET_GEN) --version)"
