package rules

import (
	"testing"
)

func TestDefaultResourceBounds(t *testing.T) {
	bounds := DefaultResourceBounds()
	if bounds.MinStamina != 1 {
		t.Errorf("DefaultResourceBounds() MinStamina = %d, want 1", bounds.MinStamina)
	}
	if bounds.MaxStamina != 8 {
		t.Errorf("DefaultResourceBounds() MaxStamina = %d, want 8", bounds.MaxStamina)
	}
	if bounds.MinSanity != 1 {
		t.Errorf("DefaultResourceBounds() MinSanity = %d, want 1", bounds.MinSanity)
	}
	if bounds.MaxSanity != 8 {
		t.Errorf("DefaultResourceBounds() MaxSanity = %d, want 8", bounds.MaxSanity)
	}
}

func TestValidateStamina(t *testing.T) {
	bounds := DefaultResourceBounds()

	tests := []struct {
		name    string
		stamina int
		wantErr error
	}{
		{"valid stamina", 5, nil},
		{"minimum stamina", 1, nil},
		{"maximum stamina", 8, nil},
		{"above maximum", 10, nil}, // Capping allowed
		{"below minimum", 0, ErrInsufficientStamina},
		{"negative", -1, ErrInsufficientStamina},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bounds.ValidateStamina(tt.stamina)
			if err != tt.wantErr {
				t.Errorf("ValidateStamina(%d) error = %v, want %v", tt.stamina, err, tt.wantErr)
			}
		})
	}
}

func TestValidateSanity(t *testing.T) {
	bounds := DefaultResourceBounds()

	tests := []struct {
		name    string
		sanity  int
		wantErr error
	}{
		{"valid sanity", 5, nil},
		{"minimum sanity", 1, nil},
		{"maximum sanity", 8, nil},
		{"above maximum", 10, nil}, // Capping allowed
		{"below minimum", 0, ErrInsufficientSanity},
		{"negative", -1, ErrInsufficientSanity},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := bounds.ValidateSanity(tt.sanity)
			if err != tt.wantErr {
				t.Errorf("ValidateSanity(%d) error = %v, want %v", tt.sanity, err, tt.wantErr)
			}
		})
	}
}

func TestClampStamina(t *testing.T) {
	bounds := DefaultResourceBounds()

	tests := []struct {
		name    string
		stamina int
		want    int
	}{
		{"within bounds", 5, 5},
		{"minimum", 1, 1},
		{"maximum", 8, 8},
		{"below minimum", 0, 1},
		{"above maximum", 10, 8},
		{"negative", -5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bounds.ClampStamina(tt.stamina)
			if got != tt.want {
				t.Errorf("ClampStamina(%d) = %d, want %d", tt.stamina, got, tt.want)
			}
		})
	}
}

func TestClampSanity(t *testing.T) {
	bounds := DefaultResourceBounds()

	tests := []struct {
		name   string
		sanity int
		want   int
	}{
		{"within bounds", 5, 5},
		{"minimum", 1, 1},
		{"maximum", 8, 8},
		{"below minimum", 0, 1},
		{"above maximum", 10, 8},
		{"negative", -5, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bounds.ClampSanity(tt.sanity)
			if got != tt.want {
				t.Errorf("ClampSanity(%d) = %d, want %d", tt.sanity, got, tt.want)
			}
		})
	}
}

func TestApplyResourceEvent(t *testing.T) {
	bounds := DefaultResourceBounds()

	tests := []struct {
		name        string
		stamina     int
		sanity      int
		event       ResourceEvent
		wantStamina int
		wantSanity  int
	}{
		{
			name:        "gain stamina",
			stamina:     5,
			sanity:      5,
			event:       ResourceEvent{StaminaDelta: 2, SanityDelta: 0},
			wantStamina: 7,
			wantSanity:  5,
		},
		{
			name:        "lose sanity",
			stamina:     5,
			sanity:      5,
			event:       ResourceEvent{StaminaDelta: 0, SanityDelta: -2},
			wantStamina: 5,
			wantSanity:  3,
		},
		{
			name:        "clamp at maximum",
			stamina:     7,
			sanity:      7,
			event:       ResourceEvent{StaminaDelta: 5, SanityDelta: 5},
			wantStamina: 8,
			wantSanity:  8,
		},
		{
			name:        "fatal stamina loss",
			stamina:     2,
			sanity:      5,
			event:       ResourceEvent{StaminaDelta: -3, SanityDelta: 0},
			wantStamina: 0,
			wantSanity:  0,
		},
		{
			name:        "fatal sanity loss",
			stamina:     5,
			sanity:      2,
			event:       ResourceEvent{StaminaDelta: 0, SanityDelta: -3},
			wantStamina: 0,
			wantSanity:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStamina, gotSanity, err := ApplyResourceEvent(tt.stamina, tt.sanity, tt.event, bounds)
			if err != nil {
				t.Errorf("ApplyResourceEvent() returned error: %v", err)
			}
			if gotStamina != tt.wantStamina {
				t.Errorf("ApplyResourceEvent() stamina = %d, want %d", gotStamina, tt.wantStamina)
			}
			if gotSanity != tt.wantSanity {
				t.Errorf("ApplyResourceEvent() sanity = %d, want %d", gotSanity, tt.wantSanity)
			}
		})
	}
}
