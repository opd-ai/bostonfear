# Component Asset Coverage Report

This report satisfies PLAN.md Phase 3 checklist item:
"Coverage report confirms manifest key usage for all components."

## Source of Truth

- Manifest file: `client/ebiten/render/assets/visuals.yaml`
- Runtime mapping: `client/ebiten/render/asset_resolver.go` (`requiredComponentKeys`)
- Enforcement test: `client/ebiten/render/asset_resolver_manifest_test.go`

## Required Coverage Matrix

| SpriteID | Required Manifest Key | Present in Manifest |
|---|---|---|
| `SpriteBackground` | `board.background` | Yes |
| `SpriteLocationDowntown` | `location.downtown` | Yes |
| `SpriteLocationUniversity` | `location.university` | Yes |
| `SpriteLocationRivertown` | `location.rivertown` | Yes |
| `SpriteLocationNorthside` | `location.northside` | Yes |
| `SpritePlayerToken` | `token.investigator.default` | Yes |
| `SpriteDoomMarker` | `hud.doom.marker` | Yes |
| `SpriteActionOverlay` | `ui.action.button.default` | Yes |

## Verification

Run:

```bash
go test ./client/ebiten/render -run Embedded
```

Passing tests confirm that every required sprite mapping key exists in the manifest
and that the embedded atlas resolver can build a valid atlas from YAML references.
