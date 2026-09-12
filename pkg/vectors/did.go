package vectors

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

func didSuite() Suite {
	valid := []string{
		"did:sage:ethereum:0x1111111111111111111111111111111111111111",
		"did:sage:ethereum:0x1111111111111111111111111111111111111111:42",
		"did:sage:solana:7Zn3QwXk9vQ2yP1sJ4cR8tL6mN5bV3xA2dF9gH1kJ8Lm",
		"did:sage:ETHEREUM:agent-one",
		"did:sage:eth:0xabc",
	}
	invalid := []string{"", "did:sage", "did:sage:ethereum", "did:web:example.com", "did:sage:polkadot:abc", "sage:ethereum:0xabc", "did:sage::0xabc"}
	popDID := did.AgentDID(vecDIDA)
	fixedTime := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)

	return Suite{
		Name: "did",
		Description: "did:sage method: did:sage:<chain>:<identifier> with chain in {ethereum, solana} (aliases eth, sol, case-insensitive), " +
			"key proof-of-possession over SHA-256(\"SAGE-PoP:\" || DID || \":\" || hex(key)), and the A2A agent-card proof (JCS + Ed25519/secp256k1, base58 proofValue).",
		Cases: []Case{
			{
				Name: "parse", Mode: ModeDeterministic,
				Description: "ParseDID splits chain and identifier; the identifier keeps any further colons. Invalid inputs fail.",
				Input:       map[string]any{"valid": valid, "invalid": invalid},
				Produce: func(in map[string]any) (map[string]any, error) {
					out := []map[string]any{}
					for _, v := range anyStrings(in["valid"]) {
						chain, id, err := did.ParseDID(did.AgentDID(v))
						if err != nil {
							return nil, fmt.Errorf("%q: %w", v, err)
						}
						out = append(out, map[string]any{"did": v, "chain": string(chain), "identifier": id})
					}
					rejected := []string{}
					for _, v := range anyStrings(in["invalid"]) {
						if _, _, err := did.ParseDID(did.AgentDID(v)); err == nil {
							return nil, fmt.Errorf("%q: expected an error", v)
						}
						rejected = append(rejected, v)
					}
					return map[string]any{"parsed": out, "rejected": rejected}, nil
				},
			},
			{
				Name: "chain-aliases", Mode: ModeDeterministic,
				Description: "ParseChain normalises case and whitespace and accepts the eth/sol aliases.",
				Input:       map[string]any{"names": []string{"ethereum", "ETH", " Solana ", "sol"}},
				Produce: func(in map[string]any) (map[string]any, error) {
					out := map[string]any{}
					for _, n := range anyStrings(in["names"]) {
						c, err := did.ParseChain(n)
						if err != nil {
							return nil, err
						}
						out[n] = string(c)
					}
					return map[string]any{"chains": out, "generated": string(did.GenerateDID(did.ChainEthereum, "0xabc"))}, nil
				},
			},
			{
				Name: "pop-ed25519", Mode: ModeDeterministic,
				Description: "Key proof-of-possession: Ed25519 signature over SHA-256 of the challenge string.",
				Input:       map[string]any{"did": vecDIDA, "seed_label": labelEd25519A},
				Produce: func(in map[string]any) (map[string]any, error) {
					priv := ed25519FromLabel(in["seed_label"].(string))
					pub := priv.Public().(ed25519.PublicKey)
					sig, err := did.GenerateKeyProofOfPossession(popDID, pub, priv, did.KeyTypeEd25519)
					if err != nil {
						return nil, err
					}
					return map[string]any{
						"challenge": fmt.Sprintf("SAGE-PoP:%s:%x", vecDIDA, []byte(pub)),
						"key_data":  hx(pub),
						"proof":     hx(sig),
					}, nil
				},
				Verify: func(in, out map[string]any) error {
					key, err := unhex(out, "key_data")
					if err != nil {
						return err
					}
					sig, err := unhex(out, "proof")
					if err != nil {
						return err
					}
					return did.VerifyKeyProofOfPossession(popDID, &did.AgentKey{Type: did.KeyTypeEd25519, KeyData: key, Signature: sig})
				},
			},
			{
				Name: "pop-secp256k1", Mode: ModeDeterministic,
				Description: "Key proof-of-possession: secp256k1 RFC 6979 signature (65 bytes r||s||v) over SHA-256 of the challenge string. Note: SHA-256, not Keccak, for this path.",
				Input:       map[string]any{"did": vecDIDA, "scalar_label": labelSecp256k1A},
				Produce: func(in map[string]any) (map[string]any, error) {
					priv, err := secp256k1FromLabel(in["scalar_label"].(string))
					if err != nil {
						return nil, err
					}
					pub := secp256k1PublicBytes(&priv.PublicKey)
					sig, err := did.GenerateKeyProofOfPossession(popDID, pub, priv, did.KeyTypeECDSA)
					if err != nil {
						return nil, err
					}
					return map[string]any{
						"challenge": fmt.Sprintf("SAGE-PoP:%s:%x", vecDIDA, pub),
						"key_data":  hx(pub),
						"proof":     hx(sig),
					}, nil
				},
				Verify: func(in, out map[string]any) error {
					key, err := unhex(out, "key_data")
					if err != nil {
						return err
					}
					sig, err := unhex(out, "proof")
					if err != nil {
						return err
					}
					return did.VerifyKeyProofOfPossession(popDID, &did.AgentKey{Type: did.KeyTypeECDSA, KeyData: key, Signature: sig})
				},
			},
			{
				Name: "a2a-card-proof-ed25519", Mode: ModeVerify,
				Description: "A2A agent card generated from AgentMetadataV4 and signed with the Ed25519 key: proof.created is the signing time, " +
					"so the vector is verify-only; the check parses the stored card and verifies the proof over the JCS form without \"proof\".",
				Input: map[string]any{"did": vecDIDA, "seed_label": labelEd25519A, "kem_label": labelX25519A, "name": "vector-agent", "endpoint": "https://agent-a.example/a2a", "created": fixedTime.Format(time.RFC3339)},
				Produce: func(in map[string]any) (map[string]any, error) {
					priv := ed25519FromLabel(in["seed_label"].(string))
					pub := priv.Public().(ed25519.PublicKey)
					kem, err := x25519FromLabel(in["kem_label"].(string))
					if err != nil {
						return nil, err
					}
					meta := &did.AgentMetadataV4{
						DID:          popDID,
						Name:         in["name"].(string),
						Description:  "SAGE test vector agent",
						Endpoint:     in["endpoint"].(string),
						Keys:         []did.AgentKey{{Type: did.KeyTypeEd25519, KeyData: pub, Verified: true, CreatedAt: fixedTime}, {Type: did.KeyTypeX25519, KeyData: kem.PublicKey().Bytes(), Verified: true, CreatedAt: fixedTime}},
						Capabilities: map[string]interface{}{"tools": true},
						Owner:        "0x1111111111111111111111111111111111111111",
						IsActive:     true,
						CreatedAt:    fixedTime,
						UpdatedAt:    fixedTime,
						PublicKEMKey: kem.PublicKey().Bytes(),
					}
					card, err := did.GenerateA2ACardWithProof(meta, priv, did.KeyTypeEd25519)
					if err != nil {
						return nil, err
					}
					raw, err := json.Marshal(card)
					if err != nil {
						return nil, err
					}
					return map[string]any{"card_json": string(raw)}, nil
				},
				Verify: func(in, out map[string]any) error {
					raw, err := str(out, "card_json")
					if err != nil {
						return err
					}
					card, err := did.ParseA2AAgentCardWithProof([]byte(raw))
					if err != nil {
						return err
					}
					ok, err := did.VerifyA2ACardProof(card)
					if err != nil {
						return err
					}
					if !ok {
						return errors.New("proof did not verify")
					}
					return nil
				},
			},
		},
	}
}

func anyStrings(v any) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, x := range t {
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	return nil
}
