package registry010

// PoPChallenge010 constructs the exact REG-04 bytes for a previously validated
// registry entry. It does not validate DID grammar, key material, proof
// signatures, signer authority, or the registry observation.
func PoPChallenge010(registryID, agentID, name, alg string, keyBytes []byte) ([]byte, error) {
	fields := [][]byte{[]byte(registryID), []byte(agentID), []byte(name), []byte(alg), keyBytes}
	for i, field := range fields {
		if len(field) == 0 || len(field) > 65535 {
			return nil, ErrRejected
		}
		if i < 4 {
			for _, b := range field {
				if b < 0x21 || b > 0x7e {
					return nil, ErrRejected
				}
			}
		}
	}
	challenge := []byte("sage-pop-0.10.0")
	for _, field := range fields {
		challenge = append(challenge, byte(len(field)>>8), byte(len(field)))
		challenge = append(challenge, field...)
	}
	return challenge, nil
}
