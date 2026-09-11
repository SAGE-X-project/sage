package chain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPresetFor(t *testing.T) {
	sepolia, ok := PresetFor("Sepolia")
	require.True(t, ok)
	assert.Equal(t, int64(11155111), sepolia.ChainID.Int64())
	assert.Equal(t, ChainTypeEthereum, sepolia.Chain)
	assert.NotEmpty(t, sepolia.RPCURL)
	assert.Equal(t, sepolia.AgentCardRegistry, sepolia.RegistryAddress())

	// aliases
	for alias, name := range map[string]string{"localhost": "local", "hardhat": "local", "mainnet": "cypress", "kaia": "cypress", "ethereum": "ethereum-mainnet", "devnet": "solana-devnet"} {
		p, ok := PresetFor(alias)
		require.True(t, ok, alias)
		assert.Equal(t, name, p.Name, alias)
	}
	local, _ := PresetFor("local")
	assert.True(t, local.Local)
	assert.Empty(t, local.RegistryAddress())

	kairos, _ := PresetFor("kairos")
	assert.Equal(t, kairos.SageRegistryV2, kairos.RegistryAddress(), "legacy registry used where no AgentCardRegistry exists")

	_, ok = PresetFor("no-such-network")
	assert.False(t, ok)
}

func TestPresets_ChainIDsAreUnique(t *testing.T) {
	seen := map[string]string{}
	for _, p := range Presets() {
		if p.ChainID == nil {
			continue
		}
		key := string(p.Chain) + ":" + p.ChainID.String()
		if prev, dup := seen[key]; dup {
			t.Fatalf("chain id %s shared by %s and %s", key, prev, p.Name)
		}
		seen[key] = p.Name
	}
}
