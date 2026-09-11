package did

import (
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/crypto"
	"github.com/sage-x-project/sage/pkg/agent/crypto/jcs"
	"github.com/sage-x-project/sage/pkg/agent/crypto/keys"
)

const testCardDID = "did:sage:ethereum:0x1234567890abcdef1234567890abcdef12345678"

// signedEd25519Card returns a card for testCardDID signed with a fresh Ed25519
// key, together with the public key bytes as they would be stored on-chain.
func signedEd25519Card(t *testing.T) (*A2AAgentCardWithProof, ed25519.PublicKey) {
	t.Helper()
	kp, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)
	pub := kp.PublicKey().(ed25519.PublicKey)
	now := time.Now()
	metadata := &AgentMetadataV4{
		DID: testCardDID, Name: "Agent", Description: "d", Endpoint: "https://agent.example",
		Keys:  []AgentKey{{Type: KeyTypeEd25519, KeyData: pub, Verified: true, CreatedAt: now}},
		Owner: "0x1234567890abcdef", IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	card, err := GenerateA2ACardWithProof(metadata, kp.PrivateKey().(ed25519.PrivateKey), KeyTypeEd25519)
	require.NoError(t, err)
	return card, pub
}

// onChain returns the metadata a resolver would produce for an agent whose
// verified signing key is pub (a parsed key, as pkg/agent/did/ethereum returns it).
func onChain(pub interface{}, active bool) *AgentMetadata {
	return &AgentMetadata{
		DID: testCardDID, Name: "Agent", Endpoint: "https://agent.example",
		PublicKey: pub, IsActive: active, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
}

func TestVerifyA2ACardProofWithDID_AcceptsKeyVerifiedOnChain(t *testing.T) {
	card, pub := signedEd25519Card(t)
	resolver := new(MockResolver)
	resolver.On("Resolve", mock.Anything, AgentDID(testCardDID)).Return(onChain(pub, true), nil)

	require.NoError(t, VerifyA2ACardProofWithDID(context.Background(), card, resolver))
	require.NoError(t, ValidateA2ACardWithProofAndDID(context.Background(), card, resolver))
}

// A forger lists their own key in the card and signs with it. The
// self-attested check passes; the chain-bound check must not.
func TestVerifyA2ACardProofWithDID_RejectsSelfSignedForgery(t *testing.T) {
	forged, _ := signedEd25519Card(t)
	victimKP, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)
	resolver := new(MockResolver)
	resolver.On("Resolve", mock.Anything, AgentDID(testCardDID)).Return(onChain(victimKP.PublicKey(), true), nil)

	valid, err := VerifyA2ACardProof(forged)
	require.NoError(t, err)
	assert.True(t, valid, "self-attested check cannot detect the forgery")

	err = VerifyA2ACardProofWithDID(context.Background(), forged, resolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a verified key")
	require.Error(t, ValidateA2ACardWithProofAndDID(context.Background(), forged, resolver))
}

func TestVerifyA2ACardProofWithDID_RejectsForeignVerificationMethod(t *testing.T) {
	card, pub := signedEd25519Card(t)
	card.Proof.VerificationMethod = "did:sage:ethereum:0xother#key-1"
	card.PublicKeys[0].ID = card.Proof.VerificationMethod
	resolver := new(MockResolver)
	resolver.On("Resolve", mock.Anything, mock.Anything).Return(onChain(pub, true), nil)

	err := VerifyA2ACardProofWithDID(context.Background(), card, resolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not belong to DID")
}

func TestVerifyA2ACardProofWithDID_RejectsInactiveAgent(t *testing.T) {
	card, pub := signedEd25519Card(t)
	resolver := new(MockResolver)
	resolver.On("Resolve", mock.Anything, AgentDID(testCardDID)).Return(onChain(pub, false), nil)

	err := VerifyA2ACardProofWithDID(context.Background(), card, resolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not active")
}

func TestVerifyA2ACardProofWithDID_ResolveFailure(t *testing.T) {
	card, _ := signedEd25519Card(t)
	resolver := new(MockResolver)
	resolver.On("Resolve", mock.Anything, AgentDID(testCardDID)).Return(nil, errors.New("rpc down"))

	err := VerifyA2ACardProofWithDID(context.Background(), card, resolver)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rpc down")
	require.Error(t, VerifyA2ACardProofWithDID(context.Background(), card, nil))
	require.Error(t, VerifyA2ACardProofWithDID(context.Background(), &A2AAgentCardWithProof{}, resolver))
}

// The resolver returns the secp256k1 key as *ecdsa.PublicKey; the card carries
// the 64-byte x||y encoding. Both must compare equal.
func TestVerifyA2ACardProofWithDID_ECDSAParsedKey(t *testing.T) {
	kp, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	keyData, err := MarshalPublicKey(kp.PublicKey())
	require.NoError(t, err)
	require.Len(t, keyData, 64)
	now := time.Now()
	metadata := &AgentMetadataV4{
		DID: testCardDID, Name: "Agent", Endpoint: "https://agent.example",
		Keys:     []AgentKey{{Type: KeyTypeECDSA, KeyData: keyData, Verified: true, CreatedAt: now}},
		IsActive: true, CreatedAt: now, UpdatedAt: now,
	}
	card, err := GenerateA2ACardWithProof(metadata, kp.PrivateKey(), KeyTypeECDSA)
	require.NoError(t, err)

	resolver := new(MockResolver)
	resolver.On("Resolve", mock.Anything, AgentDID(testCardDID)).Return(onChain(kp.PublicKey().(*ecdsa.PublicKey), true), nil)
	require.NoError(t, ValidateA2ACardWithProofAndDID(context.Background(), card, resolver))
}

// A card that carries only publicKeyHex must verify (the hex field used to be
// decoded as Base58).
func TestVerifyA2ACardProof_HexOnlyECDSAKey(t *testing.T) {
	kp, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	keyData, err := MarshalPublicKey(kp.PublicKey())
	require.NoError(t, err)

	card := A2AAgentCard{
		ID: testCardDID, Type: []string{"Agent"}, Name: "Agent",
		PublicKeys: []A2APublicKey{{
			ID: testCardDID + "#key-1", Type: "EcdsaSecp256k1VerificationKey2019",
			Controller: testCardDID, PublicKeyHex: "0x" + hex.EncodeToString(keyData),
		}},
		Endpoints: []A2AEndpoint{{Type: "MessageService", URI: "https://agent.example"}},
	}
	canonical, err := jcs.Marshal(card)
	require.NoError(t, err)
	sig, err := keys.SignSecp256k1Keccak(kp.PrivateKey().(*ecdsa.PrivateKey), canonical)
	require.NoError(t, err)
	withProof := &A2AAgentCardWithProof{A2AAgentCard: card, Proof: &A2AProof{
		Type: "EcdsaSecp256k1Signature2019", Created: time.Now(), VerificationMethod: testCardDID + "#key-1",
		ProofPurpose: "assertionMethod", ProofValue: base58.Encode(sig),
	}}

	valid, err := VerifyA2ACardProof(withProof)
	require.NoError(t, err)
	assert.True(t, valid)
}

func TestFromAgentMetadata_ParsedKeys(t *testing.T) {
	edKP, err := crypto.GenerateEd25519KeyPair()
	require.NoError(t, err)
	v4 := FromAgentMetadata(&AgentMetadata{DID: testCardDID, PublicKey: edKP.PublicKey()})
	require.Len(t, v4.Keys, 1)
	assert.Equal(t, KeyTypeEd25519, v4.Keys[0].Type)
	assert.Equal(t, []byte(edKP.PublicKey().(ed25519.PublicKey)), v4.Keys[0].KeyData)

	ecKP, err := crypto.GenerateSecp256k1KeyPair()
	require.NoError(t, err)
	v4 = FromAgentMetadata(&AgentMetadata{DID: testCardDID, PublicKey: ecKP.PublicKey()})
	require.Len(t, v4.Keys, 1)
	assert.Equal(t, KeyTypeECDSA, v4.Keys[0].Type)
	assert.Len(t, v4.Keys[0].KeyData, 64)
}

// A signed card must verify after any JSON-preserving rewrite (whitespace,
// member order, escaping), because the proof covers the RFC 8785 canonical form.
func TestVerifyA2ACardProof_CanonicalFormSurvivesReformatting(t *testing.T) {
	card, _ := signedEd25519Card(t)
	wire, err := json.Marshal(card)
	require.NoError(t, err)

	var generic map[string]interface{}
	require.NoError(t, json.Unmarshal(wire, &generic))
	reordered, err := json.MarshalIndent(generic, "", "  ") // Go sorts map keys: different order and whitespace
	require.NoError(t, err)
	require.NotEqual(t, string(wire), string(reordered))

	parsed, err := ParseA2AAgentCardWithProof(reordered)
	require.NoError(t, err)
	valid, err := VerifyA2ACardProof(parsed)
	require.NoError(t, err)
	assert.True(t, valid)

	// Changing any covered member breaks the proof even when the struct still parses.
	generic["name"] = "Other Agent"
	tampered, err := json.Marshal(generic)
	require.NoError(t, err)
	parsed, err = ParseA2AAgentCardWithProof(tampered)
	require.NoError(t, err)
	_, err = VerifyA2ACardProof(parsed)
	require.Error(t, err)
}
