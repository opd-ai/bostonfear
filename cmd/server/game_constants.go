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
)

// DiceResult constants define the 3-sided dice outcomes
// Moved from: main.go
const (
	DiceSuccess  DiceResult = "success"
	DiceBlank    DiceResult = "blank"
	DiceTentacle DiceResult = "tentacle"
)

// locationAdjacency defines movement restrictions between locations
// Moved from: main.go
var locationAdjacency = map[Location][]Location{
	Downtown:   {University, Rivertown},
	University: {Downtown, Northside},
	Rivertown:  {Downtown, Northside},
	Northside:  {University, Rivertown},
}
