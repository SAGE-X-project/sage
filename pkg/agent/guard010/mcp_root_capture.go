package guard010

// mcpRootCapture binds the exact original input list to a root request. The
// trusted host must capture those bytes and assign a fresh request ID before
// plugin or model expansion, then retain the bytes in protected storage.
type mcpRootCapture struct {
	requestID string
	digest    string
}

func newMCPRootCapture(items [][]byte, requestID string) (*mcpRootCapture, error) {
	if !uuid.MatchString(requestID) {
		return nil, ErrInvalid
	}
	digest, err := OriginalCommitment(items)
	if err != nil {
		return nil, ErrInvalid
	}
	return &mcpRootCapture{requestID: requestID, digest: digest}, nil
}

func (c *mcpRootCapture) matches(raw []byte) bool {
	if c == nil {
		return false
	}
	_, intent, _, err := intentEnvelope(raw)
	return err == nil && intent["parent_call_id"] == nil &&
		str(intent, "request_id") == c.requestID && str(intent, "original_digest") == c.digest
}
