package adapters

import (
	"testing"

	"github.com/opd-ai/bostonfear/serverengine/common/contracts"
)

// TestAdapterImplementsInterface verifies the broadcast adapter implements the canonical interface.
func TestAdapterImplementsInterface(t *testing.T) {
	adapter := NewBroadcastAdapter()

	// Compile-time check: ensure adapter implements BroadcastPayloadAdapter
	var _ contracts.BroadcastPayloadAdapter = adapter

	if adapter == nil {
		t.Fatal("NewBroadcastAdapter returned nil")
	}
}

// TestShapeGameStatePayload verifies game state payload shaping.
func TestShapeGameStatePayload(t *testing.T) {
	adapter := NewBroadcastAdapter()

	mockState := map[string]interface{}{
		"currentPlayer": "player1",
		"doom":          5,
		"museum":        "entrance",
	}

	payload := adapter.ShapeGameStatePayload(mockState)

	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		t.Fatal("payload is not a map")
	}

	if payloadMap["type"] != "gameState" {
		t.Errorf("expected type=gameState, got %v", payloadMap["type"])
	}

	if payloadMap["data"] == nil {
		t.Error("expected data field, got nil")
	}
}

// TestShapeActionResultPayload verifies action result payload shaping.
func TestShapeActionResultPayload(t *testing.T) {
	adapter := NewBroadcastAdapter()

	// Test with nil resources (fallback shape)
	payload := adapter.ShapeActionResultPayload("rollDice", "success", nil)

	payloadMap, ok := payload.(map[string]interface{})
	if !ok {
		t.Fatal("payload is not a map")
	}

	if payloadMap["type"] != "gameUpdate" {
		t.Errorf("expected type=gameUpdate, got %v", payloadMap["type"])
	}

	if payloadMap["event"] != "rollDice" {
		t.Errorf("expected event=rollDice, got %v", payloadMap["event"])
	}

	if payloadMap["result"] != "success" {
		t.Errorf("expected result=success, got %v", payloadMap["result"])
	}

	// Test with provided resources (pass-through)
	providedResources := map[string]interface{}{"custom": "data"}
	payload2 := adapter.ShapeActionResultPayload("lockDie", "fail", providedResources)

	payload2Map, ok := payload2.(map[string]interface{})
	if !ok {
		t.Fatal("payload2 is not a map")
	}

	if payload2Map["custom"] != "data" {
		t.Error("expected provided resources to be returned directly")
	}
}

// TestShapeDiceResultPayload verifies dice result payload shaping.
func TestShapeDiceResultPayload(t *testing.T) {
	adapter := NewBroadcastAdapter()

	mockDiceResult := &DiceResultPayload{
		Type:          "diceResult",
		PlayerID:      "player1",
		Action:        "rollDice",
		LockedResults: []interface{}{"red", "green"},
		ActiveResults: []interface{}{"terror", "lore"},
		TerrorCount:   1,
		Success:       true,
		DoomIncrease:  0,
	}

	payload := adapter.ShapeDiceResultPayload(mockDiceResult)

	if payload != mockDiceResult {
		t.Error("expected dice result to be returned directly")
	}

	// Test map input
	mapResult := map[string]interface{}{
		"type":     "diceResult",
		"playerId": "player2",
	}
	payload2 := adapter.ShapeDiceResultPayload(mapResult)
	if payload2 == nil {
		t.Error("expected map result to be returned")
	}
}

// TestNewDiceResultPayload verifies the helper function creates valid payloads.
func TestNewDiceResultPayload(t *testing.T) {
	locked := []interface{}{"red", "green"}
	active := []interface{}{"terror", "lore"}

	payload := NewDiceResultPayload("player1", "rollDice", locked, active, 1, true, 0)

	if payload.Type != "diceResult" {
		t.Errorf("expected type=diceResult, got %s", payload.Type)
	}
	if payload.PlayerID != "player1" {
		t.Errorf("expected playerID=player1, got %s", payload.PlayerID)
	}
	if payload.Action != "rollDice" {
		t.Errorf("expected action=rollDice, got %s", payload.Action)
	}
	if len(payload.LockedResults) != 2 {
		t.Errorf("expected 2 locked results, got %d", len(payload.LockedResults))
	}
	if len(payload.ActiveResults) != 2 {
		t.Errorf("expected 2 active results, got %d", len(payload.ActiveResults))
	}
	if payload.TerrorCount != 1 {
		t.Errorf("expected terrorCount=1, got %d", payload.TerrorCount)
	}
	if !payload.Success {
		t.Error("expected success=true")
	}
	if payload.DoomIncrease != 0 {
		t.Errorf("expected doomIncrease=0, got %d", payload.DoomIncrease)
	}
}
