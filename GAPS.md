# Implementation Gaps — 2026-05-09

## Arkham Runtime Ownership Not Yet Migrated
- **Intended Behavior**: Arkham logic should live in `serverengine/arkhamhorror/*` while `serverengine` becomes a compatibility facade.
- **Current State**: `docs/MODULE_MIGRATION_MAP.md:27` shows all slices (`S1`-`S6`) still `Planned`; `serverengine/arkhamhorror/module.go:66` still delegates to `serverengine.NewGameServer()`.
- **Blocked Goal**: The package-responsibility architecture documented in `README.md` is not yet realized.
- **Implementation Path**: Move one vertical slice at a time (start with `S1` action dispatch), introduce module-owned implementations behind internal interfaces, switch facade call sites, then progress through phases/rules/model/scenarios/adapters.
- **Dependencies**: Requires parity tests for each moved slice before cutover.
- **Effort**: large

## `scenario.default_id` Not Connected to Startup
- **Intended Behavior**: Startup should honor `[scenario].default_id` and the documented fallback chain.
- **Current State**: The key exists in `config.toml:41` and is documented in `README.md:145`, but runtime path uses `NewGameServer()` -> `DefaultScenario` (`serverengine/game_server.go:111`) without config-driven scenario resolution.
- **Blocked Goal**: Configurable default scenario selection and content-loader fallback contract are not operational.
- **Implementation Path**: Read `scenario.default_id` from Viper in server startup, resolve against content index (`serverengine/arkhamhorror/content/nightglass/scenarios/index.yaml`), pass resolved scenario into constructor.
- **Dependencies**: Prefer implementation alongside migration slice `S5` (scenario/content ownership).
- **Effort**: medium

## `web.server` Config Key Is Inert
- **Intended Behavior**: WASM client command should support explicit endpoint override via `web.server` while retaining browser-origin fallback.
- **Current State**: `cmd/web_wasm.go:43` resolves URL only from JavaScript globals/location; no Viper binding/read for `web.server` despite documentation (`README.md:147`, `config.toml:25`).
- **Blocked Goal**: Operator-configured WASM endpoint override cannot be used.
- **Implementation Path**: Add `web.server` binding, resolve configured value first, then fallback to `__serverURL` and browser-origin logic.
- **Dependencies**: None.
- **Effort**: small

## Placeholder Engines Exposed as First-Class Runtime Choices
- **Intended Behavior**: Module registry should expose runnable engines or reject placeholders before startup.
- **Current State**: `cmd/server.go:58-60` registers `eldersign`, `eldritchhorror`, `finalhour`, but each module returns `runtime.NewUnimplementedEngine(...)` (`serverengine/eldersign/module.go:43`, `serverengine/eldritchhorror/module.go:45`, `serverengine/finalhour/module.go:44`).
- **Blocked Goal**: Multi-engine selection appears supported but three options terminate as not implemented at runtime.
- **Implementation Path**: Gate placeholder registration behind an explicit experimental switch or fail during module resolution with a clear unsupported-module error.
- **Dependencies**: None.
- **Effort**: small

## Common Runtime Subpackages Are Structural Scaffolds Only
- **Intended Behavior**: `serverengine/common/*` should host active cross-game abstractions.
- **Current State**: Several packages are doc-only with deferred implementation notes (`serverengine/common/messaging/doc.go:5`, `serverengine/common/session/doc.go:5`, `serverengine/common/state/doc.go:5`, `serverengine/common/observability/doc.go:5`, `serverengine/common/monitoring/doc.go:5`, `serverengine/common/validation/doc.go:5`).
- **Blocked Goal**: Shared-runtime modularization is signaled in structure but not yet realized in executable code.
- **Implementation Path**: Either remove these packages until first extraction, or move one concrete primitive into each package as migration slices progress.
- **Dependencies**: Tied to Arkham migration backlog (`docs/MODULE_MIGRATION_MAP.md`).
- **Effort**: medium

## `UnimplementedEngine` Contract Semantics Are Partial
- **Intended Behavior**: `contracts.Engine` methods should preserve coherent semantics even in placeholder implementations.
- **Current State**: `SetAllowedOrigins` is a no-op and `AllowedOrigins` always returns `nil` in placeholder engine (`serverengine/common/runtime/unimplemented_engine.go:31`, `serverengine/common/runtime/unimplemented_engine.go:33`).
- **Blocked Goal**: Interface behavior is inconsistent across real and placeholder engine implementations.
- **Implementation Path**: Persist origins in placeholder engine state or explicitly document unsupported semantics at contract level and enforce via tests.
- **Dependencies**: None.
- **Effort**: small
