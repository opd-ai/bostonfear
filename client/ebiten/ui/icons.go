package ui

// IconID identifies a reusable UI icon semantic.
type IconID string

const (
	IconMove        IconID = "move"
	IconDowntown    IconID = "downtown"
	IconUniversity  IconID = "university"
	IconRivertown   IconID = "rivertown"
	IconNorthside   IconID = "northside"
	IconGather      IconID = "gather"
	IconInvestigate IconID = "investigate"
	IconWard        IconID = "ward"
	IconFocus       IconID = "focus"
	IconResearch    IconID = "research"
	IconTrade       IconID = "trade"
	IconComponent   IconID = "component"
	IconAttack      IconID = "attack"
	IconEvade       IconID = "evade"
	IconCloseGate   IconID = "closegate"
	IconEncounter   IconID = "encounter"
	IconDifficulty  IconID = "difficulty"
	IconConnection  IconID = "connection"
	IconHealth      IconID = "health"
	IconSanity      IconID = "sanity"
	IconClues       IconID = "clues"
	IconDoom        IconID = "doom"
	IconTurn        IconID = "turn"
	IconArrowLeft   IconID = "arrow-left"
	IconArrowRight  IconID = "arrow-right"
	IconCameraTop   IconID = "camera-top"
	IconCamera3D    IconID = "camera-3d"
	IconPlayer      IconID = "player"
)

// IconSpec describes a semantic icon's fallback glyph and visual sizing metadata.
type IconSpec struct {
	Glyph    string
	SizePx   int
	StrokePx float64
}

// IconRegistry maps icon semantics to display-safe glyph fallbacks.
type IconRegistry struct {
	items map[IconID]IconSpec
}

// NewIconRegistry creates an icon registry with default semantic icons.
func NewIconRegistry() *IconRegistry {
	mk := func(glyph string) IconSpec {
		return IconSpec{Glyph: glyph, SizePx: 32, StrokePx: 2.0}
	}
	return &IconRegistry{items: map[IconID]IconSpec{
		IconMove:        mk("M"),
		IconDowntown:    mk("DT"),
		IconUniversity:  mk("UN"),
		IconRivertown:   mk("RV"),
		IconNorthside:   mk("NS"),
		IconGather:      mk("G"),
		IconInvestigate: mk("I"),
		IconWard:        mk("W"),
		IconFocus:       mk("F"),
		IconResearch:    mk("R"),
		IconTrade:       mk("T"),
		IconComponent:   mk("C"),
		IconAttack:      mk("A"),
		IconEvade:       mk("E"),
		IconCloseGate:   mk("X"),
		IconEncounter:   mk("N"),
		IconDifficulty:  mk("*"),
		IconConnection:  mk("@"),
		IconHealth:      mk("HP"),
		IconSanity:      mk("SN"),
		IconClues:       mk("CL"),
		IconDoom:        mk("DO"),
		IconTurn:        mk(">"),
		IconArrowLeft:   mk("<"),
		IconArrowRight:  mk(">"),
		IconCameraTop:   mk("TD"),
		IconCamera3D:    mk("3D"),
		IconPlayer:      mk("P"),
	}}
}

// Get returns the configured glyph label for an icon semantic.
func (r *IconRegistry) Get(id IconID) string {
	if r == nil || r.items == nil {
		return ""
	}
	return r.items[id].Glyph
}

// Spec returns full rendering metadata for an icon semantic.
func (r *IconRegistry) Spec(id IconID) IconSpec {
	if r == nil || r.items == nil {
		return IconSpec{}
	}
	return r.items[id]
}

// Count returns the number of registered icon semantics.
func (r *IconRegistry) Count() int {
	if r == nil || r.items == nil {
		return 0
	}
	return len(r.items)
}
