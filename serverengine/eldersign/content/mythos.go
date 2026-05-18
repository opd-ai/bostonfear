package content

// MythosCard represents a museum-specific encounter or event during Elder Sign gameplay.
type MythosCard struct {
	ID          string
	Name        string
	Type        string // "encounter", "event", "omen"
	Effect      string
	DoomImpact  int // Doom increment (positive) or reduction (negative)
	ScenarioIDs []string
}

// DefaultMythosCards returns museum-specific encounter and event cards.
// These cards trigger during mythos phases and add narrative tension to gameplay.
func DefaultMythosCards() []MythosCard {
	return []MythosCard{
		// Encounter Cards - Direct investigator effects
		{
			ID:          "mythos.encounter.security",
			Name:        "Security Patrol",
			Type:        "encounter",
			Effect:      "All investigators lose 1 stamina",
			DoomImpact:  0,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.encounter.lights",
			Name:        "Flickering Lights",
			Type:        "encounter",
			Effect:      "All investigators lose 1 sanity",
			DoomImpact:  0,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.encounter.shadows",
			Name:        "Moving Shadows",
			Type:        "encounter",
			Effect:      "Current player loses 2 sanity",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.encounter.whispers",
			Name:        "Otherworldly Whispers",
			Type:        "encounter",
			Effect:      "Each investigator discards 1 item",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.encounter.cold",
			Name:        "Unnatural Cold",
			Type:        "encounter",
			Effect:      "All investigators lose 1 stamina and 1 sanity",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.cthulhu.depths"},
		},
		{
			ID:          "mythos.encounter.locked",
			Name:        "Locked Doors",
			Type:        "encounter",
			Effect:      "No investigators may move this turn",
			DoomImpact:  0,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.encounter.alarm",
			Name:        "Triggered Alarm",
			Type:        "encounter",
			Effect:      "Skip next player's turn",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.museum.nightwatch"},
		},
		{
			ID:          "mythos.encounter.exhibit",
			Name:        "Animated Exhibit",
			Type:        "encounter",
			Effect:      "Current player rolls 3 dice; for each terror, lose 1 stamina",
			DoomImpact:  0,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.yig.serpent"},
		},
		{
			ID:          "mythos.encounter.curse",
			Name:        "Ancient Curse",
			Type:        "encounter",
			Effect:      "Current player loses 2 stamina and 2 sanity",
			DoomImpact:  2,
			ScenarioIDs: []string{"eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.encounter.vision",
			Name:        "Horrifying Vision",
			Type:        "encounter",
			Effect:      "All investigators lose 2 sanity",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.azathoth.madness", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},

		// Event Cards - Game state changes
		{
			ID:          "mythos.event.power",
			Name:        "Power Failure",
			Type:        "event",
			Effect:      "All adventure difficulty increased by 1 until next mythos",
			DoomImpact:  0,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.event.fog",
			Name:        "Unnatural Fog",
			Type:        "event",
			Effect:      "Investigators cannot see other locations until next mythos",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.event.tremor",
			Name:        "Earthquake Tremor",
			Type:        "event",
			Effect:      "Discard top 3 adventure cards from deck",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.yig.serpent"},
		},
		{
			ID:          "mythos.event.moon",
			Name:        "Lunar Eclipse",
			Type:        "event",
			Effect:      "All dice rolls have +1 terror result until next mythos",
			DoomImpact:  2,
			ScenarioIDs: []string{"eldersign.azathoth.madness", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.event.gate",
			Name:        "Opening Gate",
			Type:        "event",
			Effect:      "Spawn 1 monster at random location",
			DoomImpact:  2,
			ScenarioIDs: []string{"eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},

		// Omen Cards - Major doom progression
		{
			ID:          "mythos.omen.awakening",
			Name:        "The Awakening Stirs",
			Type:        "omen",
			Effect:      "Ancient One awakening progress accelerates",
			DoomImpact:  3,
			ScenarioIDs: []string{"eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:          "mythos.omen.chaos",
			Name:        "Chaos Manifestation",
			Type:        "omen",
			Effect:      "All investigators lose 1 stamina and 1 sanity; doom increases by 3",
			DoomImpact:  3,
			ScenarioIDs: []string{"eldersign.azathoth.madness"},
		},
		{
			ID:          "mythos.omen.serpent",
			Name:        "Yig's Wrath",
			Type:        "omen",
			Effect:      "All investigators attacked by serpents; lose 2 stamina each",
			DoomImpact:  3,
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},
		{
			ID:          "mythos.omen.depths",
			Name:        "Rising from the Depths",
			Type:        "omen",
			Effect:      "All investigators lose 2 sanity; doom increases by 3",
			DoomImpact:  3,
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
		{
			ID:          "mythos.omen.yellow",
			Name:        "The Yellow Sign Spreads",
			Type:        "omen",
			Effect:      "All investigators lose 2 sanity; no items may be used until next mythos",
			DoomImpact:  3,
			ScenarioIDs: []string{"eldersign.hastur.king"},
		},

		// Scenario-specific thematic encounters
		{
			ID:          "mythos.azathoth.madness",
			Name:        "Madness at the Core",
			Type:        "encounter",
			Effect:      "Current player rolls 5 dice; for each peril, lose 1 sanity",
			DoomImpact:  2,
			ScenarioIDs: []string{"eldersign.azathoth.madness"},
		},
		{
			ID:          "mythos.yig.serpents",
			Name:        "Serpent Swarm",
			Type:        "encounter",
			Effect:      "All investigators at same location lose 2 stamina",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},
		{
			ID:          "mythos.cthulhu.water",
			Name:        "Water Infiltration",
			Type:        "encounter",
			Effect:      "All items become waterlogged; -1 effectiveness until dried",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
		{
			ID:          "mythos.hastur.mask",
			Name:        "The Pallid Mask Appears",
			Type:        "encounter",
			Effect:      "Current player loses 3 sanity",
			DoomImpact:  2,
			ScenarioIDs: []string{"eldersign.hastur.king"},
		},
		{
			ID:          "mythos.museum.artifact",
			Name:        "Cursed Artifact Activates",
			Type:        "event",
			Effect:      "Random adventure becomes difficulty 4 until claimed",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.museum.nightwatch"},
		},
		{
			ID:          "mythos.azathoth.void",
			Name:        "Void Breach",
			Type:        "event",
			Effect:      "Discard top 5 cards from adventure deck",
			DoomImpact:  2,
			ScenarioIDs: []string{"eldersign.azathoth.madness"},
		},
		{
			ID:          "mythos.yig.venom",
			Name:        "Venom Cloud",
			Type:        "event",
			Effect:      "All investigators lose 1 stamina; cannot gain stamina this turn",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},
		{
			ID:          "mythos.cthulhu.dream",
			Name:        "Cthulhu's Dream Call",
			Type:        "event",
			Effect:      "All investigators lose 1 sanity; no sanity may be gained this turn",
			DoomImpact:  1,
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
		{
			ID:          "mythos.hastur.symbol",
			Name:        "Yellow Symbol Appears",
			Type:        "event",
			Effect:      "Current player must discard all items or lose 4 sanity",
			DoomImpact:  2,
			ScenarioIDs: []string{"eldersign.hastur.king"},
		},
	}
}

// MythosForScenario filters mythos cards applicable to a specific scenario.
func MythosForScenario(scenarioID string) []MythosCard {
	allMythos := DefaultMythosCards()
	filtered := make([]MythosCard, 0)

	for _, mythos := range allMythos {
		for _, id := range mythos.ScenarioIDs {
			if id == scenarioID {
				filtered = append(filtered, mythos)
				break
			}
		}
	}

	return filtered
}
