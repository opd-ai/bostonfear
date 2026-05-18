package scenarios

import "fmt"

// Template defines a scenario blueprint for Elder Sign module development.
type Template struct {
	ID       string
	Name     string
	Enabled  bool
	WinGoal  string
	LossGoal string
}

// DefaultCatalog returns starter scenarios for Elder Sign gameplay.
// Each scenario features a different Ancient One with unique mechanics and difficulty.
func DefaultCatalog() []Template {
	return []Template{
		{
			ID:       "eldersign.museum.nightwatch",
			Name:     "Nightwatch at the Museum",
			Enabled:  true,
			WinGoal:  "seal portals with elder signs",
			LossGoal: "ancient one awakens",
		},
		{
			ID:       "eldersign.azathoth.madness",
			Name:     "Azathoth: The Nuclear Chaos",
			Enabled:  true,
			WinGoal:  "collect 11 Elder Signs before doom reaches 12",
			LossGoal: "Azathoth awakens (doom reaches 12)",
		},
		{
			ID:       "eldersign.yig.serpent",
			Name:     "Yig: Father of Serpents",
			Enabled:  true,
			WinGoal:  "collect 10 Elder Signs before doom reaches 12",
			LossGoal: "Yig awakens and curses all investigators",
		},
		{
			ID:       "eldersign.cthulhu.depths",
			Name:     "Cthulhu: The Dreamer in R'lyeh",
			Enabled:  true,
			WinGoal:  "collect 12 Elder Signs before Cthulhu rises",
			LossGoal: "Cthulhu awakens from eternal slumber",
		},
		{
			ID:       "eldersign.hastur.king",
			Name:     "Hastur: The King in Yellow",
			Enabled:  true,
			WinGoal:  "collect 10 Elder Signs and close the Yellow Sign gate",
			LossGoal: "The Yellow Sign completes and Hastur manifests",
		},
	}
}

// ResolveDefault returns the first enabled template in catalog order.
func ResolveDefault(catalog []Template) (Template, error) {
	for _, item := range catalog {
		if item.Enabled {
			return item, nil
		}
	}
	return Template{}, fmt.Errorf("no enabled elder sign scenario template found")
}
