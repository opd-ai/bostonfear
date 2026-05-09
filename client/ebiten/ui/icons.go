package ui

// IconID identifies a reusable UI icon semantic.
type IconID string

const (
	IconMove        IconID = "move"
	IconGather      IconID = "gather"
	IconInvestigate IconID = "investigate"
	IconWard        IconID = "ward"
	IconHealth      IconID = "health"
	IconSanity      IconID = "sanity"
	IconClues       IconID = "clues"
	IconDoom        IconID = "doom"
	IconTurn        IconID = "turn"
)

// IconRegistry maps icon semantics to display-safe glyph fallbacks.
type IconRegistry struct {
	items map[IconID]string
}

// NewIconRegistry creates an icon registry with default semantic icons.
func NewIconRegistry() *IconRegistry {
	return &IconRegistry{items: map[IconID]string{
		IconMove:        "M",
		IconGather:      "G",
		IconInvestigate: "I",
		IconWard:        "W",
		IconHealth:      "HP",
		IconSanity:      "SN",
		IconClues:       "CL",
		IconDoom:        "DOOM",
		IconTurn:        ">",
	}}
}

// Get returns the configured glyph label for an icon semantic.
func (r *IconRegistry) Get(id IconID) string {
	if r == nil || r.items == nil {
		return ""
	}
	return r.items[id]
}
