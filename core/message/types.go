package message

import "time"


type MessageControlHeader struct{
	// Sequence is an ever‑increasing packet counter
    Sequence uint64 	`json:"sequence"`
    // Nonce is a one‑time random value to prevent replay
    Nonce string 		`json:"nonce"`
    // Timestamp records when this packet was generated
    Timestamp time.Time `json:"timestamp"`
}

type BaseMessage struct {
	ContextID       string      `json:"-"`
	SessionID       string      `json:"id"`
    EphemeralPubKey []byte      `json:"ephemeralPubKey,omitempty"`
	DID             string 		`json:"did,omitempty"`
}

type ControlHeader interface {
    GetNonce() string  
    GetTimestamp() time.Time
	GetSequence() uint64
}