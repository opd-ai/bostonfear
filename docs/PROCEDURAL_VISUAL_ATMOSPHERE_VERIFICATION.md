# Procedural Visual Atmosphere Verification

## Scope
This runbook verifies Workstream 3 outcomes:
- deterministic procedural layers for identical scenario state
- quality tier performance expectations
- theme consistency review workflow

## Seed Determinism Checks
1. Run deterministic unit tests:
   - `go test ./client/ebiten/ui -run TestSeedFromGameState_Deterministic -count=1`
   - `go test ./client/ebiten/ui -run TestProceduralGenerate_Deterministic -count=1`
2. Confirm both tests pass and generated overlays remain stable for identical inputs.

## Frame-Time and Memory Benchmarks by Quality Tier
Run:
- `go test -bench=BenchmarkProceduralGenerate_ -benchmem ./client/ebiten/ui`

Expected trend:
- `Low` tier should allocate and execute less work than `Medium`.
- `Medium` tier should allocate and execute less work than `High`.

## Snapshot Diff Review (Theme Consistency)
1. Start server and desktop client with each quality tier:
   - `BOSTONFEAR_RENDER_QUALITY=low go run ./cmd/desktop -server ws://localhost:8080/ws`
   - `BOSTONFEAR_RENDER_QUALITY=medium go run ./cmd/desktop -server ws://localhost:8080/ws`
   - `BOSTONFEAR_RENDER_QUALITY=high go run ./cmd/desktop -server ws://localhost:8080/ws`
2. Capture equivalent board screenshots for each tier.
3. Validate visual consistency criteria:
   - atmosphere colors stay within the same theme family
   - overlays remain subtle and non-obstructive
   - gameplay labels/resources remain readable

## Notes
- Scenario seed is derived from scenario-identifying wire state fields (difficulty + act/agenda titles + player count), so equivalent scenario state yields equivalent procedural overlays.
- Runtime quality throttling is controlled by `BOSTONFEAR_RENDER_QUALITY`.
