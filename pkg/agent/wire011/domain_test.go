package wire011

import (
	"encoding/hex"
	"testing"
)

// The 0.11.0 selection manifest (sage-spec a22a275,
// profiles/sage-0.11.0/selection.json domain_entries) fixes these bytes.
func TestDomainBytesMatchSelection(t *testing.T) {
	for name, tc := range map[string]struct{ got, wantHex string }{
		"wire_request":  {requestDomain, "736167652d776972652d726571756573747c302e31312e300a"},
		"wire_response": {responseDomain, "736167652d776972652d726573706f6e73657c302e31312e300a"},
		"http tag":      {Tag, "736167652d302e31312e30"},
	} {
		if hex.EncodeToString([]byte(tc.got)) != tc.wantHex {
			t.Fatalf("%s is %q", name, tc.got)
		}
	}
}
