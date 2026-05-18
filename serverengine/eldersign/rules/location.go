package rules

// Location represents a museum room in Elder Sign.
// Unlike Arkham Horror's city neighborhoods with adjacency restrictions,
// all Elder Sign museum rooms are accessible from any position.
type Location string

const (
	// LocationEntrance is the museum entrance where investigators begin.
	LocationEntrance Location = "entrance"

	// LocationFoyer is the main museum foyer.
	LocationFoyer Location = "foyer"

	// LocationExhibitHall is the primary exhibit hall.
	LocationExhibitHall Location = "exhibitHall"

	// LocationArchives is the museum archives room.
	LocationArchives Location = "archives"

	// LocationLibrary is the museum library.
	LocationLibrary Location = "library"

	// LocationBasement is the museum basement storage area.
	LocationBasement Location = "basement"
)

// LocationSystem defines Elder Sign movement rules.
// Unlike Arkham Horror's restricted adjacency movement, Elder Sign allows
// investigators to move to any museum room from any other room.
type LocationSystem struct {
	// Locations is the list of all available museum rooms.
	Locations []Location
}

// DefaultLocationSystem returns the standard Elder Sign museum layout.
func DefaultLocationSystem() LocationSystem {
	return LocationSystem{
		Locations: []Location{
			LocationEntrance,
			LocationFoyer,
			LocationExhibitHall,
			LocationArchives,
			LocationLibrary,
			LocationBasement,
		},
	}
}

// IsValidLocation checks if a location exists in the system.
func (ls LocationSystem) IsValidLocation(loc Location) bool {
	for _, l := range ls.Locations {
		if l == loc {
			return true
		}
	}
	return false
}

// CanMoveTo checks if an investigator can move to the target location.
// In Elder Sign, all locations are accessible from any position (no adjacency rules).
func (ls LocationSystem) CanMoveTo(from, to Location) bool {
	// Validate both locations exist
	if !ls.IsValidLocation(from) || !ls.IsValidLocation(to) {
		return false
	}
	// No adjacency restrictions - all rooms are accessible
	return true
}

// AllLocations returns the complete list of museum rooms.
func (ls LocationSystem) AllLocations() []Location {
	return ls.Locations
}

// String converts Location to string for logging and serialization.
func (l Location) String() string {
	return string(l)
}
