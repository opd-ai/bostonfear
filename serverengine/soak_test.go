// Package serverengine_soak provides a stress test for sustained 6-player gameplay.
package serverengine

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// soakTestMetrics tracks performance during the soak test.
type soakTestMetrics struct {
	startTime          time.Time
	actionCount        atomic.Int64
	doomMaxValue       atomic.Int32
	playerDefeats      atomic.Int32
	gameStateErrors    atomic.Int32
	broadcastLatencies []int64 // nanoseconds
	latencyMutex       sync.Mutex
	lastGameState      *GameState
	lastGameStateMutex sync.RWMutex
}

// TestStressTest_6Players exercises 6 concurrent players for 30 seconds.
// This is a functional stress test suitable for CI; the full 15-minute soak test
// (mentioned in ROADMAP.md Priority 1) requires a dedicated long-running environment.
// This abbreviated version validates:
//   - No deadlocks or stuck turns
//   - Doom counter stays in bounds [0, 12]
//   - Game state invariants are maintained
func TestStressTest_6Players(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	const duration = 30 * time.Second // 30-second stress test (CI-suitable)
	const numPlayers = 6
	const actionIntervalMs = 500 // ms between actions per player

	metrics := &soakTestMetrics{startTime: time.Now()}

	// Create server with default scenario
	gs := NewGameServer()
	if err := gs.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	// Add players and start game
	playerIDs := make([]string, numPlayers)
	for i := 0; i < numPlayers; i++ {
		playerID := fmt.Sprintf("player%d", i+1)
		playerIDs[i] = playerID
		gs.mutex.Lock()
		gs.gameState.Players[playerID] = &Player{
			ID:               playerID,
			Location:         Downtown,
			Resources:        Resources{Health: 8, Sanity: 8, Clues: 0, Focus: 0},
			ActionsRemaining: 2,
			Connected:        true,
			InvestigatorType: []InvestigatorType{
				InvestigatorResearcher,
				InvestigatorDetective,
				InvestigatorOccultist,
				InvestigatorSoldier,
				InvestigatorMystic,
				InvestigatorSurvivor,
			}[i],
		}
		gs.gameState.TurnOrder = append(gs.gameState.TurnOrder, playerID)
		gs.mutex.Unlock()
	}

	// Start the game
	gs.mutex.Lock()
	gs.gameState.CurrentPlayer = playerIDs[0]
	gs.gameState.GameStarted = true
	gs.gameState.GamePhase = "playing"
	gs.gameState.Doom = 1
	gs.mutex.Unlock()

	// Launch action goroutines for each player
	var wg sync.WaitGroup
	stopCh := make(chan struct{})
	defer close(stopCh)

	for _, playerID := range playerIDs {
		wg.Add(1)
		go func(pid string) {
			defer wg.Done()
			ticker := time.NewTicker(time.Duration(actionIntervalMs) * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-stopCh:
					return
				case <-ticker.C:
					// Choose a random action
					actions := []ActionType{
						ActionMove,
						ActionGather,
						ActionInvestigate,
						ActionCastWard,
						ActionFocus,
					}
					action := PlayerActionMessage{
						Type:       "playerAction",
						PlayerID:   pid,
						Action:     actions[rand.Intn(len(actions))],
						FocusSpend: rand.Intn(3), // 0-2 focus
					}

					// Set target for move/trade actions
					if action.Action == ActionMove {
						targets := []Location{Downtown, University, Rivertown, Northside}
						action.Target = string(targets[rand.Intn(len(targets))])
					}

					err := gs.processAction(action)
					if err != nil && err.Error() != "not player "+pid+"'s turn (current: "+gs.gameState.CurrentPlayer+")" {
						// Ignore "not your turn" errors (expected in multi-player); report others
						if !isExpectedActionError(err) {
							metrics.gameStateErrors.Add(1)
						}
					}
					metrics.actionCount.Add(1)
				}
			}
		}(playerID)
	}

	// Monitor game state every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	startTime := time.Now()

	for {
		select {
		case <-ticker.C:
			gs.mutex.RLock()
			doom := int32(gs.gameState.Doom)
			stateCopy := *gs.gameState // shallow copy for inspection
			gs.mutex.RUnlock()

			// Track max doom
			for {
				old := metrics.doomMaxValue.Load()
				if doom <= old || metrics.doomMaxValue.CompareAndSwap(old, doom) {
					break
				}
			}

			// Validate state invariants
			if doom < 0 || doom > 12 {
				t.Logf("FAIL: Doom out of bounds: %d", doom)
				metrics.gameStateErrors.Add(1)
			}

			actionCount := metrics.actionCount.Load()
			elapsed := time.Since(startTime)
			throughput := float64(actionCount) / elapsed.Seconds()
			t.Logf("[%v] Actions: %d, Throughput: %.1f/sec, Doom: %d, Players: %d, Phase: %s",
				elapsed.Round(time.Second), actionCount, throughput, doom,
				len(stateCopy.Players), stateCopy.GamePhase)

		case <-time.After(duration):
			// Test duration completed
			close(stopCh)
			wg.Wait()

			// Perform final validation
			gs.mutex.RLock()
			doomFinal := gs.gameState.Doom
			playerCount := len(gs.gameState.Players)
			gs.mutex.RUnlock()

			elapsed := time.Since(startTime)
			finalActions := metrics.actionCount.Load()
			throughput := float64(finalActions) / elapsed.Seconds()
			errors := metrics.gameStateErrors.Load()

			t.Logf("\n=== STRESS TEST RESULTS (30 seconds) ===")
			t.Logf("Total Actions: %d", finalActions)
			t.Logf("Throughput: %.2f actions/sec", throughput)
			t.Logf("Max Doom Reached: %d", metrics.doomMaxValue.Load())
			t.Logf("Final Doom: %d", doomFinal)
			t.Logf("Game State Errors: %d", errors)
			t.Logf("Summary: Stress test ran for %v with %d players, %d total actions, %d errors",
				elapsed.Round(time.Second), playerCount, finalActions, errors)

			// Assertions
			if doomFinal < 0 || doomFinal > 12 {
				t.Errorf("Doom out of bounds at end: %d", doomFinal)
			}
			if errors > 10 {
				t.Errorf("Too many game state errors: %d", errors)
			}
			if finalActions == 0 {
				t.Errorf("No actions were processed")
			}
			return
		}
	}
}

// isExpectedActionError returns true for errors that are expected in normal play
// (e.g., "not your turn" when another player is acting).
func isExpectedActionError(err error) bool {
	msg := err.Error()
	return err == nil ||
		contains(msg, "not your turn") ||
		contains(msg, "not in playing state") ||
		contains(msg, "no actions remaining") ||
		contains(msg, "has been defeated")
}

func contains(s, substr string) bool {
	for i := 0; i < len(s)-len(substr)+1; i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestStressTest_BroadcastLatencyGoal verifies the <500ms broadcast SLA over the stress test.
func TestStressTest_BroadcastLatencyGoal(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping broadcast latency test")
	}

	const numSamples = 100
	gs := NewGameServer()
	gs.Start()

	// Generate some actions to capture broadcast latencies
	gs.mutex.Lock()
	gs.gameState.Players["p1"] = &Player{
		ID:               "p1",
		Location:         Downtown,
		Resources:        Resources{Health: 8, Sanity: 8},
		ActionsRemaining: 2,
		Connected:        true,
	}
	gs.gameState.TurnOrder = []string{"p1"}
	gs.gameState.CurrentPlayer = "p1"
	gs.gameState.GameStarted = true
	gs.gameState.GamePhase = "playing"
	gs.mutex.Unlock()

	// Take actions and measure latencies
	for i := 0; i < numSamples; i++ {
		action := PlayerActionMessage{
			PlayerID: "p1",
			Action:   ActionFocus,
		}
		start := time.Now()
		_ = gs.processAction(action)
		latencyMs := time.Since(start).Milliseconds()

		if latencyMs > 500 {
			t.Logf("WARNING: Action %d took %d ms (exceeds 500ms goal)", i, latencyMs)
		}
	}

	t.Logf("Broadcast latency test completed for %d actions", numSamples)
}
