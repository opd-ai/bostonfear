package runtime

import (
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/opd-ai/bostonfear/monitoringdata"
	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

// UnimplementedEngine is a placeholder implementation used for game families
// that have package scaffolding but no playable runtime yet.
type UnimplementedEngine struct {
	gameName string
	mu       sync.Mutex
	origins  []string
}

// NewUnimplementedEngine returns a placeholder engine for a game family.
func NewUnimplementedEngine(gameName string) contracts.Engine {
	return &UnimplementedEngine{gameName: gameName}
}

func (e *UnimplementedEngine) Start() error {
	return fmt.Errorf("%s engine not implemented", e.gameName)
}

func (e *UnimplementedEngine) HandleConnection(_ net.Conn, _ string) error {
	return fmt.Errorf("%s engine not implemented", e.gameName)
}

func (e *UnimplementedEngine) SetAllowedOrigins(origins []string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	normalized := make([]string, 0, len(origins))
	for _, origin := range origins {
		if trimmed := strings.TrimSpace(strings.ToLower(origin)); trimmed != "" {
			normalized = append(normalized, trimmed)
		}
	}
	e.origins = normalized
}

func (e *UnimplementedEngine) AllowedOrigins() []string {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.origins) == 0 {
		return nil
	}

	return append([]string(nil), e.origins...)
}

func (e *UnimplementedEngine) SnapshotHealth() monitoringdata.HealthSnapshot {
	return monitoringdata.HealthSnapshot{IsHealthy: false}
}

func (e *UnimplementedEngine) CollectPerformanceMetrics() monitoringdata.PerformanceMetrics {
	return monitoringdata.PerformanceMetrics{}
}

func (e *UnimplementedEngine) CollectConnectionAnalytics() monitoringdata.ConnectionAnalyticsSimplified {
	return monitoringdata.ConnectionAnalyticsSimplified{}
}

func (e *UnimplementedEngine) CollectMemoryMetrics() monitoringdata.MemoryMetrics {
	return monitoringdata.MemoryMetrics{}
}

func (e *UnimplementedEngine) CollectGCMetrics() monitoringdata.GCMetrics {
	return monitoringdata.GCMetrics{}
}

func (e *UnimplementedEngine) CollectMessageThroughput(_ time.Duration) monitoringdata.MessageThroughputMetrics {
	return monitoringdata.MessageThroughputMetrics{}
}

func (e *UnimplementedEngine) GameStatistics() map[string]interface{} {
	return map[string]interface{}{
		"status": "not_implemented",
		"game":   e.gameName,
	}
}
