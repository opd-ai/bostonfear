package monitoring

import (
	"testing"
	"time"

	"github.com/opd-ai/bostonfear/monitoringdata"
)

func TestDeriveHealthStatus(t *testing.T) {
	if got := DeriveHealthStatus(true, 0); got != "healthy" {
		t.Fatalf("healthy status = %q, want healthy", got)
	}
	if got := DeriveHealthStatus(false, 0); got != "degraded" {
		t.Fatalf("degraded status = %q, want degraded", got)
	}
	if got := DeriveHealthStatus(true, 11); got != "unhealthy" {
		t.Fatalf("unhealthy status = %q, want unhealthy", got)
	}
}

func TestCountRecentCorruptions(t *testing.T) {
	now := time.Now()
	history := []time.Time{
		now.Add(-2 * time.Minute),
		now.Add(-10 * time.Minute),
		now.Add(-30 * time.Second),
	}
	if got := CountRecentCorruptions(history, now, 5*time.Minute); got != 2 {
		t.Fatalf("CountRecentCorruptions() = %d, want 2", got)
	}
}

func TestBuildHealthPayloadInvariants(t *testing.T) {
	snapshot := monitoringdata.HealthSnapshot{GamePhase: "playing", IsHealthy: true}
	payload := BuildHealthPayload(
		"healthy",
		snapshot,
		monitoringdata.PerformanceMetrics{},
		monitoringdata.ConnectionAnalyticsSimplified{},
		map[string]interface{}{"k": "v"},
		[]map[string]interface{}{{"severity": "low"}},
		12345,
		0,
	)

	if payload["status"] != "healthy" {
		t.Fatalf("status = %v, want healthy", payload["status"])
	}
	if payload["gamePhase"] != "playing" {
		t.Fatalf("gamePhase = %v, want playing", payload["gamePhase"])
	}
	if payload["timestamp"] != int64(12345) {
		t.Fatalf("timestamp = %v, want 12345", payload["timestamp"])
	}
	if _, ok := payload["systemAlerts"]; !ok {
		t.Fatal("systemAlerts key missing")
	}
}
