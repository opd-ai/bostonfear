# Hardcoded Asset References

This document satisfies Phase 1 item "All current hardcoded asset references identified" from PLAN.md.

## Scope

- Focus: currently hardcoded visual asset references in the active Go Ebitengine client and web host path.
- Includes: embedded sprite and shader assets, fixed atlas coordinates, and hardcoded visual fallback values.
- Excludes: server scenario/content embeds not used as client visual assets.

## Identified Hardcoded References

| File | Reference Type | Hardcoded Value | Current Use |
|---|---|---|---|
| `client/ebiten/render/atlas.go` | Embedded PNG path | `//go:embed assets/sprites.png` | Primary sprite sheet for board, locations, token, doom marker, action overlay.
| `client/ebiten/render/atlas.go` | Fixed atlas tile coordinates | `spriteCoords` table (`64x64` tiles on `512x512`) | Maps `SpriteID` values to exact sub-image rectangles.
| `client/ebiten/render/atlas.go` | Fixed placeholder atlas dimensions | `tileSize=64`, `cols=8`, image `512x512` | Fallback texture generation when PNG decode fails.
| `client/ebiten/render/atlas.go` | Fixed placeholder colors | `placeholderColours` RGBA table | Development-safe visual fallback per sprite role.
| `client/ebiten/render/layers.go` | Location-to-sprite mapping literals | `Downtown`, `University`, `Rivertown`, `Northside` | Selects location sprite in `LocationSpriteID`.
| `client/ebiten/render/shaders.go` | Embedded shader asset paths | `//go:embed shaders/fog.kage`, `shaders/glow.kage`, `shaders/doom.kage` | Post-processing shader sources compiled at startup.
| `client/wasm/index.html` | WASM runtime script path | `wasm_exec.js` | Browser runtime bootstrap script.
| `client/wasm/index.html` | WASM binary path | `game.wasm` | Browser-loaded compiled game artifact.

## Migration Notes For YAML-Driven Assets

1. `assets/sprites.png` and `spriteCoords` should be replaced by per-component YAML keys with file paths and optional atlas metadata.
2. `LocationSpriteID` string literals should map through a config-backed resolver keyed by location ID.
3. Placeholder fallback values should move into config defaults where practical, while retaining code-level safe fallback behavior.
4. Shader embeds can remain code-embedded for now; they are effect assets, not static PNG art. If desired, they can be externalized in a later phase.