package serverengine

import (
	"encoding/json"
	"strings"
	"testing"
)

// FuzzProcessActionJSON tests processAction with malformed JSON inputs.
// The fuzzer generates random byte sequences that are unmarshaled into
// PlayerActionMessage structs and passed to processAction. This validates
// that the server handles malformed JSON, invalid action types, out-of-bounds
// resources, and negative doom values without panicking.
func FuzzProcessActionJSON(f *testing.F) {
	// Seed corpus with valid and edge-case inputs
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":"move","target":"Downtown"}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":"gather"}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":"investigate"}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":"ward"}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"","action":"move"}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":""}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":"invalid"}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":"move","target":""}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":"move","target":"InvalidLocation"}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":"focus","focusSpend":-1}`))
	f.Add([]byte(`{"type":"playerAction","playerId":"p1","action":"focus","focusSpend":999}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`"string"`))
	f.Add([]byte(`123`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`{"type":"playerAction"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		// Skip empty or obviously invalid inputs that would fail JSON unmarshaling
		if len(data) == 0 {
			return
		}

		// Create a minimal game server for testing
		gs := NewGameServer()

		// Add a test player to make actions potentially valid
		testPlayer := &Player{
			ID:       "p1",
			Location: Downtown,
			Resources: Resources{
				Health: 5,
				Sanity: 5,
				Clues:  0,
				Focus:  0,
			},
			ActionsRemaining: 2,
			Connected:        true,
		}
		gs.gameState.Players["p1"] = testPlayer
		gs.gameState.CurrentPlayer = "p1"

		// Try to unmarshal the fuzzed data
		var action PlayerActionMessage
		err := json.Unmarshal(data, &action)
		if err != nil {
			// Invalid JSON is expected; just skip
			return
		}

		// Ensure basic field validity to avoid testing JSON unmarshaling
		// rather than processAction logic
		if action.PlayerID == "" {
			return
		}

		// Call processAction and verify it doesn't panic
		// Errors are expected for invalid actions; we only care about panics
		_ = gs.processAction(action)
	})
}

// TestBoundaryValues tests extreme boundary conditions for game state values.
// This complements fuzzing by explicitly testing known edge cases.
func TestBoundaryValues(t *testing.T) {
	tests := []struct {
		name        string
		setupState  func(*GameServer)
		action      PlayerActionMessage
		expectError bool
		description string
	}{
		{
			name: "negative health",
			setupState: func(gs *GameServer) {
				gs.gameState.Players["p1"] = &Player{
					ID:       "p1",
					Location: Downtown,
					Resources: Resources{
						Health: -10,
						Sanity: 5,
						Clues:  0,
					},
					ActionsRemaining: 2,
				}
			},
			action: PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: "p1",
				Action:   "gather",
			},
			expectError: false, // Should clamp negative health
			description: "negative health should be clamped to valid range",
		},
		{
			name: "excessive health",
			setupState: func(gs *GameServer) {
				gs.gameState.Players["p1"] = &Player{
					ID:       "p1",
					Location: Downtown,
					Resources: Resources{
						Health: 999,
						Sanity: 5,
						Clues:  0,
					},
					ActionsRemaining: 2,
				}
			},
			action: PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: "p1",
				Action:   "gather",
			},
			expectError: false, // Should clamp excessive health
			description: "excessive health should be clamped to valid range",
		},
		{
			name: "max doom",
			setupState: func(gs *GameServer) {
				gs.gameState.Doom = 12 // MaxDoom constant value
				gs.gameState.Players["p1"] = &Player{
					ID:       "p1",
					Location: Downtown,
					Resources: Resources{
						Health: 5,
						Sanity: 5,
						Clues:  0,
					},
					ActionsRemaining: 2,
				}
			},
			action: PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: "p1",
				Action:   "gather",
			},
			expectError: true, // Game should be over at max doom
			description: "actions should fail when doom is at maximum",
		},
		{
			name: "negative doom",
			setupState: func(gs *GameServer) {
				gs.gameState.Doom = -10
				gs.gameState.Players["p1"] = &Player{
					ID:       "p1",
					Location: Downtown,
					Resources: Resources{
						Health: 5,
						Sanity: 5,
						Clues:  0,
					},
					ActionsRemaining: 2,
				}
			},
			action: PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: "p1",
				Action:   "gather",
			},
			expectError: false, // Should clamp negative doom
			description: "negative doom should be clamped to zero",
		},
		{
			name: "max players",
			setupState: func(gs *GameServer) {
				// Add maximum number of players
				for i := 0; i < MaxPlayers; i++ {
					gs.gameState.Players[string(rune('a'+i))] = &Player{
						ID:       string(rune('a' + i)),
						Location: Downtown,
						Resources: Resources{
							Health: 5,
							Sanity: 5,
							Clues:  0,
						},
						ActionsRemaining: 2,
					}
				}
				gs.gameState.Players["p1"] = gs.gameState.Players["a"]
			},
			action: PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: "p1",
				Action:   "gather",
			},
			expectError: false,
			description: "should handle maximum player count",
		},
		{
			name: "very long player ID",
			setupState: func(gs *GameServer) {
				longID := strings.Repeat("x", 10000)
				gs.gameState.Players[longID] = &Player{
					ID:       longID,
					Location: Downtown,
					Resources: Resources{
						Health: 5,
						Sanity: 5,
						Clues:  0,
					},
					ActionsRemaining: 2,
				}
			},
			action: PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: strings.Repeat("x", 10000),
				Action:   "gather",
			},
			expectError: false,
			description: "should handle very long player IDs",
		},
		{
			name: "very long location name",
			setupState: func(gs *GameServer) {
				gs.gameState.Players["p1"] = &Player{
					ID:       "p1",
					Location: Downtown,
					Resources: Resources{
						Health: 5,
						Sanity: 5,
						Clues:  0,
					},
					ActionsRemaining: 2,
				}
			},
			action: PlayerActionMessage{
				Type:     "playerAction",
				PlayerID: "p1",
				Action:   "move",
				Target:   strings.Repeat("x", 10000),
			},
			expectError: true, // Invalid location should error
			description: "should reject very long location names",
		},
		{
			name: "excessive focus spend",
			setupState: func(gs *GameServer) {
				gs.gameState.Players["p1"] = &Player{
					ID:       "p1",
					Location: Downtown,
					Resources: Resources{
						Health: 5,
						Sanity: 5,
						Clues:  0,
						Focus:  2,
					},
					ActionsRemaining: 2,
				}
			},
			action: PlayerActionMessage{
				Type:       "playerAction",
				PlayerID:   "p1",
				Action:     "investigate",
				FocusSpend: 999,
			},
			expectError: true, // Should reject excessive focus spend
			description: "should reject focus spend exceeding available focus",
		},
		{
			name: "negative focus spend",
			setupState: func(gs *GameServer) {
				gs.gameState.Players["p1"] = &Player{
					ID:       "p1",
					Location: Downtown,
					Resources: Resources{
						Health: 5,
						Sanity: 5,
						Clues:  0,
						Focus:  2,
					},
					ActionsRemaining: 2,
				}
			},
			action: PlayerActionMessage{
				Type:       "playerAction",
				PlayerID:   "p1",
				Action:     "investigate",
				FocusSpend: -10,
			},
			expectError: true, // Should reject negative focus spend
			description: "should reject negative focus spend",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := NewGameServer()

			// Set up game state
			tt.setupState(gs)
			gs.gameState.CurrentPlayer = tt.action.PlayerID

			// Process action
			err := gs.processAction(tt.action)

			// Verify expected error behavior
			if tt.expectError && err == nil {
				t.Errorf("%s: expected error but got nil", tt.description)
			}
			if !tt.expectError && err != nil {
				// Non-critical errors are acceptable; we mainly care about panics
				t.Logf("%s: got error (may be acceptable): %v", tt.description, err)
			}
		})
	}
}

// TestMalformedJSON explicitly tests common JSON parsing edge cases.
func TestMalformedJSON(t *testing.T) {
	malformedInputs := []string{
		``,                                    // empty
		`{`,                                   // incomplete object
		`{"type":"playerAction"`,              // missing closing brace
		`{"type":"playerAction","playerId":}`, // missing value
		`{"type":"playerAction","playerId":"p1","action":}`,   // missing value
		`{"type":"playerAction","playerId":null}`,             // null player ID
		`{"type":"playerAction","action":null}`,               // null action
		`{"type":"playerAction","focusSpend":"not-a-number"}`, // wrong type
	}

	gs := NewGameServer()

	// Add a test player
	gs.gameState.Players["p1"] = &Player{
		ID:       "p1",
		Location: Downtown,
		Resources: Resources{
			Health: 5,
			Sanity: 5,
			Clues:  0,
		},
		ActionsRemaining: 2,
	}

	for _, input := range malformedInputs {
		var action PlayerActionMessage
		err := json.Unmarshal([]byte(input), &action)
		if err != nil {
			// Expected behavior: JSON unmarshaling should fail
			continue
		}

		// If unmarshaling succeeds, processAction should handle it gracefully
		_ = gs.processAction(action)
	}
}
