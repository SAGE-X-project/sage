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
	"errors"
	"fmt"
)

// GenerateTaskID returns a task ID prefixed with the handshake step, e.g. "invitation-<uuid>".
func GenerateTaskID(p Phase) string {
	// Stable, parseable task id; adjust to match your AIP rules if needed.
	return fmt.Sprintf("handshake/%d", int(p))
}

// These helper functions are no longer needed with transport abstraction
// structpb-based helpers have been removed

func parsePhase(taskID string) (Phase, error) {
	var p int
	_, err := fmt.Sscanf(taskID, "handshake/%d", &p)
	if err != nil || p < int(Invitation) || p > int(Complete) {
		return 0, errors.New("invalid task id")
	}
	return Phase(p), nil
}
