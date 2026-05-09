# ROADMAP

This roadmap tracks near-term priorities for BostonFear as a rules-only multiplayer
Arkham Horror engine with Go server and Go/Ebitengine clients.

## Current State

- Arkham Horror gameplay path is the production-ready runtime.
- Elder Sign, Eldritch Horror, and Final Hour modules remain scaffolds.
- Server/client protocol and reconnection workflows are stable for active development.

## Priority 1: Runtime Reliability

### 1.1 Enhanced Reconnection and Session Resilience
- Maintain reconnect token reclaim behavior for dropped clients.
- Improve stale-session cleanup and connection quality telemetry.
- Preserve deterministic turn order through reconnect/disconnect events.

### 1.2 Broadcast and Turn Pipeline Hardening
- Keep game-state broadcasts race-safe under load.
- Continue reducing facade ownership where module ownership is proven.
- Preserve protocol compatibility for gameState, gameUpdate, diceResult, and connectionStatus.

## Priority 2: Arkham Rules Decomposition

- Continue migrating logic into serverengine/arkhamhorror packages with parity tests.
- Keep public serverengine API stable while internals are modularized.
- Move only tested vertical slices; avoid broad rewrites.

## Priority 3: Cross-Family Runtime Enablement

- Add minimum runnable engines for eldersign, eldritchhorror, and finalhour.
- Register non-Arkham families in production paths only after runnable server loops exist.
- Reuse common/runtime contracts without importing serverengine facade internals.

## Priority 4: Mobile and Web Delivery

- Keep desktop and WASM clients buildable with placeholder visual assets.
- Preserve touch-input parity for mobile bindings.
- Keep client/server compatibility while UI implementation matures.

## Priority 5: Performance and UX Targets

- Target >= 60 FPS on desktop clients.
- Maintain state synchronization latency under 500 ms for normal multiplayer usage.
- Keep observability endpoints available for profiling and soak-test analysis.

## Non-Goals

- Shipping copyrighted Arkham Horror narrative/card content.
- Replacing Go/Ebitengine clients with framework-based web clients.
- Introducing breaking protocol changes without migration planning.

## Working References

- docs/MODULE_MIGRATION_MAP.md
- docs/MODULE_MIGRATION_COMPLETION.md
- README.md
