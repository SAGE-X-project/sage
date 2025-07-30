package handshake

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
)

// toStructPB converts any Go value into a *structpb.Struct via JSON round‑trip.
func toStructPB(v interface{}) (*structpb.Struct, error) {
    b, err := json.Marshal(v)
    if err != nil {
        return nil, err
    }
    var m map[string]interface{}
    if err := json.Unmarshal(b, &m); err != nil {
        return nil, err
    }
    return structpb.NewStruct(m)
}

// BytesToStructPB wraps raw bytes into a Struct with a single "payload" field.
func bytesToStructPB(b []byte) (*structpb.Struct, error) {
    s := base64.StdEncoding.EncodeToString(b)
    m := map[string]interface{}{
        "payload": s,
    }
    st, err := structpb.NewStruct(m)
    if err != nil {
        return nil, err
    }
    return st, nil
}

// b64ToStructPB wraps a Base64‐encoded packet string into a structpb.Struct
// under the `"packet"` field
func b64ToStructPB(packetB64 string) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]interface{}{
		"packet": packetB64,
	})
}

// FromStructPB converts a structpb.Struct into the target Go value (pointer).
func fromStructPB(s *structpb.Struct, out interface{}) error {
    if s == nil {
        return fmt.Errorf("nil Struct")
    }
    m := s.AsMap()

    b, err := json.Marshal(m)
    if err != nil {
        return err
    }
    if err := json.Unmarshal(b, out); err != nil {
        return err
    }
    return nil
}

// fromBytes takes a JSON‐encoded byte slice and unmarshals it into out.
func fromBytes(data []byte, out interface{}) error {
    if len(data) == 0 {
        return fmt.Errorf("empty input byte slice")
    }
    if out == nil {
        return fmt.Errorf("nil output target")
    }
    if err := json.Unmarshal(data, out); err != nil {
        return fmt.Errorf("failed to unmarshal bytes: %w", err)
    }
    return nil
}

// GenerateTaskID returns a task ID prefixed with the handshake step, e.g. "invitation-<uuid>".
func generateTaskID(step string) string {
    return fmt.Sprintf("%s-%s", step, uuid.NewString())
}

