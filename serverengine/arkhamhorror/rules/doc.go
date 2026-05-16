// Package rules owns all Arkham Horror rule logic extracted from serverengine.GameServer:
// location adjacency validation via IsAdjacent (delegates to arkhamhorror/content).
//
// Migration note: spell casting thresholds, investigation success constants, and
// ward rules currently reside in the serverengine monolith and will be migrated here
// in a future phase. Pure rule functions accept game state and return validation
// errors or effect results to enable offline rule testing and easier debugging.
package rules
