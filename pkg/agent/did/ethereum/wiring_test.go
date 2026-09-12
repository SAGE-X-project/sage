package ethereum

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sage-x-project/sage/pkg/agent/did"
)

// Importing this package must make did.Manager.Configure able to build an
// Ethereum client on its own.
func TestRegister_InstallsManagerClientCreator(t *testing.T) {
	Register()
	require.NotNil(t, did.ClientCreatorFor(did.ChainEthereum), "Register must install a client creator for did.Manager")
	Register() // idempotent
	require.NotNil(t, did.ClientCreatorFor(did.ChainEthereum))
}

// The client handed to did.Manager must satisfy both interfaces it checks.
var _ did.ChainClient = (*EthereumClient)(nil)
