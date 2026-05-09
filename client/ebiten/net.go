package ebiten

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/opd-ai/bostonfear/protocol"
)

// serverMessage is the top-level envelope for all messages from the server.
// The server sends {"type": "...", "data": {...}} where game state lives under "data".
type serverMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// PlayerActionMessage is the outbound message the client sends to the server.
type PlayerActionMessage = protocol.PlayerActionMessage

// NetClient manages the WebSocket connection to the game server.
// It reads incoming messages, routes them to LocalState, and exposes a channel
// for the game loop to queue outbound player actions.
type NetClient struct {
	state       *LocalState
	actionsCh   chan PlayerActionMessage
	reconnectCh chan struct{} // closed/sent to trigger an immediate redial

	// dialer is the gorilla/websocket dialer used to open connections.
	dialer *websocket.Dialer
}

// NewNetClient creates a NetClient wired to the given LocalState.
// The returned client is not yet connected; call Connect to start.
func NewNetClient(state *LocalState) *NetClient {
	return &NetClient{
		state:       state,
		actionsCh:   make(chan PlayerActionMessage, 16),
		reconnectCh: make(chan struct{}, 1),
		dialer:      websocket.DefaultDialer,
	}
}

// Reconnect aborts any in-progress backoff sleep and immediately redials,
// using whatever ServerURL is currently set on the LocalState.
// This is called by SceneConnect when the user submits a new server address.
func (c *NetClient) Reconnect() {
	select {
	case c.reconnectCh <- struct{}{}:
	default: // signal already pending; no-op
	}
}

// SendAction enqueues a player action for delivery to the server.
// Non-blocking: drops the action if the channel is full.
func (c *NetClient) SendAction(action PlayerActionMessage) {
	select {
	case c.actionsCh <- action:
	default:
		log.Printf("net: action channel full, dropping %s", action.Action)
	}
}

// Connect dials the server and starts goroutines for reading and writing.
// It returns immediately; reconnection is handled automatically with a 5-second
// initial delay matching the legacy JS client behaviour.
func (c *NetClient) Connect() {
	go c.reconnectLoop()
}

// reconnectLoop dials, runs the read/write pair, and retries on failure.
// Initial delay is 5 seconds; subsequent delays double up to 30 seconds.
// If a reconnect token is available it is appended as a query parameter so
// the server can restore the previous player slot.
// Sending on reconnectCh (via Reconnect()) aborts the current backoff and
// redials immediately with the current ServerURL.
func (c *NetClient) reconnectLoop() {
	delay := 5 * time.Second
	const maxDelay = 30 * time.Second

	for {
		// Drain any stale reconnect signal before dialing.
		select {
		case <-c.reconnectCh:
		default:
		}

		dialURL := c.state.ServerURL
		if tok := c.state.GetReconnectToken(); tok != "" {
			dialURL = dialURL + "?token=" + tok
		}

		conn, _, err := c.dialer.Dial(dialURL, nil)
		if err != nil {
			log.Printf("net: dial %s failed: %v — retrying in %s", c.state.ServerURL, err, delay)
			c.state.SetConnected(false)
			// Wait for either the backoff timer or an explicit reconnect signal.
			select {
			case <-time.After(delay):
				delay *= 2
				if delay > maxDelay {
					delay = maxDelay
				}
			case <-c.reconnectCh:
				delay = 5 * time.Second // reset on manual reconnect
			}
			continue
		}

		// Reset delay on successful connection.
		delay = 5 * time.Second
		c.state.SetConnected(true)
		log.Printf("net: connected to %s", c.state.ServerURL)

		c.runConnection(conn)

		c.state.SetConnected(false)
		log.Printf("net: connection lost — retrying in %s", delay)
		// Wait for either the backoff timer or an explicit reconnect signal.
		select {
		case <-time.After(delay):
			delay *= 2
			if delay > maxDelay {
				delay = maxDelay
			}
		case <-c.reconnectCh:
			delay = 5 * time.Second
		}
	}
}

// runConnection drives read and write loops for an established connection.
// It blocks until the connection is closed or fails.
func (c *NetClient) runConnection(conn *websocket.Conn) {
	stop := make(chan struct{})
	done := make(chan struct{})
	var stopOnce sync.Once
	signalStop := func() {
		stopOnce.Do(func() {
			close(stop)
		})
	}

	go c.writeLoop(conn, stop, done)
	c.readLoop(conn, stop, signalStop)
	signalStop()
	<-done
	_ = conn.Close()
}

// writeLoop forwards queued actions to the server until the connection fails.
// It signals completion by closing done.
func (c *NetClient) writeLoop(conn *websocket.Conn, stop <-chan struct{}, done chan struct{}) {
	defer close(done)
	for {
		select {
		case <-stop:
			return
		case action := <-c.actionsCh:
			data, err := json.Marshal(action)
			if err != nil {
				log.Printf("net: marshal action: %v", err)
				continue
			}
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("net: write: %v", err)
				return
			}
		}
	}
}

// readLoop reads and routes incoming messages until the connection closes or done is signalled.
func (c *NetClient) readLoop(conn *websocket.Conn, stop <-chan struct{}, signalStop func()) {
	for {
		select {
		case <-stop:
			return
		default:
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("net: read: %v", err)
			signalStop()
			return
		}

		c.routeMessage(data)
	}
}

// routeMessage decodes a raw server message and updates LocalState.
func (c *NetClient) routeMessage(data []byte) {
	var msg serverMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("net: unmarshal: %v", err)
		return
	}

	switch msg.Type {
	case "gameState":
		c.state.UpdateGame(decodeGameState(msg))
	case "diceResult":
		c.applyDiceResult(data)
	case "gameUpdate":
		c.applyGameUpdate(data)
	case "connectionStatus":
		c.applyConnectionStatus(data)
	default:
		// Unknown message type; ignore silently to remain forward-compatible.
	}
}

// decodeGameState converts a serverMessage into a GameState value.
// The server wraps game state under the "data" key, so we unmarshal msg.Data directly.
func decodeGameState(msg serverMessage) GameState {
	var gs GameState
	if len(msg.Data) > 0 {
		if err := json.Unmarshal(msg.Data, &gs); err != nil {
			log.Printf("net: unmarshal gameState data: %v", err)
		}
	}
	if gs.Players == nil {
		gs.Players = make(map[string]*Player)
	}
	return gs
}

// applyDiceResult decodes and forwards a diceResult payload to LocalState.
func (c *NetClient) applyDiceResult(data []byte) {
	var dr DiceResultData
	if err := json.Unmarshal(data, &dr); err != nil {
		log.Printf("net: unmarshal diceResult: %v", err)
		return
	}
	c.state.UpdateDiceResult(dr)
}

// applyGameUpdate decodes and forwards a gameUpdate payload to LocalState.
func (c *NetClient) applyGameUpdate(data []byte) {
	var gu GameUpdateData
	if err := json.Unmarshal(data, &gu); err != nil {
		log.Printf("net: unmarshal gameUpdate: %v", err)
		return
	}
	c.state.UpdateGameEvent(gu)
}

// applyConnectionStatus decodes and forwards a connectionStatus payload to LocalState.
// It also extracts the player ID and reconnect token on the first connectionStatus received.
func (c *NetClient) applyConnectionStatus(data []byte) {
	var cs ConnectionStatusData
	if err := json.Unmarshal(data, &cs); err != nil {
		log.Printf("net: unmarshal connectionStatus: %v", err)
		return
	}
	if c.state.PlayerID == "" && cs.PlayerID != "" {
		c.state.SetPlayerID(cs.PlayerID)
	}
	if cs.Token != "" {
		c.state.SetReconnectToken(cs.Token)
	}
	c.state.UpdateConnectionStatus(cs)
}
