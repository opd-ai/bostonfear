# UI Token and Accessibility Verification

## Scope
Verifies Workstream 5 outcomes:
- design token adoption in primary HUD surfaces
- contrast baseline checks
- visual regression snapshot process

## Automated Checks
Run:
- `go test ./client/ebiten/ui -run TestTokenRegistryRequiredEntries -count=1`
- `go test ./client/ebiten/ui -run TestThemeContrastBaseline -count=1`

These tests serve as token usage lint and accessibility contrast baseline checks.

## Manual Visual Regression
1. Start server and client:
   - `go run . server`
   - `go run ./cmd/desktop -server ws://localhost:8080/ws`
2. Capture screenshots for:
   - connect scene
   - character select
   - in-game HUD with at least 2 players
3. Validate:
   - token-derived colors are applied consistently to doom/resource regions
   - labels remain readable
   - status cues remain understandable without relying on color only

## Notes
- Shared icon semantics are provided by `ui.IconRegistry` and used in HUD labels.
- Shared motion presets are provided by `ui.MotionCatalog` for consistent transition timing.
