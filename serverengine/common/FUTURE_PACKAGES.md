# Reserved Future Packages

This document describes packages that are planned for `serverengine/common` but deferred pending multi-engine architecture maturation.

## Planned Packages

### messaging
**Purpose**: Shared message encoding/decoding contracts for protocol interop between game families.

**Motivation**: When multiple game families are implemented (Elder Sign, Eldritch Horror, Final Hour), common payload shapes will emerge. This package will own the marshaling layer to allow game-family-specific adapters (like `serverengine/arkhamhorror/adapters`) to be swappable.

**Expected exports**:
- `MessageCodec` interface for encode/decode
- `MarshalMessage(msg interface{}) []byte`
- `UnmarshalMessage(data []byte) (interface{}, error)`

### session
**Purpose**: Session lifecycle contracts for player reconnection, token management, and multi-transport session persistence.

**Motivation**: As the client stability requirements harden (mobile reconnection, browser tab suspension), sessions will need to be owned by a cross-engine contract rather than protocol-tier token strings. This package will sit between the transport layer and the game state.

**Expected exports**:
- `Token` type for opaque session identifiers
- `SessionStore` interface for persistence
- `ValidateToken(token Token) error`

### state
**Purpose**: Shared resource bounds, validation, and state mutation helpers for all game families.

**Motivation**: Health, Sanity, Clues, and Doom follow similar validation patterns (1-10 bounds, increment/decrement with overflow guards). This package will centralize the rules to reduce duplication when Elder Sign and other families are added.

**Expected exports**:
- `ResourceBounds` struct
- `IncrementHealthSafe(current, delta int) int`
- `ClampToBounds(value int, bounds ResourceBounds) int`

### validation
**Purpose**: Action validation helpers shared across game engines.

**Motivation**: Movement adjacency, action-ordering, and cost validation are patterns that repeat. Centralizing checkers here enables consistent rule enforcement across families.

**Expected exports**:
- `ActionValidator` interface
- `ValidateLocationMovement(from, to string) error`
- `ValidateResourceCost(current, cost int, bounds ResourceBounds) error`

### observability
**Purpose**: Telemetry and structured logging contracts for game events without coupling to specific metric backends.

**Motivation**: As monitoring matures, abstracting the telemetry layer allows swapping Prometheus for OpenTelemetry or other sinks without changing game code.

**Expected exports**:
- `EventLogger` interface
- `LogGameEvent(event string, fields map[string]interface{})`
- `RecordMetric(metric string, value float64)`

---

## Creation Trigger

Create one of these packages and implement its baseline interface when:
1. A second game family (Elder Sign, Eldritch Horror, or Final Hour) is migrated from scaffold to functional.
2. The new family shares 2+ contracts with Arkham (e.g., movement rules, resource bounds).
3. A blocking issue in the migration roadmap requires this abstraction.

Until then, keeping these packages unimplemented reduces noise and avoids premature abstraction.
