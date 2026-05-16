# Production Hardening Guide

This guide provides essential security configurations and best practices for deploying BostonFear to production environments.

## WebSocket Origin Validation

### Security Risk
By default, the BostonFear WebSocket server accepts connections from **any origin**. This is safe for local development but poses a security risk in production, as it allows:
- Cross-Site WebSocket Hijacking (CSWSH) attacks
- Unauthorized clients connecting from untrusted domains
- Potential data leakage to malicious origins

### Solution: Configure Allowed Origins

The `GameServer` provides `SetAllowedOrigins()` to restrict WebSocket upgrades to specific trusted domains.

#### Example: Production Configuration

Edit `cmd/server/main.go` after creating the game engine:

```go
package main

import (
	"log"
	"github.com/opd-ai/bostonfear/serverengine/arkhamhorror"
	"github.com/opd-ai/bostonfear/transport/ws"
)

func main() {
	// Initialize game engine
	gameEngine, err := arkhamhorror.NewEngine()
	if err != nil {
		log.Fatalf("failed to create game engine: %v", err)
	}

	// CRITICAL: Configure allowed origins for production
	gameEngine.SetAllowedOrigins([]string{
		"mygame.example.com",       // Production domain
		"staging.example.com",      // Staging environment
		"localhost:8080",            // Keep for local testing
	})

	// Start WebSocket server
	if err := ws.StartServer(":8080", gameEngine); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

#### Using Environment Variables

For deployment flexibility, read allowed origins from environment variables:

```go
import (
	"os"
	"strings"
)

func main() {
	gameEngine, err := arkhamhorror.NewEngine()
	if err != nil {
		log.Fatalf("failed to create game engine: %v", err)
	}

	// Read allowed origins from environment variable
	// Example: ALLOWED_ORIGINS="mygame.example.com,staging.example.com"
	originsEnv := os.Getenv("ALLOWED_ORIGINS")
	if originsEnv == "" {
		log.Fatal("ALLOWED_ORIGINS environment variable must be set for production")
	}
	
	allowedOrigins := strings.Split(originsEnv, ",")
	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(allowedOrigins[i])
	}
	
	gameEngine.SetAllowedOrigins(allowedOrigins)
	
	log.Printf("Allowed origins: %v", allowedOrigins)
	
	if err := ws.StartServer(":8080", gameEngine); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
```

#### Configuration via Viper/TOML

If using the Cobra CLI with Viper config (as documented in README):

```toml
# config.toml
[network]
allowed-origins = [
  "mygame.example.com",
  "staging.example.com"
]
```

Then in your server startup code:

```go
import "github.com/spf13/viper"

func main() {
	// Load config (assumes Viper is initialized)
	allowedOrigins := viper.GetStringSlice("network.allowed-origins")
	if len(allowedOrigins) == 0 {
		log.Fatal("network.allowed-origins must be configured for production")
	}
	
	gameEngine, _ := arkhamhorror.NewEngine()
	gameEngine.SetAllowedOrigins(allowedOrigins)
	
	// Start server...
}
```

### Verification

Test origin validation with curl:

```bash
# This should be REJECTED (wrong origin)
curl -H "Origin: https://malicious-site.com" \
     -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     http://yourserver.com:8080/ws

# Expected response: HTTP 403 Forbidden

# This should be ACCEPTED (allowed origin)
curl -H "Origin: https://mygame.example.com" \
     -H "Connection: Upgrade" \
     -H "Upgrade: websocket" \
     http://yourserver.com:8080/ws

# Expected response: HTTP 101 Switching Protocols
```

### Default Behavior (Development Mode)

If `SetAllowedOrigins()` is **not called** or called with an **empty slice**:
- The server accepts WebSocket upgrades from **any origin**
- This is **only safe for local development**
- Never deploy to production without configuring allowed origins

## SSL/TLS Configuration

### Requirement
Production deployments **must** use encrypted WebSocket connections (`wss://`) instead of unencrypted (`ws://`).

### Option 1: Reverse Proxy (Recommended)
Use nginx or Caddy as a TLS termination proxy:

```nginx
# nginx configuration
server {
    listen 443 ssl http2;
    server_name mygame.example.com;

    ssl_certificate /etc/letsencrypt/live/mygame.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/mygame.example.com/privkey.pem;

    location /ws {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### Option 2: Native Go TLS
Configure the server to listen with TLS directly:

```go
import (
	"crypto/tls"
	"net/http"
)

func main() {
	// Configure TLS
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.X25519,
		},
	}

	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
	}

	// Setup routes with game engine
	gameEngine, _ := arkhamhorror.NewEngine()
	gameEngine.SetAllowedOrigins([]string{"mygame.example.com"})
	
	http.Handle("/ws", gameEngine.WebSocketHandler())
	
	log.Fatal(server.ListenAndServeTLS("cert.pem", "key.pem"))
}
```

## Rate Limiting

### Connection Rate Limiting
Limit the number of concurrent connections per IP address to prevent resource exhaustion:

```go
// Example: Simple in-memory rate limiter
var (
	connCount = make(map[string]int)
	mu        sync.Mutex
	maxConnPerIP = 5
)

func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := r.RemoteAddr
		
		mu.Lock()
		count := connCount[clientIP]
		if count >= maxConnPerIP {
			mu.Unlock()
			http.Error(w, "Too many connections from this IP", http.StatusTooManyRequests)
			return
		}
		connCount[clientIP]++
		mu.Unlock()
		
		defer func() {
			mu.Lock()
			connCount[clientIP]--
			mu.Unlock()
		}()
		
		next.ServeHTTP(w, r)
	})
}
```

For production, use a robust rate limiter like `golang.org/x/time/rate` or integrate with Redis for distributed rate limiting.

## Input Validation

The BostonFear server validates all player actions and game state transitions. However, additional defense-in-depth measures:

### Message Size Limits
Configure maximum WebSocket message size in `transport/ws/server.go`:

```go
upgrader := websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Origin validation logic
	},
}
```

### Action Validation
All player actions are validated server-side:
- Resource bounds (Health/Sanity 1-10, Clues 0-5, Doom 0-12)
- Location adjacency for movement
- Turn order and action count limits
- Resource costs for special actions

Clients **cannot** bypass these validations—the server is the source of truth.

## Monitoring and Alerting

### Health Checks
Configure health check endpoint monitoring:

```bash
# Kubernetes liveness probe
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30

# Uptime monitoring (e.g., UptimeRobot)
curl -f http://yourserver.com:8080/health || exit 1
```

### Prometheus Metrics
Export metrics for observability:

```bash
# Scrape config for Prometheus
scrape_configs:
  - job_name: 'bostonfear'
    static_configs:
      - targets: ['yourserver.com:8080']
    metrics_path: '/metrics'
```

Key metrics to monitor:
- `arkham_horror_active_connections`: Alert if > max player capacity (6)
- `arkham_horror_memory_usage_percent`: Alert if > 80%
- `arkham_horror_game_doom_level`: Alert if frequently reaching 12 (game loss)
- `arkham_horror_error_rate`: Alert if > 5%

## Deployment Checklist

Before deploying to production:

- [ ] **Origins**: `SetAllowedOrigins()` configured with production domains
- [ ] **TLS**: Server uses `wss://` with valid SSL certificate
- [ ] **Environment Variables**: Sensitive config (origins, database URLs) not hardcoded
- [ ] **Rate Limiting**: Connection and action rate limits enforced
- [ ] **Monitoring**: Health checks and metrics endpoints configured
- [ ] **Logging**: Structured logging enabled with appropriate log levels
- [ ] **Firewall**: Port 8080 (or custom port) restricted to reverse proxy or trusted IPs
- [ ] **Dependencies**: `govulncheck` passes with zero vulnerabilities
- [ ] **Backups**: Game state persistence configured (if applicable)
- [ ] **Incident Response**: On-call rotation and escalation procedures defined

## Security Contacts

For security vulnerabilities or concerns, see `SECURITY.md` in the repository root for responsible disclosure procedures.

## Additional Resources

- [OWASP WebSocket Security Cheatsheet](https://cheatsheetseries.owasp.org/cheatsheets/WebSocket_Security_Cheat_Sheet.html)
- [Go Security Best Practices](https://go.dev/security/)
- [Let's Encrypt Free SSL Certificates](https://letsencrypt.org/)
