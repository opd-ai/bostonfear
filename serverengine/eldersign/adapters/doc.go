// Package adapters translates between Elder Sign-specific game events and the shared
// runtime contracts in serverengine/common/contracts.
//
// The BroadcastPayloadAdapter implementation shapes Elder Sign-specific messages
// (6-sided dice results, adventure cards, museum rooms) for wire protocol transmission.
package adapters
