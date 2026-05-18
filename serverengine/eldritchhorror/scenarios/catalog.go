package scenarios

import "fmt"

// Template defines a scenario blueprint for Eldritch Horror module development.
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
			ID:       "eldritchhorror.world.omens-of-depth",
			Name:     "Omens of the Deep",
			Enabled:  true,
			WinGoal:  "solve mysteries across world map",
			LossGoal: "doom reaches awakening threshold",
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
	return Template{}, fmt.Errorf("no enabled eldritch horror scenario template found")
}
