package rules

import (
	"testing"
)

func TestDefaultLocationSystem(t *testing.T) {
	ls := DefaultLocationSystem()
	if len(ls.Locations) != 6 {
		t.Errorf("DefaultLocationSystem() has %d locations, want 6", len(ls.Locations))
	}

	expectedLocations := []Location{
		LocationEntrance,
		LocationFoyer,
		LocationExhibitHall,
		LocationArchives,
		LocationLibrary,
		LocationBasement,
	}

	for i, expected := range expectedLocations {
		if ls.Locations[i] != expected {
			t.Errorf("DefaultLocationSystem() Locations[%d] = %v, want %v", i, ls.Locations[i], expected)
		}
	}
}

func TestIsValidLocation(t *testing.T) {
	ls := DefaultLocationSystem()

	tests := []struct {
		name     string
		location Location
		want     bool
	}{
		{"valid entrance", LocationEntrance, true},
		{"valid foyer", LocationFoyer, true},
		{"valid exhibit hall", LocationExhibitHall, true},
		{"valid archives", LocationArchives, true},
		{"valid library", LocationLibrary, true},
		{"valid basement", LocationBasement, true},
		{"invalid location", Location("invalid"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ls.IsValidLocation(tt.location)
			if got != tt.want {
				t.Errorf("IsValidLocation(%v) = %v, want %v", tt.location, got, tt.want)
			}
		})
	}
}

func TestCanMoveTo(t *testing.T) {
	ls := DefaultLocationSystem()

	tests := []struct {
		name string
		from Location
		to   Location
		want bool
	}{
		{"entrance to foyer", LocationEntrance, LocationFoyer, true},
		{"foyer to basement", LocationFoyer, LocationBasement, true},
		{"library to archives", LocationLibrary, LocationArchives, true},
		{"archives to entrance", LocationArchives, LocationEntrance, true},
		{"same location", LocationFoyer, LocationFoyer, true},
		{"invalid from location", Location("invalid"), LocationFoyer, false},
		{"invalid to location", LocationFoyer, Location("invalid"), false},
		{"both invalid", Location("invalid1"), Location("invalid2"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ls.CanMoveTo(tt.from, tt.to)
			if got != tt.want {
				t.Errorf("CanMoveTo(%v, %v) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestAllLocations(t *testing.T) {
	ls := DefaultLocationSystem()
	locations := ls.AllLocations()

	if len(locations) != 6 {
		t.Errorf("AllLocations() returned %d locations, want 6", len(locations))
	}

	// Verify it returns the actual slice
	if &locations[0] != &ls.Locations[0] {
		t.Errorf("AllLocations() returned a copy instead of the original slice")
	}
}

func TestLocationString(t *testing.T) {
	tests := []struct {
		name     string
		location Location
		want     string
	}{
		{"entrance", LocationEntrance, "entrance"},
		{"foyer", LocationFoyer, "foyer"},
		{"exhibit hall", LocationExhibitHall, "exhibitHall"},
		{"archives", LocationArchives, "archives"},
		{"library", LocationLibrary, "library"},
		{"basement", LocationBasement, "basement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.location.String()
			if got != tt.want {
				t.Errorf("Location.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocationConstants(t *testing.T) {
	tests := []struct {
		name     string
		location Location
		want     string
	}{
		{"Entrance", LocationEntrance, "entrance"},
		{"Foyer", LocationFoyer, "foyer"},
		{"ExhibitHall", LocationExhibitHall, "exhibitHall"},
		{"Archives", LocationArchives, "archives"},
		{"Library", LocationLibrary, "library"},
		{"Basement", LocationBasement, "basement"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.location) != tt.want {
				t.Errorf("%s constant = %v, want %v", tt.name, string(tt.location), tt.want)
			}
		})
	}
}
