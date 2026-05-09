package serverengine

import (
	"net"
	"time"

	"github.com/opd-ai/bostonfear/monitoringdata"
)

// GameEngine defines the public lifecycle, transport, and monitoring surface
// exposed by the server engine.
type GameEngine interface {
	Start() error
	HandleConnection(conn net.Conn, reconnectToken string) error
	SetAllowedOrigins(origins []string)
	AllowedOrigins() []string

	SnapshotHealth() monitoringdata.HealthSnapshot
	CollectPerformanceMetrics() monitoringdata.PerformanceMetrics
	CollectConnectionAnalytics() monitoringdata.ConnectionAnalyticsSimplified
	CollectMemoryMetrics() monitoringdata.MemoryMetrics
	CollectGCMetrics() monitoringdata.GCMetrics
	CollectMessageThroughput(runtime time.Duration) monitoringdata.MessageThroughputMetrics
	GameStatistics() map[string]interface{}
}

// Compile-time guarantee that GameServer satisfies the exported engine contract.
var _ GameEngine = (*GameServer)(nil)
