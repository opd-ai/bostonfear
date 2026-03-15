package main

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
