package registry010

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPoPChallenge010(t *testing.T) {
	var fixture struct {
		ChallengeVectors []struct {
			Name            string `json:"name"`
			RegistryID      string `json:"registry_id"`
			AgentID         string `json:"agent_id"`
			Alg             string `json:"alg"`
			PublicKeyHex    string `json:"public_key_hex"`
			ChallengeHex    string `json:"challenge_hex"`
			ChallengeSHA256 string `json:"challenge_sha256"`
		} `json:"challenge_vectors"`
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "registry-proof-0.10.0.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.ChallengeVectors) != 2 {
		t.Fatal("expected signing and KEM challenge vectors")
	}
	for _, v := range fixture.ChallengeVectors {
		t.Run(v.Name, func(t *testing.T) {
			key, err := hex.DecodeString(v.PublicKeyHex)
			if err != nil {
				t.Fatal(err)
			}
			challenge, err := PoPChallenge010(v.RegistryID, v.AgentID, v.Name, v.Alg, key)
			if err != nil {
				t.Fatal(err)
			}
			if hex.EncodeToString(challenge) != v.ChallengeHex {
				t.Fatal("challenge bytes differ from 0.10.0 fixture")
			}
			digest := sha256.Sum256(challenge)
			if hex.EncodeToString(digest[:]) != v.ChallengeSHA256 {
				t.Fatal("challenge digest differs from 0.10.0 fixture")
			}
		})
	}
}

func TestPoPChallenge010RejectsUnencodableInput(t *testing.T) {
	for _, tc := range []struct {
		name, registry, agent, keyName, alg string
		key                                 []byte
	}{
		{"empty registry", "", "agent", "key", "ed25519", []byte{1}},
		{"non ASCII agent", "web:example.com", "agént", "key", "ed25519", []byte{1}},
		{"control byte", "web:example.com", "agent", "ke\ny", "ed25519", []byte{1}},
		{"length overflow", "web:example.com", "agent", "key", "ed25519", []byte(strings.Repeat("a", 65536))},
		{"empty key", "web:example.com", "agent", "key", "ed25519", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := PoPChallenge010(tc.registry, tc.agent, tc.keyName, tc.alg, tc.key); err == nil {
				t.Fatal("expected rejection")
			}
		})
	}
}
