package ebiten

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

// serverMessage is the top-level envelope for all messages from the server.
// The server always sends {"type": "...", ...} or wraps data under a "data" key.
type serverMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`

	// The server's gameState broadcast inlines all fields at the top level.
	// We decode them here so a single Unmarshal covers the full broadcast.
	Players       map[string]*Player `json:"players"`
	CurrentPlayer string             `json:"currentPlayer"`
	Doom          int                `json:"doom"`
	GamePhase     string             `json:"gamePhase"`
	TurnOrder     []string           `json:"turnOrder"`
	GameStarted   bool               `json:"gameStarted"`
	WinCondition  bool               `json:"winCondition"`
	LoseCondition bool               `json:"loseCondition"`
	RequiredClues int                `json:"requiredClues"`
}

// PlayerActionMessage is the outbound message the client sends to the server.
type PlayerActionMessage struct {
	Type     string `json:"type"`
	PlayerID string `json:"playerId"`
	Action   string `json:"action"`
	Target   string `json:"target,omitempty"`
}

// NetClient manages the WebSocket connection to the game server.
// It reads incoming messages, routes them to LocalState, and exposes a channel
// for the game loop to queue outbound player actions.
type NetClient struct {
	state     *LocalState
	actionsCh chan PlayerActionMessage

	// dialer is the gorilla/websocket dialer used to open connections.
	dialer *websocket.Dialer
}

// NewNetClient creates a NetClient wired to the given LocalState.
// The returned client is not yet connected; call Connect to start.
func NewNetClient(state *LocalState) *NetClient {
	return &NetClient{
		state:     state,
		actionsCh: make(chan PlayerActionMessage, 16),
		dialer:    websocket.DefaultDialer,
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
func (c *NetClient) reconnectLoop() {
	delay := 5 * time.Second
	const maxDelay = 30 * time.Second

	for {
		conn, _, err := c.dialer.Dial(c.state.ServerURL, nil)
		if err != nil {
			log.Printf("net: dial %s failed: %v — retrying in %s", c.state.ServerURL, err, delay)
			c.state.SetConnected(false)
			time.Sleep(delay)
			if delay < maxDelay {
				delay *= 2
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
		time.Sleep(delay)
		if delay < maxDelay {
			delay *= 2
		}
	}
}

// runConnection drives read and write loops for an established connection.
// It blocks until the connection is closed or fails.
func (c *NetClient) runConnection(conn *websocket.Conn) {
	done := make(chan struct{})
	go c.writeLoop(conn, done)
	c.readLoop(conn, done)
}

// writeLoop forwards queued actions to the server until the connection fails.
// It signals completion by closing done.
func (c *NetClient) writeLoop(conn *websocket.Conn, done chan struct{}) {
	defer close(done)
	for {
		select {
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
func (c *NetClient) readLoop(conn *websocket.Conn, done chan struct{}) {
	for {
		select {
		case <-done:
			return
		default:
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			log.Printf("net: read: %v", err)
			conn.Close()
			<-done
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
func decodeGameState(msg serverMessage) GameState {
	gs := GameState{
		Players:       msg.Players,
		CurrentPlayer: msg.CurrentPlayer,
		Doom:          msg.Doom,
		GamePhase:     msg.GamePhase,
		TurnOrder:     msg.TurnOrder,
		GameStarted:   msg.GameStarted,
		WinCondition:  msg.WinCondition,
		LoseCondition: msg.LoseCondition,
		RequiredClues: msg.RequiredClues,
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
// It also extracts the player ID on the first connectionStatus received.
func (c *NetClient) applyConnectionStatus(data []byte) {
	var cs ConnectionStatusData
	if err := json.Unmarshal(data, &cs); err != nil {
		log.Printf("net: unmarshal connectionStatus: %v", err)
		return
	}
	if c.state.PlayerID == "" && cs.PlayerID != "" {
		c.state.SetPlayerID(cs.PlayerID)
	}
	c.state.UpdateConnectionStatus(cs)
}
