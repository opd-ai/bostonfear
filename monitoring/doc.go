// Package monitoring provides HTTP handlers that expose game server health and performance metrics.
//
// Naming convention:
//   - monitoring (this package) owns HTTP transport handlers and route-facing
//     concerns for /health and /metrics.
//   - serverengine/common/monitoring owns shared DTO builders and pure helper
//     logic used by handlers and tests.
//
// Keep imports explicit to avoid ambiguity between the two packages.
//
// Endpoints:
//
// - GET /health: JSON snapshot of game health, state corruption history, and alerts
// - GET /metrics: Prometheus-compatible metrics for scraping by monitoring systems
//
// Handler Signatures:
//
// Each handler requires a Provider interface, which the gameengine implements. This decouples
// monitoring concerns from core game logic and enables testing handlers without a full engine.
//
// HealthHandler returns a JSON response with:
//   - status: "healthy", "degraded", or "unhealthy" based on corruption count and engine state
//   - doom counter and game progression
//   - player statistics (count, active, total clues)
//   - recent corruption events (last 5 minutes)
//   - system alerts (high doom, high packet loss, connection churn, etc.)
//
// MetricsHandler compiles server-side metrics into Prometheus format:
//   - latency percentiles (broadcast write time)
//   - connection counts (active, peak, total)
//   - throughput (messages per second)
//   - memory and GC statistics
//   - game statistics (turns played, average session length)
//
// Design Notes:
//
//  1. Thread Safety: All handlers use the Provider interface to acquire read-only snapshots.
//     The game engine holds all locks, so metrics are eventually consistent, not linearizable.
//  2. Performance: Metrics computation is O(n) where n = player count; suitable for 1-6 players.
//  3. Observability: Health checks are independent of game criticality—they cannot block actions.
//  4. Extensibility: New metrics can be added to Provider without changing handler signatures.
package monitoring
