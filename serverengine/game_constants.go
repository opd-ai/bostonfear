package serverengine

import "github.com/opd-ai/bostonfear/protocol"

// Player count limits matching AH3e core rulebook (1-6 investigators).
const (
	MinPlayers = 1 // Minimum players required to start a game
	MaxPlayers = 6 // Maximum concurrent players per game
)

const (
	Downtown   = protocol.Downtown
	University = protocol.University
	Rivertown  = protocol.Rivertown
	Northside  = protocol.Northside
)

const (
	ActionMove               = protocol.ActionMove
	ActionGather             = protocol.ActionGather
	ActionInvestigate        = protocol.ActionInvestigate
	ActionCastWard           = protocol.ActionCastWard
	ActionFocus              = protocol.ActionFocus
	ActionResearch           = protocol.ActionResearch
	ActionTrade              = protocol.ActionTrade
	ActionComponent          = protocol.ActionComponent
	ActionEncounter          = protocol.ActionEncounter
	ActionAttack             = protocol.ActionAttack
	ActionEvade              = protocol.ActionEvade
	ActionCloseGate          = protocol.ActionCloseGate
	ActionSelectInvestigator = protocol.ActionSelectInvestigator
	ActionSetDifficulty      = protocol.ActionSetDifficulty
	ActionSelectScenario     = protocol.ActionSelectScenario
	ActionChat               = protocol.ActionChat
)

const (
	InvestigatorResearcher = protocol.InvestigatorResearcher
	InvestigatorDetective  = protocol.InvestigatorDetective
	InvestigatorOccultist  = protocol.InvestigatorOccultist
	InvestigatorSoldier    = protocol.InvestigatorSoldier
	InvestigatorMystic     = protocol.InvestigatorMystic
	InvestigatorSurvivor   = protocol.InvestigatorSurvivor
)

// InvestigatorAbility describes the mechanical effect of a component action.
type InvestigatorAbility struct {
	Name        string // human-readable name shown in game updates
	Description string // flavour text for the client
	// Resource costs subtracted before applying the effect (zero means free).
	SanityCost int
	HealthCost int
	// Effect fields — only the relevant ones are non-zero for each archetype.
	ClueGain      int
	HealthGain    int
	SanityGain    int
	FocusGain     int
	DoomReduct    int  // positive value means doom decreases by this amount
	DrawEncounter bool // true → execute a free encounter card draw
}

// DefaultInvestigatorAbilities maps each InvestigatorType to its component ability.
// An unrecognised type falls back to InvestigatorSurvivor (safe default).
var DefaultInvestigatorAbilities = map[InvestigatorType]InvestigatorAbility{
	InvestigatorResearcher: {
		Name:        "Academic Insight",
		Description: "Your research uncovers a hidden clue without risking the dice.",
		ClueGain:    1,
	},
	InvestigatorDetective: {
		Name:          "Street Contacts",
		Description:   "You call in a favour and draw an encounter card at your location.",
		DrawEncounter: true,
	},
	InvestigatorOccultist: {
		Name:        "Dark Bargain",
		Description: "You sacrifice your sanity to push back the Ancient One's influence.",
		SanityCost:  2,
		DoomReduct:  1,
	},
	InvestigatorSoldier: {
		Name:        "Field Medic",
		Description: "Military training lets you patch yourself up on the fly.",
		SanityCost:  1,
		HealthGain:  2,
	},
	InvestigatorMystic: {
		Name:        "Arcane Focus",
		Description: "You channel the ley-lines and sharpen your concentration.",
		FocusGain:   1,
	},
	InvestigatorSurvivor: {
		Name:        "Grit",
		Description: "Sheer stubbornness restores a fragment of both body and mind.",
		HealthGain:  1,
		SanityGain:  1,
	},
}

// Resource bound constants for the extended AH3e resource vocabulary.
const (
	MaxHealth   = 10
	MaxSanity   = 10
	MaxClues    = 5
	MaxMoney    = 99
	MaxRemnants = 5
	MaxFocus    = 3
)

const (
	DiceSuccess  = protocol.DiceSuccess
	DiceBlank    = protocol.DiceBlank
	DiceTentacle = protocol.DiceTentacle
)

// maxEnemiesOnBoard caps the total number of active enemies to keep combat manageable.
const maxEnemiesOnBoard = 4

// enemyTemplates is the pool of enemy archetypes used by the spawn logic.
// On each spawn, one template is chosen at random and given a unique ID.
var enemyTemplates = []Enemy{
	{Name: "Ghoul", Health: 3, Damage: 1, Horror: 1},
	{Name: "Deep One", Health: 4, Damage: 2, Horror: 1},
	{Name: "Byakhee", Health: 2, Damage: 1, Horror: 2},
	{Name: "Shoggoth", Health: 5, Damage: 2, Horror: 2},
}

// defaultEncounterDecks returns 2-3 encounter cards per neighborhood for MVP play.
// Cards with new effect types (money_gain, focus_gain) demonstrate the extended effect system.
func defaultEncounterDecks() map[string][]EncounterCard {
	return map[string][]EncounterCard{
		string(Downtown): {
			{FlavorText: "A shadowy figure brushes past you — you feel drained.", EffectType: "sanity_loss", Magnitude: 1},
			{FlavorText: "You find a strange coin in an alley.", EffectType: "clue_gain", Magnitude: 1},
			{FlavorText: "A brief scuffle leaves you bruised.", EffectType: "health_loss", Magnitude: 1},
			{FlavorText: "A street vendor rewards your help.", EffectType: "money_gain", Magnitude: 2},
		},
		string(University): {
			{FlavorText: "Forbidden texts reveal a partial truth.", EffectType: "clue_gain", Magnitude: 1},
			{FlavorText: "A professor babbles incomprehensibly.", EffectType: "sanity_loss", Magnitude: 1},
			{FlavorText: "You stumble upon ritual components.", EffectType: "clue_gain", Magnitude: 2},
			{FlavorText: "A meditation session clears your mind.", EffectType: "focus_gain", Magnitude: 1},
		},
		string(Rivertown): {
			{FlavorText: "The dockworkers eye you with suspicion.", EffectType: "sanity_loss", Magnitude: 1},
			{FlavorText: "A fisherman warns of things below.", EffectType: "doom_inc", Magnitude: 1},
			{FlavorText: "You find a waterlogged journal.", EffectType: "clue_gain", Magnitude: 1},
			{FlavorText: "Salvaged goods prove valuable.", EffectType: "money_gain", Magnitude: 3},
		},
		string(Northside): {
			{FlavorText: "Whispering voices from the old asylum drain your resolve.", EffectType: "sanity_loss", Magnitude: 2},
			{FlavorText: "An escaped patient thrusts a torn map into your hands.", EffectType: "clue_gain", Magnitude: 1},
			{FlavorText: "A collapsing wall injures you.", EffectType: "health_loss", Magnitude: 1},
			{FlavorText: "Intense concentration sharpens your senses.", EffectType: "focus_gain", Magnitude: 2},
		},
	}
}

// Three acts; each requires progressively more collective clues.
func defaultActDeck() []ActCard {
	return []ActCard{
		{Title: "Act 1: Strange Disappearances", ClueThreshold: 4, Effect: "The investigators uncover a pattern of disappearances"},
		{Title: "Act 2: The Ritual", ClueThreshold: 8, Effect: "The ritual site is revealed"},
		{Title: "Act 3: The Final Sealing", ClueThreshold: 12, Effect: "The investigators seal the final gate — victory!"},
	}
}

// defaultAgendaDeck returns the default Agenda deck for a standard game.
// Three agendas; doom thresholds mirror AH3e default scenario.
func defaultAgendaDeck() []AgendaCard {
	return []AgendaCard{
		{Title: "Agenda 1: The Stars Align", DoomThreshold: 4, Effect: "The doom spreads further"},
		{Title: "Agenda 2: The Gate Opens", DoomThreshold: 8, Effect: "A gate to the void cracks open"},
		{Title: "Agenda 3: The Ancient One Awakens", DoomThreshold: 12, Effect: "The Ancient One awakens — investigators lose!"},
	}
}

var locationAdjacency = map[Location][]Location{
	Downtown:   {University, Rivertown},
	University: {Downtown, Northside},
	Rivertown:  {Downtown, Northside},
	Northside:  {University, Rivertown},
}

// Mythos cup token identifiers (AH3e §Mythos Phase).
const (
	MythosTokenDoom     = "doom"     // increment global doom by 1
	MythosTokenBlessing = "blessing" // heal 1 Health to current player
	MythosTokenCurse    = "curse"    // deal 1 Sanity to current player
	MythosTokenBlank    = "blank"    // no effect
)

// MythosEventAnomaly marks a MythosEvent as an anomaly-spawning event.
// When runMythosPhase processes an event whose Effect equals this constant,
// it calls spawnAnomaly at the target neighbourhood.
const MythosEventAnomaly = "anomaly"

// Additional Mythos event type constants.
// Each constant maps to a distinct mechanical resolution in resolveEventEffect.
const (
	MythosEventFogMadness  = "fog_madness"  // All investigators lose 1 Sanity
	MythosEventClueDrought = "clue_drought" // All investigators lose 1 Clue
	MythosEventDoomSpread  = "doom_spread"  // Doom +1 (per open gate once Step 10 is complete)
	MythosEventResurgence  = "resurgence"   // Each engaged enemy restores 1 Health
)

// DifficultySetup holds the initial game setup parameters per difficulty level.
type DifficultySetup struct {
	InitialDoom     int // doom counter starting value
	ExtraDoomTokens int // extra doom tokens added to MythosCup
}

// DifficultyConfig maps difficulty names to their AH3e setup parameters.
var DifficultyConfig = map[string]DifficultySetup{
	"easy":     {InitialDoom: 0, ExtraDoomTokens: 0},
	"standard": {InitialDoom: 1, ExtraDoomTokens: 1},
	"hard":     {InitialDoom: 3, ExtraDoomTokens: 3},
}

// defaultMythosEventDeck returns the starting event draw pile with a variety of
// event types spanning all four neighbourhoods. Events cycle when the deck empties.
func defaultMythosEventDeck() []MythosEvent {
	return []MythosEvent{
		{LocationID: string(Downtown), Effect: "Strange lights flicker in the streets", MythosEventType: MythosEventAnomaly},
		{LocationID: string(University), Effect: "Fog of Madness descends — all investigators lose 1 Sanity", MythosEventType: MythosEventFogMadness},
		{LocationID: string(Rivertown), Effect: "River runs black; clues wash away", MythosEventType: MythosEventClueDrought},
		{LocationID: string(Northside), Effect: "Doom spreads from the asylum", MythosEventType: MythosEventDoomSpread},
		{LocationID: string(Downtown), Effect: "Forbidden texts vanish; no new clues surface", MythosEventType: MythosEventClueDrought},
		{LocationID: string(University), Effect: "Ancient wards crumble — anomaly emerges", MythosEventType: MythosEventAnomaly},
		{LocationID: string(Rivertown), Effect: "Whispers of the deep drive investigators mad", MythosEventType: MythosEventFogMadness},
		{LocationID: string(Northside), Effect: "Monster resurgence — wounded creatures recover", MythosEventType: MythosEventResurgence},
	}
}

// DefaultScenario wraps the standard Arkham Horror 3e setup: 4 neighborhoods,
// doom starts at 0, act/agenda/encounter decks from defaults.
// Pass to NewGameServer to get standard play; substitute custom Scenario for variants.
var DefaultScenario = Scenario{
	Name:         "The Gathering",
	StartingDoom: 0,
	SetupFn: func(gs *GameState) {
		gs.Doom = 0
		gs.ActDeck = defaultActDeck()
		gs.AgendaDeck = defaultAgendaDeck()
		gs.MythosEventDeck = defaultMythosEventDeck()
		gs.EncounterDecks = defaultEncounterDecks()
		gs.LocationDoomTokens = make(map[string]int)
	},
	// nil WinFn/LoseFn: use the default deck-driven checks in checkGameEndConditions.
	WinFn:  nil,
	LoseFn: nil,
}
