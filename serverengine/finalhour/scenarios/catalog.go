package scenarios

import "fmt"

// Template defines a scenario blueprint for Final Hour module development.
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
			ID:       "finalhour.campus.lockdown",
			Name:     "Campus Lockdown",
			Enabled:  true,
			WinGoal:  "complete objectives before timer expires",
			LossGoal: "time track reaches zero",
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
	return Template{}, fmt.Errorf("no enabled final hour scenario template found")
}
