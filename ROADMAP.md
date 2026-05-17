# BostonFear Roadmap

This roadmap tracks implementation work that is planned and committed, ordered by
priority and aligned with the architecture goals in README and AUDIT.

## Phase 1: Close Remaining Scaffold Gaps

### [x] 1. Implement `serverengine/common/messaging`
- Goal: Replace placeholder package with reusable message contracts.
- Deliverables:
  - `MessageCodec` interface for encode/decode boundaries.
  - Shared JSON helper(s) used by at least one production path.
  - Unit tests for successful roundtrip and malformed payload handling.
- Acceptance criteria:
  - `go test ./serverengine/common/messaging/...` passes.
  - Package is imported by production code (not docs/tests only).

### [x] 2. Implement `serverengine/common/session`
- Goal: Introduce session token and lifecycle primitives for reconnect flows.
- Deliverables:
  - `Token` type and validation helper(s).
  - Minimal session store interface used by server/session flow.
  - Tests for token validation and expiration edge cases.
- Acceptance criteria:
  - `go test ./serverengine/common/session/...` passes.
  - Reconnect path uses exported session primitives.

### [x] 3. Implement `serverengine/common/state`
- Goal: Centralize resource bounds/clamping helpers for cross-engine reuse.
- Deliverables:
  - Resource bounds type(s) and clamp/validate function(s).
  - Initial adoption in existing Arkham resource mutation code.
  - Tests covering lower/upper bound enforcement.
- Acceptance criteria:
  - `go test ./serverengine/common/state/...` passes.
  - At least one serverengine path migrates to shared helper(s).

### [x] 4. Implement `serverengine/common/validation`
- Goal: Move reusable action validation checks into a shared package.
- Deliverables:
  - Validator interface(s) or function set for movement/resource checks.
  - Integration in movement or action precheck paths.
  - Tests for accepted and rejected actions.
- Acceptance criteria:
  - `go test ./serverengine/common/validation/...` passes.
  - Existing validation logic is partially migrated without behavior regressions.

### [x] 5. Implement `serverengine/common/observability`
- Goal: Provide shared telemetry contracts independent of concrete backend.
- Deliverables:
  - Event logging/metric recording interfaces.
  - Adapter glue used by monitoring or serverengine code.
  - Tests for nil-safe and no-op behavior.
- Acceptance criteria:
  - `go test ./serverengine/common/observability/...` passes.
  - At least one production metric/event call path uses abstractions.

### [x] 6. Implement `serverengine/common/monitoring`
- Goal: Add shared monitoring DTO/helpers used across game families.
- Deliverables:
  - Shared snapshot/aggregation primitives.
  - Wiring from engine metrics to reusable structures.
  - Tests for stable serialization and field invariants.
- Acceptance criteria:
  - `go test ./serverengine/common/monitoring/...` passes.
  - Package is used by a production monitoring path.

## Phase 2: Multi-Game Family Enablement

### [x] 7. Eldersign module MVP
- Implement functional engine (not `UnimplementedEngine`).
- Register module in server startup registry.

### [x] 8. Eldritchhorror module MVP
- Implement functional engine (not `UnimplementedEngine`).
- Register module in server startup registry.

### [x] 9. Finalhour module MVP
- Implement functional engine (not `UnimplementedEngine`).
- Register module in server startup registry.

## Execution Policy
- Scaffold packages are to be implemented, not deleted, unless explicitly approved.
- Every roadmap item must include production wiring plus tests.
- Do not mark a roadmap item complete if package code exists but remains unused.