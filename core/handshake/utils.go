package handshake

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	sagecrypto "github.com/sage-x-project/sage/crypto"
	"google.golang.org/protobuf/types/known/structpb"
)

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
func generateTaskID(p Phase) string {
	// Stable, parseable task id; adjust to match your AIP rules if needed.
	return fmt.Sprintf("handshake/%d", int(p))
}

func toStructPB(v any) (*structpb.Struct, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return structpb.NewStruct(m)
}

func fromStructPB[T any](st *structpb.Struct, out *T) error {
	if st == nil {
		return errors.New("nil payload")
	}
	b, err := st.MarshalJSON()
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func b64ToStructPB(s string) (*structpb.Struct, error) {
	return structpb.NewStruct(map[string]any{"b64": s})
}

func structPBToB64(st *structpb.Struct) (string, error) {
	if st == nil || st.Fields == nil {
		return "", errors.New("nil payload")
	}
	v, ok := st.Fields["b64"]
	if !ok {
		return "", errors.New("missing b64")
	}
	return v.GetStringValue(), nil
}

func signStruct(k sagecrypto.KeyPair, msg []byte) (*structpb.Struct, error) {
	sig, err := k.Sign(msg)
	if err != nil {
		return nil, err
	}
	return structpb.NewStruct(map[string]any{
		"signature": base64.RawURLEncoding.EncodeToString(sig),
	})
}

func parsePhase(taskID string) (Phase, error) {
	var p int
	_, err := fmt.Sscanf(taskID, "handshake/%d", &p)
	if err != nil || p < int(Invitation) || p > int(Complete) {
		return 0, errors.New("invalid task id")
	}
	return Phase(p), nil
}