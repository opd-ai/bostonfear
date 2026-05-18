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

// DefaultCatalog returns starter placeholders used by integration scaffolding.
func DefaultCatalog() []Template {
	return []Template{
		{
			ID:       "eldersign.museum.nightwatch",
			Name:     "Nightwatch at the Museum",
			Enabled:  true,
			WinGoal:  "seal portals with elder signs",
			LossGoal: "ancient one awakens",
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
