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