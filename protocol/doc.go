// Package protocol defines the shared JSON wire contract used by the Go server and Go/JavaScript clients.
//
// This package owns all data transfer objects (DTOs) and message types that traverse the WebSocket,
// enabling type-safe serialization and reducing schema drift as the protocol evolves.
//
// Core Concepts:
//
// Location System: Four interconnected neighborhoods (Downtown, University, Rivertown, Northside)
// with movement restricted to adjacent areas only. Represented by Location type alias.
//
// Actions: Twelve player-initiated actions (Move, Gather, Investigate, etc.) defined by ActionType,
// each with different resource costs, dice thresholds, and effects.
//
// Dice Resolution: Three-sided outcomes (Success, Blank, Tentacle) that determine action success;
// Tentacles always increment the global doom counter unconditionally.
//
// Resources: Six resource types tracked per investigator:
//   - Health (1-10): Defeat if dropped below 1
//   - Sanity (1-10): Defeat if dropped below 1
//   - Clues (0-5): Gained by successful Investigate actions; capped at 5 per investigator
//   - Money (0-∞): Unlimited resource gained by Research and Trade
//   - Remnants (0-∞): Overtime resources from certain encounters
//   - Focus (0-∞): Action economy enhancer spent to add dice or reroll blanks
//
// Message Types:
//
// - PlayerActionMessage: Client → Server; player requests an action
// - GameState: Server → All Clients; full game snapshot (locations, players, doom, resources)
// - GameUpdateMessage: Server → All Clients; lightweight delta showing action outcome
// - DiceResultMessage: Server → Client; dice roll breakdown (successes, tentacles, outcome)
// - ConnectionStatus: Server → Client; connection quality metrics and ping/pong
//
// Classes of DTOs:
//
// - Message envelope types (Message, GameUpdateMessage, ConnectionStatusMessage)
// - Game-state types (Player, GameState, Enemy, Gate, Anomaly)
// - Content types (ActCard, AgendaCard, EncounterCard, MythosEvent)
// - Resource and metric types (Resources, ResourcesDelta, Player)
//
// Design Principles:
//
// 1. Immutability: All types are read-pass or JSON-serializable; mutation happens on the server only
// 2. Namespace Unity: Keeping protocol types in one package prevents go-stats-generator duplication alerts
// 3. Backward Compat: New fields are added with json:"field,omitempty" to avoid breaking older clients
//
// Integration:
//
// The serverengine package re-exports protocol types as type aliases (game_types.go)
// so internal code uses consistent naming (e.g., Location, Resources) without import noise.
// This decoupling enables testing of game rules in isolation and simplifies mock setup.
package protocol
