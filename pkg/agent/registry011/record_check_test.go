package registry011

import (
	"bytes"
	"context"
	"os"
	"testing"
)

const checkRegistry = "eip155:31337:0x2279b7a0a67db372996a5fab50d91eaa73d2ebe6"

// record-current.json is the record the chain Registry returned on Anvil after
// the lifecycle scenario revoked a leaked signing key: it holds a revoked
// signing key and a KEM key whose endorsing signer is that revoked key.
func currentRecord(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile("testdata/record-current.json")
	if err != nil {
		t.Fatal(err)
	}
	return bytes.TrimSpace(raw)
}

func TestCheckRecordAcceptsTheCurrentRecord(t *testing.T) {
	if err := CheckRecord(checkRegistry, "payment-agent-2", currentRecord(t)); err != nil {
		t.Fatalf("current record refused: %v", err)
	}
}

func TestCheckRecordRefusals(t *testing.T) {
	raw := currentRecord(t)
	for name, mutated := range map[string][]byte{
		"signing proof":   bytes.Replace(raw, []byte(`"value":"u8V1`), []byte(`"value":"v8V1`), 1),
		"KEM endorsement": bytes.Replace(raw, []byte(`"value":"pbZT`), []byte(`"value":"qbZT`), 1),
		"KEM signer gone": bytes.Replace(raw, []byte(`payment-agent-2#sig-1","value":"pbZT`), []byte(`payment-agent-2#sig-9","value":"pbZT`), 1),
		"version zero":    bytes.Replace(raw, []byte(`"version":"6"`), []byte(`"version":"0"`), 1),
		"leading zero":    bytes.Replace(raw, []byte(`"version":"6"`), []byte(`"version":"06"`), 1),
		"state":           bytes.Replace(raw, []byte(`"state":"active"`), []byte(`"state":"paused"`), 1),
		"key state":       bytes.Replace(raw, []byte(`"state":"revoked"`), []byte(`"state":"retired"`), 1),
		"controller":      bytes.Replace(raw, []byte(`"controller":"0x7099`), []byte(`"controller":"0X7099`), 1),
		"service URI":     bytes.Replace(raw, []byte(`"uri":"https://agent.example.invalid/a2a"`), []byte(`"uri":"http://agent.example.invalid/a2a"`), 1),
		"not canonical":   append([]byte(" "), raw...),
	} {
		if bytes.Equal(mutated, raw) {
			t.Fatalf("%s: mutation did not apply", name)
		}
		if CheckRecord(checkRegistry, "payment-agent-2", mutated) == nil {
			t.Fatalf("%s accepted", name)
		}
	}
	if CheckRecord("eip155:31337:0x0000000000000000000000000000000000000001", "payment-agent-2", raw) == nil {
		t.Fatal("record of another Registry accepted")
	}
	if CheckRecord(checkRegistry, "payment-agent-2", nil) == nil || CheckRecord(checkRegistry, "", raw) == nil {
		t.Fatal("empty input accepted")
	}
	var nilReads *PinnedAgentReads
	if nilReads.CheckCode(context.Background()) == nil {
		t.Fatal("unconfigured code check passed")
	}
}
