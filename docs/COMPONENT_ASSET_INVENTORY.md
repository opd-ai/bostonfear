# Component-To-Asset Inventory

This document satisfies Phase 1 item "Complete component inventory with owner and priority" from PLAN.md.

## Scope

- Rendering path: Go Ebitengine client only.
- Primary packages: `client/ebiten/render`, `client/ebiten/app`, `client/ebiten/ui`.
- Goal: enumerate visual components that must be mapped to static PNG assets in a YAML-driven pipeline.

## Inventory

| Component ID | Current Visual Source | Owner | Priority | Notes |
|---|---|---|---|---|
| `board.background` | Procedural board fill with atlas fallback tile | Client Rendering (`client/ebiten/render`) | P0 | Core playfield backdrop.
| `location.downtown` | Procedural district background + atlas fallback tile | Client Rendering (`client/ebiten/render`) | P0 | Must remain visually distinct for movement clarity.
| `location.university` | Procedural district background + atlas fallback tile | Client Rendering (`client/ebiten/render`) | P0 | Same requirements as other location tiles.
| `location.rivertown` | Procedural district background + atlas fallback tile | Client Rendering (`client/ebiten/render`) | P0 | Same requirements as other location tiles.
| `location.northside` | Procedural district background + atlas fallback tile | Client Rendering (`client/ebiten/render`) | P0 | Same requirements as other location tiles.
| `token.investigator.default` | Procedural circular token badge with initials | Client Rendering (`client/ebiten/app`) | P0 | Supports multiple players via tint.
| `hud.doom.marker` | Programmatic 12-step doom track | Client Rendering (`client/ebiten/app`) | P0 | Needed for doom bar readability.
| `hud.action.overlay` | `SpriteActionOverlay` atlas tile | Client Rendering (`client/ebiten/render`) | P1 | Overlay treatment can be themed later.
| `hud.turn.indicator` | Text/UI draw path in app layer | Client UI (`client/ebiten/app`) | P0 | Move to explicit icon + text asset pair.
| `hud.resource.health` | Programmatic segmented track in app layer | Client UI (`client/ebiten/app`) | P0 | Requires icon label and numeric legibility checks.
| `hud.resource.sanity` | Programmatic segmented track in app layer | Client UI (`client/ebiten/app`) | P0 | Requires icon label and numeric legibility checks.
| `hud.resource.clues` | Programmatic segmented track in app layer | Client UI (`client/ebiten/app`) | P0 | Requires icon label and numeric legibility checks.
| `ui.results.panel` | Programmatic panel draw in UI layer | Client UI (`client/ebiten/ui`) | P1 | Candidate for themed 9-slice panel PNG.
| `ui.onboarding.panel` | Programmatic overlay in onboarding UI | Client UI (`client/ebiten/ui`) | P1 | Should support wrapped text and icon callouts.
| `ui.action.button.*` | Programmatic/touch action strip in app input scene | Client UI + Input (`client/ebiten/app`) | P0 | Needs state variants: default/hover/pressed/disabled.
| `fx.dice.outcome.*` | Programmatic dice/result feedback in app/UI | Client UI (`client/ebiten/app`, `client/ebiten/ui`) | P1 | Needs explicit success/blank/tentacle icon set.
| `fx.connection.status.*` | WASM host + in-game status text | Web host + Client UI (`client/wasm`, `client/ebiten`) | P2 | Optional iconography for reconnect state.

## Priority Legend

- `P0`: blocks core mechanics clarity or turn execution.
- `P1`: high-value UX improvement, not blocker for basic play.
- `P2`: polish and observability visuals.

## Next Mapping Step

Use this inventory as the source list for YAML component keys in Phase 2. Each row should resolve to:

- `component key`
- `png file path`
- `fallback asset`
- optional metadata (`scaleMode`, anchor, state variants)