package contracts

import (
	"net"
	"time"

	"github.com/opd-ai/bostonfear/monitoringdata"
)

// Engine defines the transport-neutral runtime surface required by server startup,
// transport adapters, and monitoring handlers.
type Engine interface {
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
