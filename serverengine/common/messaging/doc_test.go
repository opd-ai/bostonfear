package messaging

import (
	"errors"
	"testing"
)

type roundtripMessage struct {
	Type string `json:"type"`
	Doom int    `json:"doom"`
}

func TestEncodeDecodeJSONRoundTrip(t *testing.T) {
	in := roundtripMessage{Type: string(MessageGameState), Doom: 4}

	payload, err := EncodeJSON(in)
	if err != nil {
		t.Fatalf("EncodeJSON() error = %v", err)
	}

	var out roundtripMessage
	if err := DecodeJSON(payload, &out); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}

	if out != in {
		t.Fatalf("roundtrip mismatch: got %+v want %+v", out, in)
	}
}

func TestDecodeJSONMalformedPayload(t *testing.T) {
	var out roundtripMessage
	if err := DecodeJSON([]byte(`{"type":`), &out); err == nil {
		t.Fatal("DecodeJSON() error = nil, want malformed payload error")
	}
}

func TestDecodeJSONRequiresPointerTarget(t *testing.T) {
	if err := DecodeJSON([]byte(`{"type":"gameState"}`), roundtripMessage{}); !errors.Is(err, ErrDecodeTargetRequired) {
		t.Fatalf("DecodeJSON() error = %v, want %v", err, ErrDecodeTargetRequired)
	}
}
