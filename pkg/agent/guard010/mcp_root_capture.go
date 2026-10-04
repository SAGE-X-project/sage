package guard010

// RootCapture binds the exact original input list to a root request. The
// trusted host must capture those bytes and assign a fresh request ID before
// plugin or model expansion, then retain the bytes in protected storage.
// This value does not attest that the host captured the user's actual input.
type RootCapture struct {
	requestID string
	digest    string
}

// NewRootCapture commits the ordered, exact UTF-8 bytes at the trusted input
// boundary. The host owns durable storage for the input and this request ID.
func NewRootCapture(items [][]byte, requestID string) (*RootCapture, error) {
	if !uuid.MatchString(requestID) {
		return nil, ErrInvalid
	}
	digest, err := OriginalCommitment(items)
	if err != nil {
		return nil, ErrInvalid
	}
	return &RootCapture{requestID: requestID, digest: digest}, nil
}

// Keep the existing private MCP owner path bound to the same capture type.
type mcpRootCapture = RootCapture

func newMCPRootCapture(items [][]byte, requestID string) (*mcpRootCapture, error) {
	return NewRootCapture(items, requestID)
}

func (c *RootCapture) matches(raw []byte) bool {
	if c == nil {
		return false
	}
	_, intent, _, err := intentEnvelope(raw)
	return err == nil && intent["parent_call_id"] == nil &&
		str(intent, "request_id") == c.requestID && str(intent, "original_digest") == c.digest
}
