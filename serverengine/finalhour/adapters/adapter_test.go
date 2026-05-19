package adapters

import (
	"testing"
)

func TestNewBroadcastAdapter(t *testing.T) {
	adapter := NewBroadcastAdapter()
	if adapter == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestShapeGameStatePayload(t *testing.T) {
	adapter := NewBroadcastAdapter()

	state := map[string]interface{}{
		"countdown": 10,
		"players":   []string{"player1", "player2"},
	}

	shaped := adapter.ShapeGameStatePayload(state)

	payload, ok := shaped.(map[string]interface{})
	if !ok {
		t.Fatal("expected shaped payload to be a map")
	}

	if payload["type"] != "gameState" {
		t.Errorf("expected type 'gameState', got '%v'", payload["type"])
	}

	if payload["data"] == nil {
		t.Error("expected data field to be present")
	}
}

func TestShapeActionResultPayloadWithResources(t *testing.T) {
	adapter := NewBroadcastAdapter()

	resources := map[string]interface{}{
		"type":   "gameUpdate",
		"event":  "priorityResolution",
		"result": "success",
	}

	shaped := adapter.ShapeActionResultPayload("action", "result", resources)

	payload, ok := shaped.(map[string]interface{})
	if !ok {
		t.Fatal("expected shaped payload to be a map")
	}

	if payload["type"] != "gameUpdate" {
		t.Errorf("expected type 'gameUpdate', got '%v'", payload["type"])
	}
}

func TestShapeActionResultPayloadFallback(t *testing.T) {
	adapter := NewBroadcastAdapter()

	shaped := adapter.ShapeActionResultPayload("bidPriority", "success", nil)

	payload, ok := shaped.(map[string]interface{})
	if !ok {
		t.Fatal("expected shaped payload to be a map")
	}

	if payload["type"] != "gameUpdate" {
		t.Errorf("expected type 'gameUpdate', got '%v'", payload["type"])
	}

	if payload["event"] != "bidPriority" {
		t.Errorf("expected event 'bidPriority', got '%v'", payload["event"])
	}

	if payload["result"] != "success" {
		t.Errorf("expected result 'success', got '%v'", payload["result"])
	}
}

func TestShapeDiceResultPayloadWithPriorityResult(t *testing.T) {
	adapter := NewBroadcastAdapter()

	priorityResult := &PriorityResultPayload{
		Type:         "priorityResult",
		PlayerID:     "player1",
		Action:       "resolveAction",
		PriorityBid:  5,
		Success:      true,
		CountdownDec: 1,
	}

	shaped := adapter.ShapeDiceResultPayload(priorityResult)

	result, ok := shaped.(*PriorityResultPayload)
	if !ok {
		t.Fatal("expected shaped payload to be *PriorityResultPayload")
	}

	if result.PlayerID != "player1" {
		t.Errorf("expected player1, got %s", result.PlayerID)
	}
}

func TestShapeDiceResultPayloadWithMap(t *testing.T) {
	adapter := NewBroadcastAdapter()

	mapResult := map[string]interface{}{
		"type":     "priorityResult",
		"playerId": "player2",
	}

	shaped := adapter.ShapeDiceResultPayload(mapResult)

	result, ok := shaped.(map[string]interface{})
	if !ok {
		t.Fatal("expected shaped payload to be a map")
	}

	if result["playerId"] != "player2" {
		t.Errorf("expected player2, got %v", result["playerId"])
	}
}

func TestConnectionStatusPayload(t *testing.T) {
	payload := ConnectionStatusPayload("player1", "connected")

	if payload["type"] != "connectionStatus" {
		t.Errorf("expected type 'connectionStatus', got '%v'", payload["type"])
	}
	if payload["playerId"] != "player1" {
		t.Errorf("expected playerId 'player1', got '%v'", payload["playerId"])
	}
	if payload["status"] != "connected" {
		t.Errorf("expected status 'connected', got '%v'", payload["status"])
	}
}

func TestGameUpdatePayload(t *testing.T) {
	payload := GameUpdatePayload("player1", "action", "success")

	if payload["type"] != "gameUpdate" {
		t.Errorf("expected type 'gameUpdate', got '%v'", payload["type"])
	}
	if payload["playerId"] != "player1" {
		t.Errorf("expected playerId 'player1', got '%v'", payload["playerId"])
	}
	if payload["event"] != "action" {
		t.Errorf("expected event 'action', got '%v'", payload["event"])
	}
	if payload["result"] != "success" {
		t.Errorf("expected result 'success', got '%v'", payload["result"])
	}
}
