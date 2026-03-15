package main

// clientDir is the path to the client assets directory, relative to cmd/server/.
// Both the static file handler and the dashboard handler use this constant so
// that a single change keeps them in sync.
const clientDir = "../client"

// Player count limits matching AH3e core rulebook (1-6 investigators).
const (
	MinPlayers = 1 // Minimum players required to start a game
	MaxPlayers = 6 // Maximum concurrent players per game
)

// Location constants define the 4 interconnected neighborhoods
// Moved from: main.go
const (
	Downtown   Location = "Downtown"
	University Location = "University"
	Rivertown  Location = "Rivertown"
	Northside  Location = "Northside"
)

// ActionType constants define the 4 available actions per turn
// Moved from: main.go
const (
	ActionMove        ActionType = "move"
	ActionGather      ActionType = "gather"
	ActionInvestigate ActionType = "investigate"
	ActionCastWard    ActionType = "ward"
	ActionFocus       ActionType = "focus"
	ActionResearch    ActionType = "research"
	ActionTrade       ActionType = "trade"
	ActionComponent   ActionType = "component"
	ActionEncounter   ActionType = "encounter"
)

// Resource bound constants for the extended AH3e resource vocabulary.
const (
	MaxHealth   = 10
	MaxSanity   = 10
	MaxClues    = 5
	MaxMoney    = 99
	MaxRemnants = 5
	MaxFocus    = 3
)

// Moved from: main.go
const (
	DiceSuccess  DiceResult = "success"
	DiceBlank    DiceResult = "blank"
	DiceTentacle DiceResult = "tentacle"
)

// defaultEncounterDecks returns 2-3 encounter cards per neighborhood for MVP play.
func defaultEncounterDecks() map[string][]EncounterCard {
	return map[string][]EncounterCard{
		string(Downtown): {
			{FlavorText: "A shadowy figure brushes past you — you feel drained.", EffectType: "sanity_loss", Magnitude: 1},
			{FlavorText: "You find a strange coin in an alley.", EffectType: "clue_gain", Magnitude: 1},
			{FlavorText: "A brief scuffle leaves you bruised.", EffectType: "health_loss", Magnitude: 1},
		},
		string(University): {
			{FlavorText: "Forbidden texts reveal a partial truth.", EffectType: "clue_gain", Magnitude: 1},
			{FlavorText: "A professor babbles incomprehensibly.", EffectType: "sanity_loss", Magnitude: 1},
			{FlavorText: "You stumble upon ritual components.", EffectType: "clue_gain", Magnitude: 2},
		},
		string(Rivertown): {
			{FlavorText: "The dockworkers eye you with suspicion.", EffectType: "sanity_loss", Magnitude: 1},
			{FlavorText: "A fisherman warns of things below.", EffectType: "doom_inc", Magnitude: 1},
			{FlavorText: "You find a waterlogged journal.", EffectType: "clue_gain", Magnitude: 1},
		},
		string(Northside): {
			{FlavorText: "Whispering voices from the old asylum drain your resolve.", EffectType: "sanity_loss", Magnitude: 2},
			{FlavorText: "An escaped patient thrusts a torn map into your hands.", EffectType: "clue_gain", Magnitude: 1},
			{FlavorText: "A collapsing wall injures you.", EffectType: "health_loss", Magnitude: 1},
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

// Moved from: main.go
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

// defaultMythosEventDeck returns the starting event draw pile with one event
// per neighborhood.  Events cycle: when the deck empties it is rebuilt.
func defaultMythosEventDeck() []MythosEvent {
	return []MythosEvent{
		{LocationID: string(Downtown), Effect: "Strange lights flicker in the streets", Spread: false},
		{LocationID: string(University), Effect: "Forbidden texts vanish from the library", Spread: false},
		{LocationID: string(Rivertown), Effect: "River runs black with ichor", Spread: false},
		{LocationID: string(Northside), Effect: "Whispers from the old asylum grow louder", Spread: false},
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
