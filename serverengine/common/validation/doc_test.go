package validation

import "testing"

func TestTurnCheckerIsLegal(t *testing.T) {
	checker := TurnChecker{
		GamePhase:        "playing",
		CurrentPlayer:    "player1",
		ActionsRemaining: 2,
		IsAllowedAction: func(actionType string) bool {
			return actionType == "gather"
		},
	}

	if err := checker.IsLegal("gather", "player1"); err != nil {
		t.Fatalf("IsLegal() error = %v, want nil", err)
	}
}

func TestTurnCheckerRejectsInvalidCases(t *testing.T) {
	cases := []struct {
		name    string
		checker TurnChecker
		action  string
		player  string
	}{
		{
			name: "wrong phase",
			checker: TurnChecker{
				GamePhase:        "setup",
				CurrentPlayer:    "player1",
				ActionsRemaining: 1,
			},
			action: "gather",
			player: "player1",
		},
		{
			name: "wrong player turn",
			checker: TurnChecker{
				GamePhase:        "playing",
				CurrentPlayer:    "player2",
				ActionsRemaining: 1,
			},
			action: "gather",
			player: "player1",
		},
		{
			name: "no actions remaining",
			checker: TurnChecker{
				GamePhase:        "playing",
				CurrentPlayer:    "player1",
				ActionsRemaining: 0,
			},
			action: "gather",
			player: "player1",
		},
		{
			name: "invalid action",
			checker: TurnChecker{
				GamePhase:        "playing",
				CurrentPlayer:    "player1",
				ActionsRemaining: 1,
				IsAllowedAction: func(actionType string) bool {
					return actionType == "gather"
				},
			},
			action: "invalid",
			player: "player1",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.checker.IsLegal(tc.action, tc.player); err == nil {
				t.Fatal("IsLegal() error = nil, want validation error")
			}
		})
	}
}
