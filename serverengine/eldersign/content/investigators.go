package content

// Investigator represents an Elder Sign investigator with unique starting resources and abilities.
type Investigator struct {
	ID              string
	Name            string
	StartingStamina int // 1-8
	StartingSanity  int // 1-8
	StartingItems   []string
	SpecialAbility  string
	AbilityDesc     string
}

// DefaultInvestigators returns the base investigator roster for Elder Sign.
// Investigators overlap with Arkham Horror but have different resource ranges (Stamina/Sanity 1-8 vs Health/Sanity 1-10)
// and unique Elder Sign-specific abilities.
func DefaultInvestigators() []Investigator {
	return []Investigator{
		{
			ID:              "inv.roland.banks",
			Name:            "Roland Banks",
			StartingStamina: 7,
			StartingSanity:  5,
			StartingItems:   []string{"item.magnifyingglass"},
			SpecialAbility:  "federal_agent",
			AbilityDesc:     "When completing an investigation task, may reroll 1 die",
		},
		{
			ID:              "inv.daisy.walker",
			Name:            "Daisy Walker",
			StartingStamina: 5,
			StartingSanity:  7,
			StartingItems:   []string{"item.forbiddentome"},
			SpecialAbility:  "librarian",
			AbilityDesc:     "When completing a lore task, may reroll 1 die",
		},
		{
			ID:              "inv.agnes.baker",
			Name:            "Agnes Baker",
			StartingStamina: 6,
			StartingSanity:  6,
			StartingItems:   []string{"item.protectiveward"},
			SpecialAbility:  "waitress",
			AbilityDesc:     "When rolling terror dice, reduce terror count by 1",
		},
		{
			ID:              "inv.wendy.adams",
			Name:            "Wendy Adams",
			StartingStamina: 6,
			StartingSanity:  6,
			StartingItems:   []string{"item.luckycharm"},
			SpecialAbility:  "urchin",
			AbilityDesc:     "May discard 1 locked die and redraw it",
		},
		{
			ID:              "inv.skids.otoole",
			Name:            "\"Skids\" O'Toole",
			StartingStamina: 7,
			StartingSanity:  5,
			StartingItems:   []string{"item.ancientartifact"},
			SpecialAbility:  "ex_convict",
			AbilityDesc:     "May ignore 1 peril result per adventure",
		},
		{
			ID:              "inv.jim.culver",
			Name:            "Jim Culver",
			StartingStamina: 5,
			StartingSanity:  7,
			StartingItems:   []string{"item.blessedtalisman"},
			SpecialAbility:  "musician",
			AbilityDesc:     "After locking dice, may reroll all unlocked dice",
		},
		{
			ID:              "inv.harvey.walters",
			Name:            "Harvey Walters",
			StartingStamina: 5,
			StartingSanity:  8,
			StartingItems:   []string{"item.magnifyingglass", "item.forbiddentome"},
			SpecialAbility:  "professor",
			AbilityDesc:     "May lock 1 additional die per roll",
		},
		{
			ID:              "inv.zoey.samaras",
			Name:            "Zoey Samaras",
			StartingStamina: 8,
			StartingSanity:  4,
			StartingItems:   []string{"item.enchantedweapon"},
			SpecialAbility:  "chef",
			AbilityDesc:     "When facing terror, may reroll all terror dice once",
		},
		{
			ID:              "inv.norman.withers",
			Name:            "Norman Withers",
			StartingStamina: 4,
			StartingSanity:  8,
			StartingItems:   []string{"item.ancientartifact"},
			SpecialAbility:  "astronomer",
			AbilityDesc:     "At start of turn, may look at top 3 adventures and reorder",
		},
		{
			ID:              "inv.joe.diamond",
			Name:            "Joe Diamond",
			StartingStamina: 7,
			StartingSanity:  5,
			StartingItems:   []string{"item.magnifyingglass"},
			SpecialAbility:  "private_investigator",
			AbilityDesc:     "When gaining clues, gain 1 additional clue",
		},
		{
			ID:              "inv.carolyn.fern",
			Name:            "Carolyn Fern",
			StartingStamina: 5,
			StartingSanity:  7,
			StartingItems:   []string{"item.protectiveward"},
			SpecialAbility:  "psychologist",
			AbilityDesc:     "When gaining sanity, gain 1 additional sanity",
		},
		{
			ID:              "inv.mark.harrigan",
			Name:            "Mark Harrigan",
			StartingStamina: 8,
			StartingSanity:  4,
			StartingItems:   []string{"item.enchantedweapon"},
			SpecialAbility:  "soldier",
			AbilityDesc:     "When losing stamina, lose 1 less (minimum 0)",
		},
		{
			ID:              "inv.minh.thi.phan",
			Name:            "Minh Thi Phan",
			StartingStamina: 4,
			StartingSanity:  8,
			StartingItems:   []string{"item.forbiddentome"},
			SpecialAbility:  "secretary",
			AbilityDesc:     "At start of turn, may draw 1 additional item card",
		},
		{
			ID:              "inv.tony.morgan",
			Name:            "Tony Morgan",
			StartingStamina: 7,
			StartingSanity:  5,
			StartingItems:   []string{"item.luckycharm"},
			SpecialAbility:  "bounty_hunter",
			AbilityDesc:     "When defeating a monster, gain 1 Elder Sign token",
		},
		{
			ID:              "inv.amanda.sharpe",
			Name:            "Amanda Sharpe",
			StartingStamina: 5,
			StartingSanity:  7,
			StartingItems:   []string{"item.magnifyingglass"},
			SpecialAbility:  "student",
			AbilityDesc:     "May copy another investigator's ability once per turn",
		},
		{
			ID:              "inv.trish.scarborough",
			Name:            "Trish Scarborough",
			StartingStamina: 6,
			StartingSanity:  6,
			StartingItems:   []string{"item.ancientartifact"},
			SpecialAbility:  "spy",
			AbilityDesc:     "After rolling, may swap 1 die result with another investigator",
		},
	}
}

// InvestigatorByID retrieves an investigator by their unique identifier.
func InvestigatorByID(id string) (Investigator, bool) {
	investigators := DefaultInvestigators()
	for _, inv := range investigators {
		if inv.ID == id {
			return inv, true
		}
	}
	return Investigator{}, false
}

// ValidateInvestigator checks if an investigator's resources are within Elder Sign bounds.
func ValidateInvestigator(stamina, sanity int) bool {
	return stamina >= 1 && stamina <= 8 && sanity >= 1 && sanity <= 8
}
