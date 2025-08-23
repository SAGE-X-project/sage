package rfc9421

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"
)

// Verifier provides RFC-9421 signature verification
type Verifier struct {
	httpVerifier *HTTPVerifier
}

// NewVerifier creates a new RFC-9421 verifier
func NewVerifier() *Verifier {
	return &Verifier{
		httpVerifier: NewHTTPVerifier(),
	}
}

// VerifySignature verifies a signature according to RFC-9421
func (v *Verifier) VerifySignature(publicKey interface{}, message *Message, opts *VerificationOptions) error {
	if opts == nil {
		opts = DefaultVerificationOptions()
	}
	
	// Verify timestamp is within acceptable range
	if opts.MaxClockSkew > 0 {
		now := time.Now()
		diff := now.Sub(message.Timestamp)
		if diff < -opts.MaxClockSkew || diff > opts.MaxClockSkew {
			return fmt.Errorf("message timestamp outside acceptable range: %v", diff)
		}
	}
	
	// Construct the message to verify based on RFC-9421 partial signing
	signatureBase := v.ConstructSignatureBase(message)
	
	// Verify the signature
	if err := v.verifySignatureWithAlgorithm(publicKey, []byte(signatureBase), message.Signature, message.Algorithm); err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}
	
	return nil
}

// VerifyWithMetadata verifies a signature and checks metadata constraints
func (v *Verifier) VerifyWithMetadata(
	publicKey interface{},
	message *Message,
	expectedMetadata map[string]interface{},
	requiredCapabilities []string,
	opts *VerificationOptions,
) (*VerificationResult, error) {
	result := &VerificationResult{
		VerifiedAt: time.Now(),
	}
	
	// Basic signature verification
	if err := v.VerifySignature(publicKey, message, opts); err != nil {
		result.Valid = false
		result.Error = err.Error()
		return result, nil
	}
	
	// Verify metadata if requested
	if opts != nil && opts.VerifyMetadata && expectedMetadata != nil {
		if err := v.verifyMetadataMatch(message.Metadata, expectedMetadata); err != nil {
			result.Valid = false
			result.Error = fmt.Sprintf("metadata verification failed: %v", err)
			return result, nil
		}
	}
	
	// Check required capabilities
	if len(requiredCapabilities) > 0 && message.Metadata != nil {
		if capabilities, ok := message.Metadata["capabilities"].(map[string]interface{}); ok {
			if !hasRequiredCapabilities(capabilities, requiredCapabilities) {
				result.Valid = false
				result.Error = "agent missing required capabilities"
				return result, nil
			}
		} else {
			result.Valid = false
			result.Error = "agent capabilities not found in metadata"
			return result, nil
		}
	}
	
	result.Valid = true
	return result, nil
}

// ConstructSignatureBase builds the signature base string according to RFC-9421
func (v *Verifier) ConstructSignatureBase(msg *Message) string {
	// RFC-9421 allows partial signing of message components
	var parts []string
	
	for _, field := range msg.SignedFields {
		switch field {
		case "agent_did":
			parts = append(parts, fmt.Sprintf("agent_did: %s", msg.AgentDID))
		case "message_id":
			parts = append(parts, fmt.Sprintf("message_id: %s", msg.MessageID))
		case "timestamp":
			parts = append(parts, fmt.Sprintf("timestamp: %s", msg.Timestamp.Format(time.RFC3339)))
		case "nonce":
			parts = append(parts, fmt.Sprintf("nonce: %s", msg.Nonce))
		case "body":
			parts = append(parts, fmt.Sprintf("body: %s", string(msg.Body)))
		default:
			// Check if it's a header field
			if strings.HasPrefix(field, "header.") {
				headerName := strings.TrimPrefix(field, "header.")
				if value, ok := msg.Headers[headerName]; ok {
					parts = append(parts, fmt.Sprintf("%s: %s", headerName, value))
				}
			}
		}
	}
	
	return strings.Join(parts, "\n")
}

// verifySignatureWithAlgorithm verifies the signature using the appropriate algorithm
func (v *Verifier) verifySignatureWithAlgorithm(publicKey interface{}, message, signature []byte, algorithm string) error {
	switch algorithm {
	case string(AlgorithmEdDSA):
		pk, ok := publicKey.(ed25519.PublicKey)
		if !ok {
			return fmt.Errorf("invalid public key type for EdDSA")
		}
		if !ed25519.Verify(pk, message, signature) {
			return fmt.Errorf("EdDSA signature verification failed")
		}
		
	case string(AlgorithmES256K), string(AlgorithmECDSA), string(AlgorithmECDSASecp256k1):
		ecdsaKey, ok := publicKey.(*ecdsa.PublicKey)
		if !ok {
			return fmt.Errorf("invalid public key type for ECDSA")
		}
		
		// ECDSA signatures in Ethereum are typically 65 bytes (r + s + v)
		// But standard ECDSA is just r + s (64 bytes)
		if len(signature) < 64 {
			return fmt.Errorf("invalid ECDSA signature length: %d", len(signature))
		}
		
		// Extract r and s from the signature
		r := new(big.Int).SetBytes(signature[:32])
		s := new(big.Int).SetBytes(signature[32:64])
		
		// Create a hash of the message
		hash := sha256.Sum256(message)
		
		// Verify the signature
		if !ecdsa.Verify(ecdsaKey, hash[:], r, s) {
			return fmt.Errorf("ECDSA signature verification failed")
		}
		
	default:
		return fmt.Errorf("unsupported signature algorithm: %s", algorithm)
	}
	
	return nil
}

// verifyMetadataMatch checks if message metadata matches expected values
func (v *Verifier) verifyMetadataMatch(actual, expected map[string]interface{}) error {
	for key, expectedValue := range expected {
		actualValue, exists := actual[key]
		if !exists {
			return fmt.Errorf("missing expected metadata field: %s", key)
		}
		
		if !compareValues(expectedValue, actualValue) {
			return fmt.Errorf("metadata field %s mismatch", key)
		}
	}
	
	return nil
}

// VerifyHTTPRequest verifies an HTTP request signature according to RFC-9421
func (v *Verifier) VerifyHTTPRequest(req *http.Request, publicKey interface{}, opts *HTTPVerificationOptions) error {
	return v.httpVerifier.VerifyRequest(req, publicKey, opts)
}

// SignHTTPRequest signs an HTTP request according to RFC-9421
func (v *Verifier) SignHTTPRequest(req *http.Request, sigName string, params *SignatureInputParams, privateKey interface{}) error {
	signer, ok := privateKey.(crypto.Signer)
	if !ok {
		return fmt.Errorf("private key must implement crypto.Signer interface")
	}
	return v.httpVerifier.SignRequest(req, sigName, params, signer)
}

// Helper functions

func hasRequiredCapabilities(agentCaps map[string]interface{}, required []string) bool {
	for _, req := range required {
		if _, exists := agentCaps[req]; !exists {
			return false
		}
	}
	return true
}

func compareValues(v1, v2 interface{}) bool {
	// Simple comparison - can be enhanced for deep object comparison
	j1, _ := json.Marshal(v1)
	j2, _ := json.Marshal(v2)
	return string(j1) == string(j2)
}