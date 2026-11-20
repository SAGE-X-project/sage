// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package rfc9421

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"testing"
	"time"
)

// signedMessage returns a correctly signed SAGE Message and its key.
func signedMessage(t *testing.T) (ed25519.PublicKey, *Message) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	msg := &Message{
		AgentDID: "did:sage:ethereum:0xa11ce", MessageID: "m-1", Timestamp: time.Now(), Nonce: "nonce-0123456789",
		Body: []byte(`{"tool":"get_weather"}`), Algorithm: string(AlgorithmEdDSA), KeyID: "k1",
		SignedFields: []string{"agent_did", "message_id", "timestamp", "nonce", "body"},
	}
	msg.Signature = ed25519.Sign(priv, []byte(NewVerifier().ConstructSignatureBase(msg)))
	return pub, msg
}

// TestMessageFormatIsRefused: the SAGE Message format is not part of
// 0.9.0-draft.1.1 and MUST NOT be accepted (E03-19); its verifier is
// forgeable by field aliasing (C1). Every verification entry refuses, even
// a correctly signed message (W-010, decision D25).
func TestMessageFormatIsRefused(t *testing.T) {
	pub, msg := signedMessage(t)
	v := NewVerifier()
	if err := v.VerifySignature(pub, msg, nil); !errors.Is(err, ErrMessageFormatRemoved) {
		t.Fatalf("VerifySignature of a correctly signed message: %v", err)
	}
	res, err := v.VerifyWithMetadata(pub, msg, nil, nil, DefaultVerificationOptions())
	if !errors.Is(err, ErrMessageFormatRemoved) {
		t.Fatalf("VerifyWithMetadata error: %v", err)
	}
	if res == nil || res.Valid || res.Error == "" {
		t.Fatalf("VerifyWithMetadata result must say invalid too: %+v", res)
	}
}

// TestC1ForgeryIsRefused replays the C1 probe of the security review: the
// signed values are copied into Headers, SignedFields names the header
// aliases, the body is replaced and a fresh nonce is set.
func TestC1ForgeryIsRefused(t *testing.T) {
	pub, msg := signedMessage(t)
	forged := *msg
	forged.Headers = map[string]string{
		"agent_did": msg.AgentDID, "message_id": msg.MessageID,
		"timestamp": msg.Timestamp.Format(time.RFC3339Nano), "nonce": msg.Nonce, "body": string(msg.Body),
	}
	forged.SignedFields = []string{"header.agent_did", "header.message_id", "header.timestamp", "header.nonce", "header.body"}
	forged.Body = []byte(`{"tool":"delete_all_files","path":"/"}`)
	forged.Nonce = "fresh-nonce-9876543210"
	if err := NewVerifier().VerifySignature(pub, &forged, nil); !errors.Is(err, ErrMessageFormatRemoved) {
		t.Fatalf("forged message: %v", err)
	}
}
