// Copyright (C) 2025 sage-x-project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

// SPDX-License-Identifier: LGPL-3.0-or-later


package ethereum

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed SageRegistryV2.abi.json
var sageRegistryV2ABI []byte

// SageRegistryABI returns the ABI of the SageRegistry contract
func GetSageRegistryABI() (string, error) {
	// Validate the ABI is valid JSON
	var abiData []interface{}
	if err := json.Unmarshal(sageRegistryV2ABI, &abiData); err != nil {
		return "", fmt.Errorf("invalid ABI JSON: %w", err)
	}
	return string(sageRegistryV2ABI), nil
}

// SageRegistryABI is the ABI string (for backward compatibility)
var SageRegistryABI string

func init() {
	var err error
	SageRegistryABI, err = GetSageRegistryABI()
	if err != nil {
		panic(fmt.Sprintf("Failed to load SageRegistry ABI: %v", err))
	}
}