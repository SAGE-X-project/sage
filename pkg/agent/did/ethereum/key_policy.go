package ethereum

import (
	"fmt"
	"math/big"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// onChainKey mirrors the AgentKey struct returned by AgentCardRegistry.getKey.
type onChainKey struct {
	KeyType      uint8    `abi:"keyType"` // 0=ECDSA, 1=Ed25519, 2=X25519
	KeyData      []byte   `abi:"keyData"`
	Signature    []byte   `abi:"signature"`
	Verified     bool     `abi:"verified"`
	RegisteredAt *big.Int `abi:"registeredAt"`
}

// selectAgentKeys applies the key policy to the keys listed on chain for an
// agent and returns the signing public key and the raw X25519 KEM key.
//
// Policy:
//   - Only verified keys are eligible. The registry expresses revocation by
//     clearing the verified flag while keeping the key hash in the agent's
//     list, so this single rule covers both "never proven" and "revoked".
//   - The signing key is the first verified ECDSA (secp256k1) key; if the
//     agent has none, the first verified Ed25519 key.
//   - The KEM key is the first verified X25519 key (the dedicated
//     kemPublicKey field on the agent record takes precedence in Resolve).
//
// A verified key whose bytes cannot be parsed is an error rather than a skip,
// because it indicates corrupt registry data that must not be silently
// downgraded to another key.
func selectAgentKeys(keys []onChainKey) (pub interface{}, kem []byte, err error) {
	var edPub interface{}
	for _, k := range keys {
		if !k.Verified {
			continue
		}
		switch did.KeyType(k.KeyType) {
		case did.KeyTypeECDSA:
			if pub != nil {
				continue
			}
			pk, perr := did.UnmarshalPublicKey(k.KeyData, "secp256k1")
			if perr != nil {
				return nil, nil, fmt.Errorf("verified ECDSA key is malformed: %w", perr)
			}
			pub = pk
		case did.KeyTypeEd25519:
			if edPub != nil {
				continue
			}
			pk, perr := did.UnmarshalPublicKey(k.KeyData, "ed25519")
			if perr != nil {
				return nil, nil, fmt.Errorf("verified Ed25519 key is malformed: %w", perr)
			}
			edPub = pk
		case did.KeyTypeX25519:
			if kem == nil && len(k.KeyData) == 32 {
				kem = append([]byte(nil), k.KeyData...)
			}
		}
	}
	if pub == nil {
		pub = edPub
	}
	return pub, kem, nil
}

// applyKeyPolicy fills the legacy single-key view of agent from its key
// list with the same rules as selectAgentKeys, so that both Ethereum clients
// report the same signing and KEM keys. A KEM key already present on the
// agent record is kept.
func applyKeyPolicy(agent *did.AgentMetadata) error {
	rows := make([]onChainKey, 0, len(agent.Keys))
	for _, k := range agent.Keys {
		rows = append(rows, onChainKey{KeyType: uint8(k.Type), KeyData: k.KeyData, Signature: k.Signature, Verified: k.Verified})
	}
	pub, kem, err := selectAgentKeys(rows)
	if err != nil {
		return fmt.Errorf("agent %s: %w", agent.DID, err)
	}
	agent.PublicKey = pub
	if existing, ok := agent.PublicKEMKey.([]byte); !ok || len(existing) == 0 {
		agent.PublicKEMKey = kem
	}
	return nil
}
