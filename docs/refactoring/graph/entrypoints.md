# Entry points and feature reachability

Reachability follows call and ref edges from each entry point, including dynamic dispatch through module-internal interfaces (interface method -> every implementer). Numbers in parentheses are reachable functions per package.

## Binaries, cgo library and examples

| Entry package | Kind | Reachable module packages |
|---|---|---|
| cmd/deployment-verify | cmd | deployments/config(6) |
| cmd/metrics-demo | cmd | pkg/agent/session(42), pkg/telemetry/metrics(6) |
| cmd/sage-crypto | cmd | internal/cli(1), pkg/agent/crypto/chain(12), pkg/agent/crypto/chain/ethereum(5), pkg/agent/crypto/chain/solana(7), pkg/agent/crypto/formats(18), pkg/agent/crypto/keys(61), pkg/agent/crypto/rotation(4), pkg/agent/crypto/storage(8) |
| cmd/sage-did | cmd | internal/cli(1), pkg/agent/crypto/chain(2), pkg/agent/crypto/chain/ethereum(3), pkg/agent/crypto/chain/solana(4), pkg/agent/crypto/formats(11), pkg/agent/crypto/jcs(6), pkg/agent/crypto/keys(29), pkg/agent/crypto/storage(4), pkg/agent/did(46), pkg/agent/did/ethereum(16), pkg/agent/did/solana(5), pkg/blockchain/ethereum/contracts/agentcardregistry(8) |
| cmd/sage-verify | cmd | deployments/config(6), pkg/health(4) |
| examples/mcp-integration/basic-demo | examples | pkg/agent/core/message/nonce(4), pkg/agent/core/rfc9421(45), pkg/agent/crypto(5), pkg/agent/crypto/keys(21) |
| examples/mcp-integration/basic-tool | examples | pkg/agent/core(2), pkg/agent/core/message/nonce(4), pkg/agent/core/rfc9421(42), pkg/agent/crypto(5), pkg/agent/crypto/keys(3), pkg/agent/did(18), pkg/agent/did/ethereum(2) |
| examples/mcp-integration/client | examples | pkg/agent/core/message/nonce(4), pkg/agent/core/rfc9421(45), pkg/agent/crypto(5), pkg/agent/crypto/keys(21) |
| examples/mcp-integration/simple-standalone | examples | pkg/agent/core/message/nonce(3), pkg/agent/core/rfc9421(20), pkg/agent/crypto/keys(13) |
| examples/mcp-integration/vulnerable-vs-secure/attacker | examples |  |
| examples/mcp-integration/vulnerable-vs-secure/secure-chat | examples |  |
| examples/mcp-integration/vulnerable-vs-secure/vulnerable-chat | examples |  |
| lib | other |  |

## CLI commands (cobra)

| Binary | Command | Handler | Reachable pkg packages |
|---|---|---|---|
| cmd/sage-crypto | `address` | - |  |
| cmd/sage-crypto | `generate` | cmd/sage-crypto.runAddressGenerate | internal/cli(1), pkg/agent/crypto/chain(10), pkg/agent/crypto/chain/ethereum(3), pkg/agent/crypto/chain/solana(4), pkg/agent/crypto/formats(11), pkg/agent/crypto/keys(32), pkg/agent/crypto/storage(4) |
| cmd/sage-crypto | `generate` | cmd/sage-crypto.runGenerate | pkg/agent/crypto/formats(7), pkg/agent/crypto/keys(35), pkg/agent/crypto/storage(4) |
| cmd/sage-crypto | `list` | cmd/sage-crypto.runList | pkg/agent/crypto/formats(10), pkg/agent/crypto/keys(25), pkg/agent/crypto/storage(6) |
| cmd/sage-crypto | `parse [address]` | cmd/sage-crypto.runAddressParse | pkg/agent/crypto/chain(4), pkg/agent/crypto/chain/ethereum(4), pkg/agent/crypto/chain/solana(5) |
| cmd/sage-crypto | `rotate` | cmd/sage-crypto.runRotate | pkg/agent/crypto/formats(12), pkg/agent/crypto/keys(34), pkg/agent/crypto/rotation(4), pkg/agent/crypto/storage(6) |
| cmd/sage-crypto | `sage-crypto` | - |  |
| cmd/sage-crypto | `sign` | cmd/sage-crypto.runSign | internal/cli(1), pkg/agent/crypto/formats(11), pkg/agent/crypto/keys(35), pkg/agent/crypto/storage(4) |
| cmd/sage-crypto | `verify` | cmd/sage-crypto.runVerify | pkg/agent/crypto/formats(12), pkg/agent/crypto/keys(42) |
| cmd/sage-did | `activate <commit-hash>` | cmd/sage-did.runActivate | pkg/agent/did/ethereum(3), pkg/blockchain/ethereum/contracts/agentcardregistry(3) |
| cmd/sage-did | `add <did> <keyfile>` | cmd/sage-did.runKeyAdd | pkg/agent/did(11) |
| cmd/sage-did | `approve <keyhash>` | cmd/sage-did.runKeyApprove | pkg/agent/did(11) |
| cmd/sage-did | `card` | - |  |
| cmd/sage-did | `commit` | cmd/sage-did.runCommit | pkg/agent/did/ethereum(4), pkg/blockchain/ethereum/contracts/agentcardregistry(4) |
| cmd/sage-did | `deactivate [DID]` | cmd/sage-did.runDeactivate | internal/cli(1), pkg/agent/crypto/formats(11), pkg/agent/crypto/keys(28), pkg/agent/crypto/storage(4), pkg/agent/did(13), pkg/agent/did/ethereum(3), pkg/agent/did/solana(3) |
| cmd/sage-did | `debug` | cmd/sage-did.runDebug | pkg/agent/did(16), pkg/agent/did/ethereum(2) |
| cmd/sage-did | `generate [DID]` | cmd/sage-did.runCardGenerate | pkg/agent/crypto/keys(3), pkg/agent/did(22), pkg/agent/did/ethereum(2) |
| cmd/sage-did | `key` | - |  |
| cmd/sage-did | `list` | cmd/sage-did.runList | pkg/agent/did(12), pkg/agent/did/ethereum(3) |
| cmd/sage-did | `list <did>` | cmd/sage-did.runKeyList | pkg/agent/crypto/keys(3), pkg/agent/did(20), pkg/agent/did/ethereum(2) |
| cmd/sage-did | `register [commit-hash]` | cmd/sage-did.runRegister | pkg/agent/did/ethereum(5), pkg/blockchain/ethereum/contracts/agentcardregistry(5) |
| cmd/sage-did | `resolve [DID]` | cmd/sage-did.runResolve | pkg/agent/did(16), pkg/agent/did/ethereum(2) |
| cmd/sage-did | `revoke <did> <keyhash>` | cmd/sage-did.runKeyRevoke | pkg/agent/did(11) |
| cmd/sage-did | `sage-did` | - |  |
| cmd/sage-did | `show [DID]` | cmd/sage-did.runCardShow | pkg/agent/crypto/keys(3), pkg/agent/did(22), pkg/agent/did/ethereum(2) |
| cmd/sage-did | `update [DID]` | cmd/sage-did.runUpdate | internal/cli(1), pkg/agent/crypto/chain(2), pkg/agent/crypto/chain/ethereum(3), pkg/agent/crypto/chain/solana(4), pkg/agent/crypto/formats(11), pkg/agent/crypto/keys(28), pkg/agent/crypto/storage(4), pkg/agent/did(13), pkg/agent/did/ethereum(4), pkg/agent/did/solana(4) |
| cmd/sage-did | `validate [FILE]` | cmd/sage-did.runCardValidate | pkg/agent/crypto/jcs(6), pkg/agent/crypto/keys(4), pkg/agent/did(27), pkg/agent/did/ethereum(2) |
| cmd/sage-did | `verify [DID]` | cmd/sage-did.runVerify | pkg/agent/did(16), pkg/agent/did/ethereum(2) |
| cmd/sage-did | `verify-pop <did>` | cmd/sage-did.runKeyVerifyPop | pkg/agent/crypto/keys(3), pkg/agent/did(22), pkg/agent/did/ethereum(2) |

## pkg packages not reachable from any cmd/lib/example

- pkg/agent/core/message (0 funcs, 42 LOC)
- pkg/agent/core/message/dedupe (7 funcs, 124 LOC)
- pkg/agent/core/message/order (9 funcs, 153 LOC)
- pkg/agent/core/message/validator (5 funcs, 175 LOC)
- pkg/agent/crypto/vault (15 funcs, 407 LOC)
- pkg/agent/handshake (36 funcs, 951 LOC)
- pkg/agent/hpke (49 funcs, 1569 LOC)
- pkg/agent/transport (12 funcs, 355 LOC)
- pkg/agent/transport/http (15 funcs, 520 LOC)
- pkg/agent/transport/websocket (33 funcs, 766 LOC)
- pkg/blockchain/ethereum (13 funcs, 355 LOC)
- pkg/oidc (0 funcs, 40 LOC)
- pkg/oidc/auth0 (13 funcs, 490 LOC)
- pkg/storage (0 funcs, 155 LOC)
- pkg/storage/memory (26 funcs, 477 LOC)
- pkg/storage/postgres (26 funcs, 688 LOC)
- pkg/storage/storagetest (6 funcs, 201 LOC)
- pkg/telemetry (0 funcs, 23 LOC)
- pkg/telemetry/logger (30 funcs, 361 LOC)
- pkg/version (7 funcs, 143 LOC)
