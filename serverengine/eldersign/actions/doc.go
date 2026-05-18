// Package actions implements Elder Sign action dispatching and validation.
//
// This package owns the routing logic for Elder Sign-specific actions
// (PlaceInvestigator, RollDice, LockDie, DiscardItem, ClaimAdventure)
// without importing serverengine, enabling testability through callback injection.
//
// The DispatchAction function is the entry point for all Elder Sign actions,
// routing each action type to its corresponding callback implementation provided
// by the serverengine GameServer.
package actions
