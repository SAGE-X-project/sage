package ethereum

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Importing this package must make did.Manager.Configure able to build an
// Ethereum client on its own.
func TestInit_RegistersManagerClientCreator(t *testing.T) {
	require.NotNil(t, did.GetEthereumV4ClientCreator(), "ethereum package must register a client creator for did.Manager")
}

// The client handed to did.Manager must satisfy both interfaces it checks.
var (
	_ did.Registry = (*EthereumClient)(nil)
	_ did.Resolver = (*EthereumClient)(nil)
)
