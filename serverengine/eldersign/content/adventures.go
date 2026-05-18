// Package content provides Elder Sign adventure card templates and content packs.
package content

import "github.com/opd-ai/bostonfear/serverengine/eldersign/model"

// AdventureTemplate represents a reusable adventure card configuration.
type AdventureTemplate struct {
	ID          string
	Name        string
	Difficulty  int // 1=Easy, 2=Medium, 3=Hard, 4=VeryHard
	Tasks       []model.AdventureTask
	Rewards     []model.Reward
	Penalties   []model.Penalty
	ScenarioIDs []string // Which scenarios include this adventure
}

// DefaultAdventures returns the base set of adventure cards for Elder Sign gameplay.
// This catalog includes 30+ unique adventures spanning multiple difficulty levels and
// Ancient One scenarios.
func DefaultAdventures() []AdventureTemplate {
	return []AdventureTemplate{
		// Easy Adventures (Difficulty 1) - 8 cards
		{
			ID:         "adventure.museum.entrance",
			Name:       "Museum Entrance",
			Difficulty: 1,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2}, Description: "Search the lobby"},
			},
			Rewards: []model.Reward{
				{Type: "sanity", Value: 1},
				{Type: "stamina", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.museum.giftshop",
			Name:       "Gift Shop",
			Difficulty: 1,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 1, "investigation": 1}, Description: "Browse occult souvenirs"},
			},
			Rewards: []model.Reward{
				{Type: "item", ItemID: "item.magnifyingglass"},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.museum.reading_room",
			Name:       "Reading Room",
			Difficulty: 1,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 2}, Description: "Study ancient texts"},
			},
			Rewards: []model.Reward{
				{Type: "sanity", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.museum.cafe",
			Name:       "Museum Café",
			Difficulty: 1,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 1}, Description: "Investigate suspicious activity"},
			},
			Rewards: []model.Reward{
				{Type: "stamina", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.museum.courtyard",
			Name:       "Museum Courtyard",
			Difficulty: 1,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2}, Description: "Search the grounds"},
			},
			Rewards: []model.Reward{
				{Type: "clue", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.museum.storage",
			Name:       "Storage Room",
			Difficulty: 1,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 1, "lore": 1}, Description: "Examine stored artifacts"},
			},
			Rewards: []model.Reward{
				{Type: "item", ItemID: "item.ancientartifact"},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.museum.restoration",
			Name:       "Restoration Lab",
			Difficulty: 1,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 1, "investigation": 1}, Description: "Analyze restorations"},
			},
			Rewards: []model.Reward{
				{Type: "sanity", Value: 1},
				{Type: "clue", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.museum.security",
			Name:       "Security Office",
			Difficulty: 1,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2}, Description: "Review footage"},
			},
			Rewards: []model.Reward{
				{Type: "clue", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},

		// Medium Adventures (Difficulty 2) - 12 cards
		{
			ID:         "adventure.egypt.sarcophagus",
			Name:       "Egyptian Sarcophagus Hall",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 2, "investigation": 1}, Description: "Decipher hieroglyphics"},
				{RequiredResults: map[string]int{"peril": 1}, Description: "Face the guardian curse"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 1},
				{Type: "sanity", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.asia.temple",
			Name:       "Asian Temple Exhibit",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 2}, Description: "Study temple inscriptions"},
				{RequiredResults: map[string]int{"investigation": 2}, Description: "Search for hidden compartments"},
			},
			Rewards: []model.Reward{
				{Type: "item", ItemID: "item.blessedtalisman"},
				{Type: "sanity", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.yig.serpent", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.medieval.armor",
			Name:       "Medieval Armory",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 3}, Description: "Examine the display"},
				{RequiredResults: map[string]int{"terror": 1}, Description: "Confront animated armor"},
			},
			Rewards: []model.Reward{
				{Type: "item", ItemID: "item.enchantedweapon"},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 2},
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.oceanic.depths",
			Name:       "Oceanic Exhibit",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 1, "investigation": 2}, Description: "Study deep sea artifacts"},
			},
			Rewards: []model.Reward{
				{Type: "clue", Value: 2},
				{Type: "stamina", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.astronomy.planetarium",
			Name:       "Planetarium",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 2, "investigation": 1}, Description: "Observe star patterns"},
				{RequiredResults: map[string]int{"terror": 1}, Description: "Face cosmic horror"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 2},
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.natural.fossils",
			Name:       "Fossil Hall",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2, "lore": 1}, Description: "Examine prehistoric remains"},
			},
			Rewards: []model.Reward{
				{Type: "clue", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 1},
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.yig.serpent", "eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.library.restricted",
			Name:       "Restricted Section",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 3}, Description: "Access forbidden knowledge"},
			},
			Rewards: []model.Reward{
				{Type: "item", ItemID: "item.forbiddentome"},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.native.totems",
			Name:       "Native American Totems",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 2, "investigation": 1}, Description: "Interpret totem symbols"},
			},
			Rewards: []model.Reward{
				{Type: "item", ItemID: "item.protectiveward"},
				{Type: "stamina", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.yig.serpent"},
		},
		{
			ID:         "adventure.victorian.gallery",
			Name:       "Victorian Gallery",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2, "lore": 1}, Description: "Examine period pieces"},
			},
			Rewards: []model.Reward{
				{Type: "clue", Value: 2},
				{Type: "sanity", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.basement.furnace",
			Name:       "Basement Furnace Room",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2}, Description: "Investigate strange sounds"},
				{RequiredResults: map[string]int{"terror": 1}, Description: "Confront lurking horror"},
			},
			Rewards: []model.Reward{
				{Type: "stamina", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 2},
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.archives.manuscripts",
			Name:       "Archives - Manuscript Collection",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 3}, Description: "Translate ancient manuscripts"},
			},
			Rewards: []model.Reward{
				{Type: "clue", Value: 3},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.gems.minerals",
			Name:       "Gem and Mineral Hall",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2, "lore": 1}, Description: "Study crystalline formations"},
			},
			Rewards: []model.Reward{
				{Type: "item", ItemID: "item.enchantedgem"},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.yig.serpent", "eldersign.cthulhu.depths"},
		},

		// Hard Adventures (Difficulty 3) - 8 cards
		{
			ID:         "adventure.vault.sealed",
			Name:       "Sealed Vault",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 3}, Description: "Pick the lock"},
				{RequiredResults: map[string]int{"lore": 2, "peril": 1}, Description: "Bypass wards"},
				{RequiredResults: map[string]int{"terror": 1}, Description: "Face the guardian"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
				{Type: "item", ItemID: "item.ancientrelic"},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 2},
				{Type: "stamina", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.catacombs.entrance",
			Name:       "Catacombs Entrance",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 3, "lore": 1}, Description: "Navigate tunnels"},
				{RequiredResults: map[string]int{"terror": 2}, Description: "Face undead horrors"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 3},
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.yig.serpent"},
		},
		{
			ID:         "adventure.tower.observatory",
			Name:       "Tower Observatory",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 3, "investigation": 2}, Description: "Decode star charts"},
				{RequiredResults: map[string]int{"peril": 2}, Description: "Resist cosmic influence"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
				{Type: "clue", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 3},
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.ritual.chamber",
			Name:       "Ritual Chamber",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 4}, Description: "Complete protective ritual"},
				{RequiredResults: map[string]int{"terror": 1, "peril": 1}, Description: "Banish summoned entity"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 2},
				{Type: "sanity", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.serpent.shrine",
			Name:       "Serpent Shrine",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 2, "investigation": 2}, Description: "Appease the serpent god"},
				{RequiredResults: map[string]int{"terror": 2}, Description: "Survive serpent swarm"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
				{Type: "stamina", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 3},
				{Type: "doom", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},
		{
			ID:         "adventure.deep.portal",
			Name:       "Deep Sea Portal",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 3, "investigation": 2}, Description: "Close underwater rift"},
				{RequiredResults: map[string]int{"peril": 2}, Description: "Resist aquatic horrors"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 3},
				{Type: "doom", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.yellow.sign",
			Name:       "Hall of the Yellow Sign",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 4}, Description: "Deface the Yellow Sign"},
				{RequiredResults: map[string]int{"terror": 2}, Description: "Resist madness"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
				{Type: "sanity", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 4},
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.hastur.king"},
		},
		{
			ID:         "adventure.chaos.nexus",
			Name:       "Nexus of Chaos",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 3, "investigation": 2}, Description: "Stabilize reality"},
				{RequiredResults: map[string]int{"peril": 2, "terror": 1}, Description: "Survive chaos manifestation"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 3},
			},
			ScenarioIDs: []string{"eldersign.azathoth.madness"},
		},

		// Very Hard Adventures (Difficulty 4) - 4 cards
		{
			ID:         "adventure.elder.gate",
			Name:       "Elder Gate",
			Difficulty: 4,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 4, "investigation": 2}, Description: "Channel ancient power"},
				{RequiredResults: map[string]int{"peril": 3}, Description: "Withstand gate's influence"},
				{RequiredResults: map[string]int{"terror": 2}, Description: "Seal dimensional breach"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 3},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 3},
				{Type: "sanity", Value: 3},
				{Type: "stamina", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.ancient.sanctum",
			Name:       "Ancient Sanctum",
			Difficulty: 4,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 5}, Description: "Decipher elder prophecy"},
				{RequiredResults: map[string]int{"investigation": 3, "peril": 2}, Description: "Unlock inner sanctum"},
				{RequiredResults: map[string]int{"terror": 3}, Description: "Defeat the guardian"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 3},
				{Type: "item", ItemID: "item.elderkey"},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 3},
				{Type: "sanity", Value: 4},
			},
			ScenarioIDs: []string{"eldersign.museum.nightwatch", "eldersign.azathoth.madness", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.final.confrontation",
			Name:       "Final Confrontation",
			Difficulty: 4,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 4, "investigation": 3}, Description: "Prepare the ritual"},
				{RequiredResults: map[string]int{"peril": 3, "terror": 2}, Description: "Face the Ancient One"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 4},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 4},
				{Type: "sanity", Value: 3},
				{Type: "stamina", Value: 3},
			},
			ScenarioIDs: []string{"eldersign.azathoth.madness", "eldersign.yig.serpent", "eldersign.cthulhu.depths", "eldersign.hastur.king"},
		},
		{
			ID:         "adventure.void.threshold",
			Name:       "Threshold of the Void",
			Difficulty: 4,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 5, "investigation": 2}, Description: "Cross the void barrier"},
				{RequiredResults: map[string]int{"peril": 4}, Description: "Resist void corruption"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 3},
				{Type: "clue", Value: 3},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 4},
				{Type: "sanity", Value: 4},
			},
			ScenarioIDs: []string{"eldersign.azathoth.madness", "eldersign.hastur.king"},
		},

		// Additional Yig-specific adventures
		{
			ID:         "adventure.yig.jungle",
			Name:       "Jungle Temple Ruins",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2, "lore": 1}, Description: "Explore overgrown ruins"},
			},
			Rewards: []model.Reward{
				{Type: "clue", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},
		{
			ID:         "adventure.yig.scales",
			Name:       "Chamber of Scales",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 2}, Description: "Decipher serpent hieroglyphs"},
			},
			Rewards: []model.Reward{
				{Type: "item", ItemID: "item.serpentscale"},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},
		{
			ID:         "adventure.yig.venom",
			Name:       "Venom Pool",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2}, Description: "Extract antivenom"},
			},
			Rewards: []model.Reward{
				{Type: "stamina", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},
		{
			ID:         "adventure.yig.nest",
			Name:       "Serpent Nest",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 3}, Description: "Navigate the nest"},
				{RequiredResults: map[string]int{"terror": 1}, Description: "Fight serpent guardians"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 2},
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},
		{
			ID:         "adventure.yig.coils",
			Name:       "Coils of Yig",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 3}, Description: "Break the serpent's grip"},
				{RequiredResults: map[string]int{"peril": 1}, Description: "Resist crushing force"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 3},
			},
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},
		{
			ID:         "adventure.yig.altar",
			Name:       "Serpent God's Altar",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 3, "investigation": 1}, Description: "Desecrate the altar"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
				{Type: "clue", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.yig.serpent"},
		},

		// Additional Cthulhu-specific adventures
		{
			ID:         "adventure.cthulhu.abyss",
			Name:       "Abyssal Trench",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2, "lore": 1}, Description: "Explore deep waters"},
			},
			Rewards: []model.Reward{
				{Type: "clue", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.cthulhu.shipwreck",
			Name:       "Sunken Shipwreck",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 3}, Description: "Salvage the wreckage"},
			},
			Rewards: []model.Reward{
				{Type: "item", ItemID: "item.waterloggedtome"},
			},
			Penalties: []model.Penalty{
				{Type: "stamina", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.cthulhu.reef",
			Name:       "Coral Reef Labyrinth",
			Difficulty: 2,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"investigation": 2}, Description: "Navigate coral passages"},
			},
			Rewards: []model.Reward{
				{Type: "stamina", Value: 2},
			},
			Penalties: []model.Penalty{
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.cthulhu.city",
			Name:       "Sunken City Ruins",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 3}, Description: "Decipher R'lyeh inscriptions"},
				{RequiredResults: map[string]int{"terror": 1}, Description: "Face deep one guardian"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 2},
				{Type: "doom", Value: 1},
			},
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.cthulhu.monolith",
			Name:       "Cyclopean Monolith",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 2, "investigation": 2}, Description: "Study alien geometry"},
			},
			Rewards: []model.Reward{
				{Type: "clue", Value: 3},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 2},
			},
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
		{
			ID:         "adventure.cthulhu.dreams",
			Name:       "Chamber of Dreams",
			Difficulty: 3,
			Tasks: []model.AdventureTask{
				{RequiredResults: map[string]int{"lore": 3}, Description: "Resist Cthulhu's dreams"},
				{RequiredResults: map[string]int{"peril": 2}, Description: "Break psychic connection"},
			},
			Rewards: []model.Reward{
				{Type: "elderSign", Value: 1},
			},
			Penalties: []model.Penalty{
				{Type: "sanity", Value: 3},
			},
			ScenarioIDs: []string{"eldersign.cthulhu.depths"},
		},
	}
}

// AdventuresForScenario filters adventures applicable to a specific scenario.
func AdventuresForScenario(scenarioID string) []AdventureTemplate {
	allAdventures := DefaultAdventures()
	filtered := make([]AdventureTemplate, 0)

	for _, adv := range allAdventures {
		for _, id := range adv.ScenarioIDs {
			if id == scenarioID {
				filtered = append(filtered, adv)
				break
			}
		}
	}

	return filtered
}
