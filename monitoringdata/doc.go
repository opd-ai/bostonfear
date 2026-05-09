// Package monitoringdata defines data transfer objects (DTOs) for monitoring and observability.
//
// This package is shared between the serverengine and monitoring packages, decoupling metrics
// collection from both game logic and HTTP presentation layers.
//
// Type Categories:
//
// Health Snapshots:
// - HealthSnapshot: Minimal state needed to assemble a health response (lock-bounded subset)
//
// Performance & Throughput:
// - PerformanceMetrics: Comprehensive server metrics (uptime, connections, GC, response time)
// - MessageThroughputMetrics: Message processing (per-second, latency, broadcast time)
// - MemoryMetrics: Detailed memory usage breakdown
// - MemoryStats: Compact memory statistics struct (for PerformanceMetrics)
// - GCMetrics: Garbage collection performance (pause times, frequency, CPU fraction)
//
// Connection Analytics:
// - ConnectionAnalyticsSimplified: Aggregated connection data (reconnection rate, churn)
// - PlayerSessionMetricsSimplified: Per-player session lifetime and action count
// - ConnectionEventSimplified: Timestamped connection events (connect, disconnect, latency)
//
// Integration Points:
//
//  1. serverengine.GameServer implements the monitoring.Provider interface by collecting
//     these types via snapshot methods (SnapshotHealth, CollectPerformanceMetrics, etc.)
//  2. monitoring handlers deserialize these DTOs into JSON responses and HTML templates
//  3. client/ebiten and JavaScript clients can poll /metrics and /health to visualize server state
//
// Design Rationale:
//
// - No engine-specific types: These are pure data structures without behavior
// - JSON tags for marshaling: All types are immediately JSON-serializable for HTTP responses
// - Simplified names: "Simplified" suffix indicates aggregated/sampled data (vs. detailed logs)
// - Lock-free snapshots: Each DTO is immutable after creation; serverengine holds all locks
package monitoringdata
