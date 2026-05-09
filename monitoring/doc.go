// Package monitoring provides HTTP handlers that expose game server health, performance metrics,
// system diagnostics, and an interactive performance dashboard.
//
// Endpoints:
//
// - GET /health: JSON snapshot of game health, state corruption history, and alerts
// - GET /metrics: Prometheus-compatible metrics for scraping by monitoring systems
// - GET /dashboard: Interactive HTML5 dashboard displaying real-time performance and player state
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
// DashboardHandler serves an HTML5 single-page app that polls /health and /metrics periodically,
// rendering:
//   - Real-time doom counter and game phase indicator
//   - Per-player resource graphs (health, sanity, clues) with color-coded status
//   - Connection quality heatmap (latency, packet loss per player)
//   - Broadcast latency trend (rolling 100-sample window)
//   - Alert panel with system warnings and corruption history
//
// Design Notes:
//
//  1. Thread Safety: All handlers use the Provider interface to acquire read-only snapshots.
//     The game engine holds all locks, so metrics are eventually consistent, not linearizable.
//  2. Performance: Metrics computation is O(n) where n = player count; suitable for 1-6 players.
//  3. Observability: Health checks are independent of game criticality—they cannot block actions.
//  4. Extensibility: New metrics can be added to Provider without changing handler signatures.
package monitoring
