// Package adapters defines broadcast payload shaping for Arkham Horror.
// S6 Migration: Broadcast payload shaping and mechanic events - arkhamhorror module ownership.
package adapters

import "github.com/opd-ai/bostonfear/serverengine/common/contracts"

// BroadcastPayloadAdapter is the canonical interface from serverengine/common/contracts.
// Re-exported here for use within the adapters package without requiring callers of
// NewBroadcastAdapter to import contracts directly.
type BroadcastPayloadAdapter = contracts.BroadcastPayloadAdapter

// ActionResultPayload is the arkhamhorror-owned shape for action results
type ActionResultPayload struct {
	Type      string      `json:"type"`
	PlayerID  string      `json:"playerId"`
	Event     string      `json:"event"`
	Result    string      `json:"result"`
	Timestamp interface{} `json:"timestamp"`
}

// DiceResultPayload is the arkhamhorror-owned shape for dice outcomes
type DiceResultPayload struct {
	Type         string        `json:"type"`
	PlayerID     string        `json:"playerId"`
	Action       string        `json:"action"`
	Results      []interface{} `json:"results"`
	Successes    int           `json:"successes"`
	Tentacles    int           `json:"tentacles"`
	Success      bool          `json:"success"`
	DoomIncrease int           `json:"doomIncrease"`
}

// S6: Message ordering and payload format is owned by arkhamhorror module
// to maintain consistency when other game families are implemented later.
// The adapter pattern allows swapping payload shapes without changing core serverengine.
