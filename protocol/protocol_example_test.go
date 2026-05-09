package protocol_test

import (
	"encoding/json"
	"fmt"

	"github.com/opd-ai/bostonfear/protocol"
)

func ExamplePlayerActionMessage() {
	msg := protocol.PlayerActionMessage{
		Type:     "playerAction",
		PlayerID: "player1",
		Action:   protocol.ActionInvestigate,
		Target:   string(protocol.University),
	}

	data, _ := json.Marshal(msg)
	fmt.Println(string(data))
	// Output: {"type":"playerAction","playerId":"player1","action":"investigate","target":"University"}
}

func ExampleGameState() {
	state := protocol.GameState{
		CurrentPlayer: "player2",
		Doom:          5,
		GamePhase:     "playing",
	}
	envelope := protocol.Message{Type: "gameState", Data: state}

	data, _ := json.Marshal(envelope)
	var decoded protocol.Message
	_ = json.Unmarshal(data, &decoded)

	fmt.Println(decoded.Type)
	// Output: gameState
}
