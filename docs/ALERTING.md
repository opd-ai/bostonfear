# Alerting Rules for BostonFear Arkham Horror Game Server

This document provides sample Prometheus alerting rules and Grafana dashboard templates for monitoring production deployments of the BostonFear game server.

## Overview

The BostonFear server exposes Prometheus metrics at `/metrics` and health status at `/health`. This document shows how to configure alerts for critical operational conditions.

## Prometheus Alerting Rules

### Sample alert.rules.yml

Create this file and reference it in your Prometheus configuration:

```yaml
groups:
  - name: bostonfear_game_server
    interval: 30s
    rules:
      # Broadcast Latency Alert
      - alert: HighBroadcastLatency
        expr: arkham_horror_broadcast_latency_ms > 200
        for: 2m
        labels:
          severity: warning
          component: game_server
        annotations:
          summary: "High broadcast latency detected"
          description: "Broadcast latency is {{ $value }}ms, exceeding the 200ms threshold. Game state synchronization may be delayed."
          
      - alert: CriticalBroadcastLatency
        expr: arkham_horror_broadcast_latency_ms > 500
        for: 1m
        labels:
          severity: critical
          component: game_server
        annotations:
          summary: "Critical broadcast latency detected"
          description: "Broadcast latency is {{ $value }}ms, severely impacting player experience."

      # Error Rate Alert
      - alert: HighErrorRate
        expr: rate(arkham_horror_error_count[5m]) > 0.05
        for: 3m
        labels:
          severity: warning
          component: game_server
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }}, exceeding 5% threshold."
          
      - alert: CriticalErrorRate
        expr: rate(arkham_horror_error_count[5m]) > 0.20
        for: 1m
        labels:
          severity: critical
          component: game_server
        annotations:
          summary: "Critical error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }}, indicating severe issues."

      # Doom Counter Alert
      - alert: DoomCounterCritical
        expr: arkham_horror_game_doom_level >= 11
        for: 30s
        labels:
          severity: warning
          component: game_logic
        annotations:
          summary: "Doom counter approaching maximum"
          description: "Doom level is {{ $value }} (max 12). Game may end soon."
          
      - alert: DoomCounterMaxReached
        expr: arkham_horror_game_doom_level >= 12
        for: 10s
        labels:
          severity: info
          component: game_logic
        annotations:
          summary: "Doom counter reached maximum"
          description: "Doom level has reached 12. Game should have ended."

      # Connection Stability Alert
      - alert: NoActiveConnections
        expr: arkham_horror_active_connections == 0
        for: 5m
        labels:
          severity: info
          component: connectivity
        annotations:
          summary: "No active player connections"
          description: "Server has no active connections for 5 minutes."
          
      - alert: HighConnectionDropRate
        expr: rate(arkham_horror_disconnections[5m]) > 0.1
        for: 3m
        labels:
          severity: warning
          component: connectivity
        annotations:
          summary: "High connection drop rate"
          description: "Connection drop rate is {{ $value }}/s, indicating network or server issues."

      # Memory Usage Alert
      - alert: HighMemoryUsage
        expr: arkham_horror_memory_usage_percent > 80
        for: 5m
        labels:
          severity: warning
          component: resources
        annotations:
          summary: "High memory usage detected"
          description: "Memory usage is {{ $value }}%, approaching resource limits."
          
      - alert: CriticalMemoryUsage
        expr: arkham_horror_memory_usage_percent > 95
        for: 1m
        labels:
          severity: critical
          component: resources
        annotations:
          summary: "Critical memory usage"
          description: "Memory usage is {{ $value }}%, server may become unstable."

      # Health Check Alert
      - alert: HealthCheckFailing
        expr: up{job="bostonfear"} == 0
        for: 1m
        labels:
          severity: critical
          component: availability
        annotations:
          summary: "BostonFear server is down"
          description: "Health check endpoint is not responding. Server may be down or unresponsive."

      # Action Processing Rate Alert
      - alert: LowActionProcessingRate
        expr: rate(arkham_horror_actions_processed[5m]) < 0.01 and arkham_horror_active_connections > 0
        for: 5m
        labels:
          severity: warning
          component: game_logic
        annotations:
          summary: "Low action processing rate with active players"
          description: "Actions are being processed at {{ $value }}/s with {{ $labels.active_connections }} active players. May indicate stalled game state."
```

## Critical Thresholds

| Metric | Warning | Critical | Impact |
|--------|---------|----------|--------|
| Broadcast Latency | >200ms | >500ms | Players experience delayed state updates |
| Error Rate | >5% | >20% | Frequent action failures or server instability |
| Doom Level | ≥11 | ≥12 | Game approaching or at end condition |
| Memory Usage | >80% | >95% | Server may run out of memory |
| Connection Drops | >0.1/s | >0.5/s | Network or server issues affecting players |
| Health Check | Down for 1m | Down for 5m | Server unavailable |

## Grafana Dashboard Template

### Dashboard JSON

Create a new Grafana dashboard and import this JSON:

```json
{
  "dashboard": {
    "title": "BostonFear Arkham Horror Game Server",
    "tags": ["bostonfear", "game", "arkham-horror"],
    "timezone": "browser",
    "panels": [
      {
        "title": "Active Connections",
        "type": "graph",
        "targets": [
          {
            "expr": "arkham_horror_active_connections",
            "legendFormat": "Active Players"
          }
        ],
        "yaxes": [
          {
            "format": "short",
            "label": "Connections"
          }
        ]
      },
      {
        "title": "Broadcast Latency",
        "type": "graph",
        "targets": [
          {
            "expr": "arkham_horror_broadcast_latency_ms",
            "legendFormat": "Latency (ms)"
          }
        ],
        "yaxes": [
          {
            "format": "ms",
            "label": "Latency"
          }
        ],
        "alert": {
          "conditions": [
            {
              "evaluator": {
                "params": [200],
                "type": "gt"
              },
              "query": {
                "params": ["A", "5m", "now"]
              },
              "type": "query"
            }
          ],
          "name": "High Broadcast Latency"
        }
      },
      {
        "title": "Doom Level",
        "type": "graph",
        "targets": [
          {
            "expr": "arkham_horror_game_doom_level",
            "legendFormat": "Doom Counter"
          }
        ],
        "yaxes": [
          {
            "format": "short",
            "label": "Doom",
            "max": 12,
            "min": 0
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(arkham_horror_error_count[5m])",
            "legendFormat": "Errors/sec"
          }
        ],
        "yaxes": [
          {
            "format": "percentunit",
            "label": "Error Rate"
          }
        ]
      },
      {
        "title": "Memory Usage",
        "type": "gauge",
        "targets": [
          {
            "expr": "arkham_horror_memory_usage_percent",
            "legendFormat": "Memory %"
          }
        ],
        "options": {
          "thresholds": [
            {"value": 80, "color": "yellow"},
            {"value": 95, "color": "red"}
          ]
        }
      },
      {
        "title": "Action Types Distribution",
        "type": "piechart",
        "targets": [
          {
            "expr": "arkham_horror_action_type_count",
            "legendFormat": "{{action_type}}"
          }
        ]
      },
      {
        "title": "Connection Quality (Packet Loss)",
        "type": "graph",
        "targets": [
          {
            "expr": "arkham_horror_connection_packet_loss",
            "legendFormat": "Packet Loss %"
          }
        ],
        "yaxes": [
          {
            "format": "percentunit",
            "label": "Loss Rate"
          }
        ]
      },
      {
        "title": "Messages Sent/Received",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(arkham_horror_messages_sent[5m])",
            "legendFormat": "Sent/sec"
          },
          {
            "expr": "rate(arkham_horror_messages_received[5m])",
            "legendFormat": "Received/sec"
          }
        ]
      }
    ],
    "refresh": "5s",
    "time": {
      "from": "now-1h",
      "to": "now"
    }
  }
}
```

## Validation Steps

### 1. Import Dashboard

1. Open Grafana
2. Navigate to **Dashboards** → **Import**
3. Paste the JSON above or upload as file
4. Select your Prometheus data source
5. Click **Import**

### 2. Verify Metrics

Run these queries in Prometheus to confirm metrics are being collected:

```promql
# Check if server is up
up{job="bostonfear"}

# View broadcast latency
arkham_horror_broadcast_latency_ms

# Check active connections
arkham_horror_active_connections

# View doom level
arkham_horror_game_doom_level

# Check error rate
rate(arkham_horror_error_count[5m])
```

### 3. Test Alerts

Trigger test conditions to verify alerts fire:

```bash
# Generate high latency (simulate slow network)
tc qdisc add dev eth0 root netem delay 300ms

# Generate errors (send malformed messages)
echo '{"invalid": "json}' | websocat ws://localhost:8080/ws

# Check alert status in Prometheus
curl http://localhost:9090/api/v1/alerts | jq '.data.alerts[] | select(.labels.alertname=="HighBroadcastLatency")'
```

## Integration with Alertmanager

### Sample alertmanager.yml

```yaml
global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'component']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 12h
  receiver: 'team-game-ops'
  routes:
    - match:
        severity: critical
      receiver: 'team-game-ops-pager'
      continue: true
    - match:
        component: game_logic
      receiver: 'team-game-dev'

receivers:
  - name: 'team-game-ops'
    slack_configs:
      - api_url: 'YOUR_SLACK_WEBHOOK_URL'
        channel: '#game-ops'
        title: 'BostonFear Alert'
        text: '{{ range .Alerts }}{{ .Annotations.summary }}\n{{ .Annotations.description }}{{ end }}'
        
  - name: 'team-game-ops-pager'
    pagerduty_configs:
      - service_key: 'YOUR_PAGERDUTY_KEY'
        
  - name: 'team-game-dev'
    email_configs:
      - to: 'gamedev@example.com'
        from: 'alerts@example.com'
```

## Common Scenarios

### Scenario 1: High Latency Alert

**Symptom**: Broadcast latency consistently >200ms

**Investigation**:
1. Check network bandwidth: `iftop` or `nethogs`
2. Review server load: `top`, check CPU usage
3. Inspect Prometheus metrics for correlated spikes
4. Check logs for connection errors

**Resolution**:
- Scale horizontally if CPU-bound
- Optimize network path if network-bound
- Review action processing bottlenecks

### Scenario 2: Memory Pressure

**Symptom**: Memory usage >80%

**Investigation**:
1. Check active connections: `arkham_horror_active_connections`
2. Review goroutine count: `runtime.NumGoroutine()` via pprof
3. Look for memory leaks in connection handling

**Resolution**:
- Increase server memory allocation
- Implement connection limits
- Review and fix memory leaks

### Scenario 3: Doom Counter Reached Max

**Symptom**: Doom level ≥12

**Action**:
- This is a game-end condition, not a server issue
- Verify game state reset occurs properly
- Log game outcome for analytics

## Production Checklist

- [ ] Prometheus scraping `/metrics` endpoint every 15s
- [ ] Alert rules loaded in Prometheus
- [ ] Grafana dashboard imported and accessible
- [ ] Alertmanager configured with notification channels
- [ ] Test alerts verified (send test notifications)
- [ ] Runbook links added to alert annotations
- [ ] On-call rotation configured for critical alerts
- [ ] Dashboard shared with operations team

## References

- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Alerting Guide](https://grafana.com/docs/grafana/latest/alerting/)
- [BostonFear Metrics Endpoint](http://localhost:8080/metrics)
- [BostonFear Health Endpoint](http://localhost:8080/health)
