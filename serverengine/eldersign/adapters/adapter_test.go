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
}
