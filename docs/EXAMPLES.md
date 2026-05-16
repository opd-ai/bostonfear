# BostonFear Code Examples

This document provides runnable code examples and common troubleshooting scenarios for the BostonFear Arkham Horror multiplayer game server.

## Table of Contents
1. [Quick Start Examples](#quick-start-examples)
2. [Server Integration Examples](#server-integration-examples)
3. [Client Connection Examples](#client-connection-examples)
4. [Game Action Examples](#game-action-examples)
5. [Troubleshooting Scenarios](#troubleshooting-scenarios)

## Quick Start Examples

### Example 1: Basic Server Startup

**Goal**: Start the server and connect a desktop client for a single-player game.

```bash
# Terminal 1: Start the server
cd /path/to/bostonfear
go run . server

# Expected output (JSON-formatted):
# {"time":"...","level":"INFO","msg":"Game server started with broadcast and action handlers"}
# {"time":"...","level":"INFO","msg":"Arkham Horror server starting","address":"[::]:8080"}
# {"time":"...","level":"INFO","msg":"Game client","url":"http://localhost:8080/"}
# {"time":"...","level":"INFO","msg":"WebSocket endpoint","url":"ws://localhost:8080/ws"}
```

```bash
# Terminal 2: Connect desktop client
go run ./cmd/desktop -server ws://localhost:8080/ws

# Expected: Client window opens, you spawn at Downtown with 5 Health, 5 Sanity, 0 Clues
```

**Actions you can perform**:
- Move between adjacent locations (Downtown ↔ University ↔ Rivertown ↔ Northside)
- Gather resources (roll 2 dice for Health/Sanity)
- Investigate (roll 3 dice, need 2 successes for a clue)
- Cast Ward (costs 1 Sanity, roll 3 dice, need 3 successes to reduce doom by 2)

### Example 2: Multi-Player Game

**Goal**: Start a 2-player game with two desktop clients.

```bash
# Terminal 1: Start server
go run . server
```

```bash
# Terminal 2: Player 1
go run ./cmd/desktop -server ws://localhost:8080/ws
```

```bash
# Terminal 3: Player 2
go run ./cmd/desktop -server ws://localhost:8080/ws

# Both players see each other's positions, resources, and turn indicator
```

**Turn order**: Players alternate turns automatically. Each player gets 2 actions per turn.

### Example 3: Web Client (WASM)

**Goal**: Play in a browser using the WASM client.

```bash
# Step 1: Build WASM binary
GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web

# Step 2: Start server (serves WASM at /play)
go run . server

# Step 3: Open browser
# Navigate to http://localhost:8080/play
# Game loads in browser (2MB WASM binary)
```

**Browser requirements**: Chrome 57+, Firefox 52+, Safari 11+, Edge 16+

## Server Integration Examples

### Example 4: Custom Server with Module Selection

**Goal**: Start the server with a specific game module and custom config.

Create `my-config.toml`:
```toml
[server]
game = "arkhamhorror"
listen = ":9000"

[network]
allowed-origins = ["localhost:9000", "example.com"]

[scenario]
default_id = "scn.nightglass.harbor-signal"

[desktop]
server = "ws://localhost:9000/ws"
```

```bash
# Start server with custom config
go run . server --config my-config.toml

# Output:
# {"time":"...","level":"INFO","msg":"Arkham Horror server starting","address":"[::]:9000"}
```

### Example 5: Programmatic Server Startup

**Goal**: Embed the game server in your own Go application.

```go
package main

import (
    "context"
    "log"
    "net"
    "net/http"
    "time"

    "github.com/opd-ai/bostonfear/monitoring"
    "github.com/opd-ai/bostonfear/serverengine/arkhamhorror"
    "github.com/opd-ai/bostonfear/transport/ws"
)

func main() {
    // Create game engine
    engine, err := arkhamhorror.NewEngine()
    if err != nil {
        log.Fatal(err)
    }

    // Set allowed origins for production
    engine.SetAllowedOrigins([]string{"mygame.example.com", "localhost:8080"})

    // Start game engine background handlers
    if err := engine.Start(context.Background()); err != nil {
        log.Fatal(err)
    }

    // Create HTTP listener
    listener, err := net.Listen("tcp", ":8080")
    if err != nil {
        log.Fatal(err)
    }
    defer listener.Close()

    // Setup HTTP routes
    handlers := ws.RouteHandlers{
        WebSocket:  ws.NewWebSocketHandler(engine),
        Health:     monitoring.HealthHandler(engine),
        Metrics:    monitoring.MetricsHandler(engine),
        Play:       http.FileServer(http.Dir("client/wasm")),
        WASMAssets: http.FileServer(http.Dir("client/wasm")),
    }

    // Start server (blocks until shutdown)
    log.Println("Server starting on :8080")
    if err := ws.SetupServer(listener, handlers); err != nil {
        log.Fatal(err)
    }
}
```

Run it:
```bash
go run my-server.go
```

### Example 6: Health and Metrics Monitoring

**Goal**: Query server health and Prometheus metrics.

```bash
# Check health status
curl http://localhost:8080/health | jq

# Expected output:
# {
#   "status": "healthy",
#   "timestamp": 1749441525,
#   "performanceMetrics": {
#     "uptime": 39368523587,
#     "activeConnections": 2,
#     "responseTimeMs": 0.24,
#     "errorRate": 0
#   }
# }
```

```bash
# Get Prometheus metrics
curl http://localhost:8080/metrics | grep arkham_horror

# Expected output:
# arkham_horror_active_connections 2
# arkham_horror_broadcast_latency_ms 45
# arkham_horror_game_doom_level 3
# arkham_horror_memory_usage_percent 12.5
# arkham_horror_messages_sent 150
# arkham_horror_messages_received 100
```

## Client Connection Examples

### Example 7: Custom Client with Auto-Reconnection

**Goal**: Build a custom Go client that connects to the server.

```go
package main

import (
    "encoding/json"
    "fmt"
    "log"
    "time"

    "github.com/gorilla/websocket"
    "github.com/opd-ai/bostonfear/protocol"
)

func main() {
    serverURL := "ws://localhost:8080/ws"
    var conn *websocket.Conn
    var err error

    // Retry connection with exponential backoff
    for retries := 0; retries < 5; retries++ {
        conn, _, err = websocket.DefaultDialer.Dial(serverURL, nil)
        if err == nil {
            break
        }
        delay := time.Duration(5<<retries) * time.Second
        log.Printf("Connection failed, retrying in %s", delay)
        time.Sleep(delay)
    }
    if err != nil {
        log.Fatal("Failed to connect:", err)
    }
    defer conn.Close()

    log.Println("Connected to server")

    // Read connection status message
    _, msg, err := conn.ReadMessage()
    if err != nil {
        log.Fatal("Read error:", err)
    }

    var status protocol.ConnectionStatusMessage
    if err := json.Unmarshal(msg, &status); err != nil {
        log.Fatal("Unmarshal error:", err)
    }

    playerID := status.PlayerID
    log.Printf("Assigned player ID: %s", playerID)

    // Send a move action
    action := protocol.PlayerAction{
        Type:     "playerAction",
        PlayerID: playerID,
        Action:   "move",
        Target:   "University",
    }

    actionData, _ := json.Marshal(action)
    if err := conn.WriteMessage(websocket.TextMessage, actionData); err != nil {
        log.Fatal("Write error:", err)
    }

    log.Println("Sent move action to University")

    // Read game state update
    _, msg, err = conn.ReadMessage()
    if err != nil {
        log.Fatal("Read error:", err)
    }

    var gameStateMsg protocol.GameStateMessage
    if err := json.Unmarshal(msg, &gameStateMsg); err != nil {
        log.Fatal("Unmarshal error:", err)
    }

    log.Printf("Game state updated: Current player = %s, Doom = %d",
        gameStateMsg.Data.CurrentPlayer, gameStateMsg.Data.Doom)
}
```

Run it:
```bash
go run my-client.go
```

### Example 8: Session Reclaim with Token

**Goal**: Reconnect to the same player slot after disconnection.

```go
// First connection - receive token
var token string
// ... connect, receive connectionStatus message ...
token = status.Token  // Save this token

// Later, reconnect with token
reconnectURL := fmt.Sprintf("%s?token=%s", serverURL, token)
conn, _, err := websocket.DefaultDialer.Dial(reconnectURL, nil)
// Server restores your player slot
```

## Game Action Examples

### Example 9: Perform All 12 Actions

Here's how to programmatically perform each of the 12 available actions:

```go
playerID := "player1"  // Your assigned ID

// 1. Move
move := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "move", Target: "University",
}

// 2. Gather Resources
gather := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "gather",
}

// 3. Investigate
investigate := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "investigate",
}

// 4. Cast Ward
ward := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "ward",
}

// 5. Encounter
encounter := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "encounter",
}

// 6. Trade
trade := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "trade", Target: "player2",
}

// 7. Acquire Asset
acquireAsset := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "acquire_asset", Target: "flashlight",
}

// 8. Use Action Token
useToken := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "use_action_token",
}

// 9. Focus Action
focus := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "focus",
}

// 10. Recover
recover := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "recover",
}

// 11. Select Investigator (pregame only)
selectInvestigator := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "select_investigator", Target: "researcher",
}

// 12. Pass Turn
pass := protocol.PlayerAction{
    Type: "playerAction", PlayerID: playerID,
    Action: "pass",
}
```

### Example 10: Dice Roll Outcomes

Understanding dice results:

```go
// Server sends diceResult message:
{
    "type": "diceResult",
    "playerId": "player1",
    "results": ["success", "blank", "tentacle"]
}

// Interpretation:
// - "success": Counts toward action success threshold
// - "blank": No effect
// - "tentacle": Increments global doom counter by 1 (unconditional)

// Example: Investigate action (needs 2 successes)
// Roll: ["success", "success", "tentacle"]
// Outcome: Success! Gain 1 clue, doom increments by 1
```

## Troubleshooting Scenarios

### Scenario 1: Connection Refused

**Problem**: Desktop client fails to connect with "connection refused" error.

```bash
# Error:
# dial tcp [::1]:8080: connect: connection refused
```

**Solution**:
1. Verify server is running:
   ```bash
   curl http://localhost:8080/health
   ```
2. Check server logs for startup errors
3. Verify port 8080 is not blocked by firewall:
   ```bash
   sudo netstat -tlnp | grep 8080
   ```

### Scenario 2: WASM Load Error

**Problem**: Browser shows "failed to load WASM module" error.

**Solution**:
1. Verify WASM binary exists:
   ```bash
   ls -lh client/wasm/game.wasm
   # Should be ~2MB
   ```
2. Rebuild WASM binary:
   ```bash
   GOOS=js GOARCH=wasm go build -o client/wasm/game.wasm ./cmd/web
   ```
3. Check browser console for MIME type errors (server must serve with `application/wasm`)
4. Verify browser supports WASM:
   ```javascript
   console.log(typeof WebAssembly);  // Should not be "undefined"
   ```

### Scenario 3: Game State Desync

**Problem**: Client shows different game state than server.

**Symptoms**:
- Player position mismatch
- Resource values incorrect
- Doom counter differs

**Solution**:
1. Check server logs for validation errors:
   ```bash
   # Look for:
   # {"level":"ERROR","msg":"Game state validation errors detected"}
   ```
2. Restart client to force full state resync
3. Check network latency:
   ```bash
   curl http://localhost:8080/metrics | grep broadcast_latency
   # Should be <200ms
   ```

### Scenario 4: High Broadcast Latency

**Problem**: Slow game state updates, latency >200ms.

**Diagnosis**:
```bash
# Check metrics
curl http://localhost:8080/metrics | grep broadcast_latency_ms
# arkham_horror_broadcast_latency_ms 450

# Check active connections
curl http://localhost:8080/metrics | grep active_connections
# arkham_horror_active_connections 6
```

**Solutions**:
1. Reduce number of concurrent players (limit 6)
2. Check server CPU usage:
   ```bash
   top -p $(pgrep bostonfear)
   ```
3. Check network bandwidth with `iftop` or `nethogs`
4. Consider horizontal scaling for >6 players

### Scenario 5: Mobile Connection Issues

**Problem**: Mobile client can't connect to server on local network.

**Android Emulator**:
```bash
# Use special IP for host machine
ws://10.0.2.2:8080/ws
```

**iOS Simulator**:
```bash
# Use localhost (works in iOS simulator)
ws://localhost:8080/ws
```

**Physical Device**:
```bash
# Use host machine's LAN IP
ip addr show | grep "inet " | grep -v 127.0.0.1

# Example: 192.168.1.100
ws://192.168.1.100:8080/ws
```

**Firewall rules** (Linux):
```bash
# Allow port 8080
sudo ufw allow 8080/tcp
```

### Scenario 6: Tests Fail with "display: cannot connect to X server"

**Problem**: Ebitengine tests fail on headless Linux servers.

**Solution**: Use Xvfb (X Virtual Framebuffer)

```bash
# Install Xvfb
sudo apt-get install xvfb

# Run tests with virtual display
Xvfb :99 -screen 0 1280x720x24 &
DISPLAY=:99 go test -tags=requires_display ./client/ebiten/...

# Or use xvfb-run wrapper
xvfb-run -a go test -tags=requires_display ./client/ebiten/...
```

## Additional Resources

- [README](../README.md): Project overview and quick setup
- [CONTRIBUTING](../CONTRIBUTING.md): Contribution guidelines
- [Architecture Decision Records](adr/README.md): Design decisions
- [ALERTING](ALERTING.md): Monitoring and alerting setup
- [ROADMAP](../ROADMAP.md): Future development plans

---

**Have a question not covered here?** Open a [GitHub Discussion](https://github.com/opd-ai/bostonfear/discussions) or check existing [Issues](https://github.com/opd-ai/bostonfear/issues).
