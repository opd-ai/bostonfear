# Implementation Gaps — 2026-06-01 (updated)

> This file covers gaps between the project's stated goals (README, RULES.md,
> CLIENT_SPEC.md, ROADMAP.md) and the current implementation, ordered by severity.
>
> **Previous-cycle gaps resolved since the last report:**
> GAP-11 (handleConnection unbalanced RUnlock), GAP-12 (performComponent always
> errors), GAP-13 (win condition player-count scaling), GAP-14 (JS localStorage
> token), GAP-15 (gs.connections wrong mutex), GAP-16 (app/render test files),
> GAP-17 (/health+/metrics nested RLock deadlock), GAP-18 (gs.gameState.Doom
> unlocked read), GAP-19 (win threshold not rescaled on late join),
> GAP-20 (ActionComponent dead code — promoted to live feature),
> GAP-21 (Ebitengine tests skipped — documented in README; display-independent
> tests blocked by GLFW init() panic, tracked as open below).
>
> **No actionable HIGH or MEDIUM gaps remain.** One LOW-severity documentation gap
> (GAP-21) persists. See summary table below.

---

## ✅ RESOLVED: GAP-17 — Latent Deadlock in `/health` and `/metrics`

- **Resolution**: `handleHealthCheck` and `handleMetrics` now snapshot all game
  state fields under a single `gs.mutex.RLock()` block, release the lock, then
  pass pre-snapshotted values to helpers. No nested RLock calls remain.
- **Test coverage**: `TestHandleHealthCheck_ConcurrentActions` (4 writer goroutines
  + 50 concurrent health check requests; race detector finds no deadlock).
- **Resolved in**: `cmd/server/health.go` (refactor), `cmd/server/observability_test.go` (test)

---

## ✅ RESOLVED: GAP-18 — `gs.gameState.Doom` Read Without Lock in Alert Messages

- **Resolution**: `getSystemAlerts` captures `doom` inside the `gs.mutex.RLock()`
  block and uses the local copy in all `fmt.Sprintf` calls. No post-unlock reads
  of `gs.gameState.Doom` remain.
- **Resolved in**: `cmd/server/health.go`

---

## ✅ RESOLVED: GAP-19 — Win Threshold Not Rescaled When Player Joins Mid-Game

- **Resolution**: `registerPlayer` late-join branch (game already started, phase
  "playing") calls `rescaleActDeck(len(gs.gameState.Players))` after adding the
  new player to the turn order.
- **Test coverage**: `TestRescaleActDeck_LateJoin` (2P: `ActDeck[2].ClueThreshold == 8`),
  `TestRescaleActDeck_LateJoin_ThreePlayers` (3P: `ActDeck[2].ClueThreshold == 12`).
- **Resolved in**: `cmd/server/connection.go`, `cmd/server/connection_test.go`

---

## ✅ RESOLVED: GAP-20 — `ActionComponent` Promoted to Live Feature

- **Resolution**: Rather than removing the dead code, `performComponent` was fully
  implemented with `DefaultInvestigatorAbilities` (6 investigator archetypes, each
  with a distinct ability effect). `ActionComponent` is now added to `isValidActionType()`
  and is reachable.
- **Test coverage**: `TestProcessAction_Component` (all 6 archetypes), `TestProcessAction_ComponentActionAccepted`.
- **Resolved in**: `cmd/server/game_mechanics.go`, `cmd/server/game_constants.go`,
  `cmd/server/game_server.go`, `cmd/server/component_test.go`

---

## GAP-21: Ebitengine `app` and `render` Tests Silently Skipped in Standard CI

- **Stated Goal**: README §Ebitengine Client Features — "Sprite/Layer Rendering:
  Board, tokens, UI overlays, and animations via Ebitengine draw layers." The
  Quick Setup guide instructs contributors to run `go test ./...` with no mention
  of build tags.
- **Current State**: `client/ebiten/app/game_test.go:9` and
  `client/ebiten/render/atlas_test.go:9` carry `//go:build requires_display`.
  Running `go test ./...` (the only command documented in the README) outputs
  `[no test files]` for both packages, silently skipping all rendering regression
  tests.

  Extracting display-independent test logic into tag-free files is **blocked** by
  `github.com/hajimehoshi/ebiten/v2`'s `internal/ui.init.0()` which calls GLFW
  and panics with `"X11: The DISPLAY environment variable is missing"` before any
  test function can run — even in a `_test.go` file that does not itself call any
  Ebitengine API. This is an `init()` side-effect; it fires before `TestMain` and
  cannot be recovered.

- **Impact** (LOW): Rendering regressions in `drawPlayerPanel`, atlas initialisation
  panics, and shader compilation failures are not caught in standard `go test ./...`
  runs. The `app` package has the highest coupling score in the project (5
  dependencies).
- **Mitigation already in place**: README now documents `make test-display` (which
  uses Xvfb) for display-requiring tests. Contributors are informed.
- **Remaining path to closure**:
  Refactor `client/ebiten/app` and `client/ebiten/render` to separate
  Ebitengine-API-calling code from pure logic behind build tags, so the pure logic
  is importable without triggering the GLFW `init()`. This requires a package
  restructure (e.g., `client/ebiten/app/logic` with no ebiten import) and is a
  non-trivial refactor.

---

## Summary Table

| Gap ID | Area | Severity | Status |
|--------|------|----------|--------|
| GAP-17 | `/health`/`/metrics` nested RLock deadlock | HIGH | ✅ Resolved |
| GAP-18 | `gs.gameState.Doom` unlocked read in alerts | HIGH | ✅ Resolved |
| GAP-19 | Win threshold not rescaled on mid-game join | MEDIUM | ✅ Resolved |
| GAP-20 | `ActionComponent` dead code in dispatch switch | LOW | ✅ Resolved (promoted to live feature) |
| GAP-21 | Ebitengine `app`/`render` tests skip in standard CI | LOW | ⚠️ Open — blocked by GLFW init() |
