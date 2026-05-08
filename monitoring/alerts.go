package monitoring

import (
	"fmt"

	"github.com/opd-ai/bostonfear/monitoringdata"
)

// BuildSystemAlerts derives operational alerts from monitoring snapshots.
func BuildSystemAlerts(performance monitoringdata.PerformanceMetrics, doom int) []map[string]interface{} {
	alerts := []map[string]interface{}{}

	if performance.MemoryUsage.AllocMB > 100 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "warning",
			"message":  fmt.Sprintf("High memory usage: %.1f MB", performance.MemoryUsage.AllocMB),
			"severity": "medium",
		})
	}

	if performance.ResponseTimeMs > 100 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "warning",
			"message":  fmt.Sprintf("High response time: %.1f ms", performance.ResponseTimeMs),
			"severity": "medium",
		})
	}

	if performance.ErrorRate > 5 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "error",
			"message":  fmt.Sprintf("High error rate: %.1f%%", performance.ErrorRate),
			"severity": "high",
		})
	}

	doomPercent := float64(doom) / 12.0 * 100
	if doomPercent > 80 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "error",
			"message":  fmt.Sprintf("Critical doom level: %d/12 (%.0f%%)", doom, doomPercent),
			"severity": "critical",
		})
	} else if doomPercent > 60 {
		alerts = append(alerts, map[string]interface{}{
			"type":     "warning",
			"message":  fmt.Sprintf("High doom level: %d/12 (%.0f%%)", doom, doomPercent),
			"severity": "medium",
		})
	}

	return alerts
}
