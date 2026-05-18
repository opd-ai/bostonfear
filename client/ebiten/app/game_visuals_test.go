package app

import (
	"testing"

	ebclient "github.com/opd-ai/bostonfear/client/ebiten"
)

func TestInitialsFromString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "two words", input: "Roland Carter", want: "R.C."},
		{name: "single word", input: "Mystic", want: "M.Y."},
		{name: "whitespace trimmed", input: "  Dana  Scully  ", want: "D.S."},
		{name: "empty", input: "   ", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := initialsFromString(tc.input); got != tc.want {
				t.Fatalf("initialsFromString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestInvestigatorInitials(t *testing.T) {
	if got := investigatorInitials(nil); got != "?" {
		t.Fatalf("investigatorInitials(nil) = %q, want ?", got)
	}

	player := &ebclient.Player{DisplayName: "Agent Ada", InvestigatorType: "researcher"}
	if got := investigatorInitials(player); got != "A.A." {
		t.Fatalf("investigatorInitials(display name) = %q, want %q", got, "A.A.")
	}

	player.DisplayName = ""
	if got := investigatorInitials(player); got != "R." {
		t.Fatalf("investigatorInitials(type) = %q, want %q", got, "R.")
	}
}
