// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.


package handshake

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	sagecrypto "github.com/sage-x-project/sage/crypto"
	"google.golang.org/protobuf/types/known/structpb"
)

// GenerateTaskID returns a task ID prefixed with the handshake step, e.g. "invitation-<uuid>".
func GenerateTaskID(p Phase) string {
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

func signStruct(k sagecrypto.KeyPair, msg []byte, did string) (*structpb.Struct, error) {
	sig, err := k.Sign(msg)
	if err != nil {
		return nil, err
	}
	return structpb.NewStruct(map[string]any{
		"signature": base64.RawURLEncoding.EncodeToString(sig),
		"did": did,
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
