# DID Layer Architecture Analysis (read-only)

Scope: `pkg/agent/did`, `pkg/agent/did/ethereum`, `pkg/agent/did/solana`, `pkg/blockchain/...`, `deployments/config`, `cmd/sage-did`.

## 0. Two findings that dominate everything else

1. **`did.Manager` never receives a client in any non-test code path.** `Manager.Configure` (pkg/agent/did/manager.go:74-116) only installs a client when `GetEthereumV4ClientCreator()` is non-nil (manager.go:95); `RegisterEthereumV4ClientCreator` (manager.go:40) has zero callers repo-wide, and `Manager.SetClient` (manager.go:145) has zero callers repo-wide (including tests). The `ChainSolana` branch is an empty case (manager.go:107). Every `cmd/sage-did` command built on `did.NewManager()`+`Configure` (resolve, list, verify, update, deactivate, card, key), `pkg/agent/core.ConfigureDID` (pkg/agent/core/core.go:54), and all `examples/*` fail at first use. Verified: `go run ./cmd/sage-did resolve did:sage:ethereum:0x…0001` prints `failed to resolve DID: no resolver for chain ethereum`.
2. **The legacy `EthereumClient` cannot write to the contract it is bound to.** It loads `AgentCardRegistryABI` (pkg/agent/did/ethereum/client.go:97) but packs V2 signatures: `registerAgent` with 7 args (client.go:174-182) vs ABI `registerAgent(string agentId, string endpoint)`; `updateAgent` with 6 args (client.go:464-471) vs `(bytes32, string, string)`; `deactivateAgent(bytes32)` (client.go:494) vs `(string agentId)`. Reads (`getAgentByDID`, `getKey`, `getAgentsByOwner`) match. Its own header says it is DEPRECATED and points to `NewEthereumClientV4()` / `clientv4.go` (client.go:44-53, :73), which do not exist.

## 1. Package roles and leakage

| Package | Intended role | Leaks / violations |
|---|---|---|
| `pkg/agent/did` | Chain-agnostic DID types, interfaces, multi-chain dispatch | A2A card generation/validation/merge (a2a.go:35-342), A2A proof signing/verification with raw ed25519/ecdsa (a2a_proof.go:63-287), key proof-of-possession (key_proof.go:46-271), Ethereum address derivation (utils.go:224), secp256k1/X25519 key (un)marshalling (utils.go:36-110), HTTP endpoint health probing (verification.go:243-305), chain-name switch statements (manager.go:280-295, resolver.go:181-200, factory.go:56-105), global mutable registration hooks (factory.go:139-155, manager.go:31-50), a package-level default `Manager` (did.go:30-65). Chain prefix strings hard-coded 5 times. |
| `pkg/agent/did/ethereum` | Ethereum implementation of registry/resolver | Two unrelated clients (`EthereumClient` raw-ABI, `AgentCardClient` abigen), a third mock `Resolver`/`DIDCache`/`DIDDocument`/`ParsedDID` returning `"mock-public-key"` (resolver.go:252-444, :343), embedded V2 ABI that nothing uses (abi.go:27-41, `SageRegistryABI` referenced only inside abi.go), dead `toKeyHashes` (client.go:575) and `computeAgentID` (agentcard_client.go:590). |
| `pkg/agent/did/solana` | Solana implementation | Also imports `pkg/agent/crypto/chain` (client.go:32) for address handling; JSON-serialised "instructions" (client.go:556-565) rather than Anchor/Borsh. Implements `did.Client`/`Registry` but **not** `did.Resolver` (see section 3). |
| `pkg/blockchain/ethereum/contracts/agentcardregistry` | abigen output (AgentCardRegistry.go, AgentCardStorage.go, IRegistryHook.go; 5388 LOC, "Code generated - DO NOT EDIT") | Consumed only by `pkg/agent/did/ethereum/agentcard_client.go:40`. Role respected. |
| `deployments/config` | YAML/env config loading | Owns its own network presets and a hard-coded Kaia contract address (blockchain.go:43-72, :177; deployment_loader.go:120); defines `DIDConfig.Method/Network/CacheSize/CacheTTL` (config.go:46-52) that `pkg/agent/did` never reads. Not imported by `pkg/agent/did` or `cmd/sage-did`; importers are `cmd/deployment-verify`, `cmd/sage-verify`, `pkg/agent/crypto/chain/ethereum/enhanced_provider.go`. |
| `cmd/sage-did` | CLI consumer | Re-implements chain parsing (helpers.go:44-52), key-type detection for PEM/JWK/raw (helpers.go:98-316), and its own RPC/contract defaults (register.go:171-197). |

## 2. Exported API surface and the "client" zoo

**pkg/agent/did** — types: `AgentDID`, `AgentMetadata` (types.go:31), `AgentMetadataV4`/`AgentKey`/`KeyType` (types_v4.go:34-80), `RegistrationRequest/Result`, `VerificationResult`, `Chain`, `Network`, `DIDError` + 6 sentinels (types.go:97-115), `RegistryConfig` (registry.go:58), `SearchCriteria`, `RegistrationParams`/`CommitmentState`/`CommitmentStatus`/`RegistrationPhase` (types_v4.go:209-248), A2A types. Interfaces: `Client`, `Registry`, `RegistryV4`, `Resolver`, `ClientFactory`. Structs: `MultiChainRegistry`, `MultiChainResolver`, `Manager`, `MetadataVerifier`. Package-level: `Configure/RegisterAgent/ResolveAgent/ValidateAgent/CheckCapabilities/ValidateDID` on a global manager (did.go:42-76), `CreateClient/GetRecommendedKeyType/ValidateKeyTypeForChain/GetRFC9421Algorithm` on a global factory (factory.go:114-133), `GenerateDID/ParseDID`, utils, A2A, PoP funcs.

**pkg/agent/did/ethereum** — `EthereumClient` + `NewEthereumClient` (client.go:54,75), `AgentCardClient` + `NewAgentCardClient` (agentcard_client.go:67,77), `Resolver`/`NewResolver`/`NewResolverWithCache`/`DIDCache`/`DIDDocument`/`ParsedDID` (resolver.go:234-444), `SageRegistryABI`/`AgentCardRegistryABI` strings and getters (abi.go).

**pkg/agent/did/solana** — `SolanaClient` + `NewSolanaClient` (client.go:37,67), `AgentAccount` (client.go:46).

Overlap matrix:

| Abstraction | What it is | Real consumers (non-test) | Status |
|---|---|---|---|
| `did.Client` (client.go:28) | Register/Resolve/Update/Deactivate | Only the factory's return type | Legacy; nothing calls `CreateClient` outside tests |
| `did.Registry` (registry.go:29) | `Client` minus Resolve, plus GetRegistrationStatus | `MultiChainRegistry`, `Manager.setClientUnlocked` type-assert | Never populated in prod |
| `did.RegistryV4` (registry.go:44) | AddKey/RevokeKey/ApproveEd25519Key | `Manager.AddKey/RevokeKey/ApproveEd25519Key` type-assert (manager.go:300-360) | **No implementer anywhere** |
| `did.Resolver` (resolver.go:27) | 6 methods incl. Search/ListAgentsByOwner | `handshake/server.go:62`, `hpke/server.go:47`, `hpke/client.go:46`, `MetadataVerifier`, `ValidateA2ACardWithDID` | The one interface that matters |
| `MultiChainResolver` / `MultiChainRegistry` | map[Chain]->impl dispatch | Only via `Manager` | Redundant with Manager |
| `Manager` | Facade over the two multis + verifier + RWMutex | `cmd/sage-did` (12 call sites), `pkg/agent/core`, examples, `lib/` | Public entry point, but unwired |
| `ClientFactory` (factory.go:29) | Chain switch + key-type mapper pass-through | tests only | Dead in prod |
| `EthereumClient` | Raw-ABI V2-shaped client | `tests/integration/e2e_sepolia_test.go:102` only | Reads work, writes broken |
| `AgentCardClient` | abigen commit-reveal client | `cmd/sage-did/commit.go:175` (+activate/register flow) | The only working write path; not an implementer of any did interface |
| `ethereum.Resolver` | Mock document resolver with cache | `cmd/sage-did/debug.go:77` | Mock; misleading |
| `SolanaClient` | JSON-instruction client | none | Not `did.Resolver`-compatible |

`Manager` itself does not satisfy `did.Resolver` (has `ResolveAgent`, `SearchAgents`; lacks `Resolve`, `ResolveKEMKey`, `VerifyMetadata`), so it cannot be handed to handshake/hpke. It does satisfy `core.DIDResolver`.

## 3. Interfaces: definers, implementers, consumers

- `Client`: implemented by `EthereumClient`, `SolanaClient` (via init hooks client.go:65, solana/client.go:60). Consumed only by factory. Single-purpose duplicate of `Registry`+`Resolver.Resolve`.
- `Registry`: implemented by both clients; consumed by `MultiChainRegistry`.
- `RegistryV4`: zero implementers; consumed by `Manager` type-asserts -> those three `Manager` methods always error "does not support multi-key management" (manager.go:312-315 etc.). `sage-did key add/revoke` therefore cannot work through `Manager`.
- `Resolver`: implemented by `EthereumClient` and `MultiChainResolver`. `SolanaClient.ResolvePublicKey` returns `crypto.PublicKey` (solana/resolver.go:35) instead of `interface{}` and has no `ResolveKEMKey`, so `SolanaClient` is not a `Resolver`; `Manager.SetClient` would reject it (manager.go:152-155).
- `ClientFactory`: single impl `defaultClientFactory`; never consumed in prod.
- `EthereumV4ClientCreator` (func type, manager.go:31): never registered.
- `chain.ChainKeyTypeMapper` (pkg/agent/crypto/chain/key_mapper.go:31-40): wrapped 1:1 by `ClientFactory` with a `did.Chain`->`chain.ChainType` switch repeated twice (factory.go:78-87, 94-103).

## 4. Dependency direction and chain wiring

Wiring uses three different mechanisms simultaneously: (a) `init()` side-effect registration of creators into package globals (ethereum/client.go:65, solana/client.go:60 -> factory.go:139-155); (b) a second global hook `RegisterEthereumV4ClientCreator` that nobody calls; (c) `Manager.SetClient(interface{})` with runtime type assertions. Consumers must blank-import the chain package for (a) to fire; `cmd/sage-did` imports `did/ethereum` only in commit.go and debug.go, never `did/solana`.

Chain-name / DID-method / address / RPC definitions (every location):

| Concern | Locations |
|---|---|
| Chain enum | `did.Chain` types.go:76-79; `chain.ChainType` crypto/chain/types.go:33-36 (adds bitcoin, cosmos); mapping switch factory.go:78-87, 94-103 |
| Chain-string parsing | `ParseDID` manager.go:280-295 (`ethereum|eth|solana|sol`); `extractChainFromDID` resolver.go:181-200 (first 3 chars `eth`/`sol`); `hasChainPrefix/addChainPrefix` registry.go:176-189; `ValidateDID` did.go:67; `ethereum.Resolver.ParseDID` resolver.go:290-321 (4-part only); `cmd parseChain` helpers.go:44-52; `commit.go:101` string compare |
| `did:sage:` prefix | registry.go:177,186,188; manager.go:276; utils.go:154,191; cmd/commit.go:121; config default `Method="sage"` config.go:170, validator.go:165 |
| RPC defaults | cmd/register.go:171-178 (Alchemy placeholder, Solana mainnet); `--rpc` flag default `http://localhost:8545` register.go:71, commit.go:85, activate.go:67; deployments/config/blockchain.go:45,54,63 (local, Kaia kairos, Kaia mainnet — no Ethereum/Sepolia preset); env `SAGE_RPC_URL` |
| Contract addresses | cmd/register.go:192 zero address, :195 Solana `1111…`; deployments/config/deployment_loader.go:120 and blockchain.go:177 `0x4Ba6Fc…6545` (Kaia, keyed as `SageRegistryV2` at deployment_loader.go:41); env `SAGE_REGISTRY_ADDRESS`/`SAGE_CONTRACT_ADDRESS`; deployment JSON search paths deployment_loader.go:69-75. The Sepolia addresses in CLAUDE.md appear nowhere in Go code. |
| Network enum | `did.Network` types.go:82-93 (sepolia/goerli/devnet…) is defined but never read. |

Three unrelated config structs carry the same data with no converter: `did.RegistryConfig` (registry.go:58), `config.BlockchainConfig` (blockchain.go:31), `config.DIDConfig` (config.go:46).

## 5. Duplication / SSOT violations (verified)

| Kind | A | B |
|---|---|---|
| Exact | `EthereumClient.ResolvePublicKey` ethereum/resolver.go:36-47 | `SolanaClient.ResolvePublicKey` solana/resolver.go:35-46; also `MultiChainResolver.ResolvePublicKey` resolver.go:101 and `ResolveKEMKey` :115 / ethereum/resolver.go:50 (same resolve+IsActive shape) |
| Exact | `EthereumClient.Search` stub ethereum/resolver.go:171-179 | `SolanaClient.Search` solana/resolver.go:171-179 |
| Exact | `prepareRegistrationMessage/prepareUpdateMessage` ethereum/client.go:566-573 | solana/client.go:546-553 |
| Near | `VerifyMetadata` ethereum/resolver.go:64-123 | solana/resolver.go:49-111 |
| Near | `AgentCardClient.SetApprovalForAgent/UpdateAgent/DeactivateAgent/UpdateKEMKey` (transactor->tx->wait->status check) agentcard_client.go:422-546 | each other; and `EthereumClient.getTransactOpts/waitForTransaction` client.go:506-564 vs `AgentCardClient.getTransactor` :618 |
| Dead + dup | `toKeyHashes` client.go:575-671 | `chunk32s` closure client.go:228-237; `coerceAgent` duplicates its own keyHashes switch twice (:271-291, :311-330) |
| Type dup (agent metadata) | `did.AgentMetadata` types.go:31 | `did.AgentMetadataV4` types_v4.go:67; `agentMetadataLocal` client.go:205; `agentcardregistry.AgentCardStorageAgentMetadata` AgentCardRegistry.go:42; `solana.AgentAccount` solana/client.go:46; `ethereum.DIDDocument` resolver.go:234; `config.Agent` deployment_loader.go:54 |
| Type dup (key) | `did.AgentKey` types_v4.go:57 | `agentKeyLocal` client.go:219; `AgentCardStorageAgentKey` AgentCardRegistry.go:33 |
| Type dup (params) | `did.RegistrationParams` types_v4.go:209 | `AgentCardStorageRegistrationParams` AgentCardRegistry.go:58 (converted at agentcard_client.go:594) |
| Key-type enums | `did.KeyType` int (types_v4.go:34) | `sagecrypto.KeyType` string; `UnmarshalPublicKey(data, "secp256k1")` string switch utils.go:73; cmd re-detects key types helpers.go:98-316 |
| DID parsing | 6 implementations (section 4) with divergent rules (`ParseDID` accepts >=4 parts; `ethereum.Resolver.ParseDID` requires exactly 4 and an address) | |

## 6. Conventions and inconsistencies

- **Errors**: mostly `fmt.Errorf("…: %w")`. Sentinels (`did.ErrDIDNotFound`) are returned by `EthereumClient.Resolve` (client.go:369) and `SolanaClient.Resolve` (:255) but `AgentCardClient` returns ad-hoc `fmt.Errorf("agent not found")` (agentcard_client.go:322,376). `MultiChainResolver.ListAgentsByOwner/Search` and `ListAgentsByOwner` in both clients silently drop per-chain/per-agent errors (`continue` at resolver.go:151,166; ethereum/resolver.go:162). `MultiChainResolver.Resolve` falls back to probing every chain when the DID has no recognisable prefix (resolver.go:78-90).
- **Context**: `NewEthereumClient` uses `context.Background()` for `NetworkID` (client.go:81); `waitForTransaction` polls with `time.Sleep(5s)` ignoring `ctx` (client.go:552,560) and runs `MaxRetries` iterations — `RegistryConfig.MaxRetries` is never set by `cmd/sage-did`, so the zero value makes it return "transaction timeout" without polling; Solana guards this (`if maxRetries == 0` solana/client.go:518). `AgentCardClient` uses `bind.WaitMined`.
- **Constructors**: `NewX(*did.RegistryConfig) (*X, error)` and dial inside the constructor (network I/O in ctor); abigen contract bound at ctor. `AgentCardClient` re-derives nonce/gas price per tx (agentcard_client.go:618-648).
- **Tx results**: `Register` returns `RegistrationResult{TxHash, Block, Timestamp: time.Now(), GasUsed}` (client.go:193-198) while `GetRegistrationStatus` uses block time (ethereum/resolver.go:208-213); `Update/Deactivate` return only `error` (no hash); `AgentCardClient` returns `CommitmentStatus` with no tx hash and its `UpdateAgent/DeactivateAgent` return only `error`.
- **Caching**: only in the mock `ethereum.Resolver.DIDCache` (resolver.go:381-424). No caching in real resolvers; `DIDConfig.CacheSize/CacheTTL` unused by `did`.
- **Concurrency**: `Manager` holds a `RWMutex`; the underlying `MultiChainRegistry/Resolver` maps are unguarded and exported with `Add*` methods, so direct use is racy.

## 7. Refactoring candidates (moves/merges/extractions)

1. **Wire `Manager.Configure` to a real client (blocker).** Replace the two-hook scheme with one registration (`did.RegisterChain(Chain, func(*RegistryConfig) (Registry, Resolver, error))`) invoked from `ethereum`/`solana` `init()`, and delete `EthereumV4ClientCreator`, `SetClient(interface{})`, `createEthereumClient/createSolanaClient`. Risk: low; nothing calls the removed hooks. Files: manager.go:31-50,74-170; factory.go:139-155; ethereum/client.go:65-69; solana/client.go:60-64.
2. **Collapse `Client` into `Registry`+`Resolver`; drop `RegistryV4`/`ClientFactory`.** Proposed minimal set:
   ```go
   type Registry interface {           // write side
       Register(ctx, *RegistrationRequest) (*RegistrationResult, error)
       Update(ctx, AgentDID, map[string]any, crypto.KeyPair) error
       Deactivate(ctx, AgentDID, crypto.KeyPair) error
       GetRegistrationStatus(ctx, txHash string) (*RegistrationResult, error)
   }
   type Resolver interface {           // read side, what handshake/hpke need
       Resolve(ctx, AgentDID) (*AgentMetadata, error)
       ResolvePublicKey(ctx, AgentDID) (any, error)
       ResolveKEMKey(ctx, AgentDID) (any, error)
   }
   type Lister interface { ListAgentsByOwner(ctx, owner string) ([]*AgentMetadata, error) } // optional
   ```
   Move `VerifyMetadata` into `MetadataVerifier` and delete `Search` (both stubs). Add a shared helper in `did` so `ResolvePublicKey/ResolveKEMKey` are written once. Risk: medium — `handshake`/`hpke` mocks implement the 6-method `Resolver`.
3. **Make `AgentCardClient` the Ethereum `Registry`/`Resolver`; retire `EthereumClient`.** Adapt `AgentCardClient.GetAgentByDID` -> `Resolve`; keep `EthereumClient` only if a real V2 deployment must be supported (bind it to `SageRegistryABI`). Risk: medium — `tests/integration/e2e_sepolia_test.go:102` and `Register` semantics change (commit-reveal is 3 txs).
4. **Delete the mock `ethereum.Resolver`/`DIDCache`/`DIDDocument`/`ParsedDID`** (resolver.go:234-444) and rewrite `cmd/sage-did/debug.go:77` on `did.ParseDID` + the real resolver. If caching is wanted, add a `CachingResolver` decorator in `did` fed by `config.DIDConfig.CacheSize/CacheTTL`.
5. **Single DID parser.** One `did.Parse(AgentDID) (Chain, identifier)` used everywhere; define the `did:sage:` prefix and alias table (`eth`,`sol`) once next to `Chain`.
6. **Move A2A card + proof + PoP into `pkg/agent/did/a2a`** (~900 LOC) and Ethereum-specific helpers into `did/ethereum`.
7. **One config source.** `deployments/config` gains a converter to `did.RegistryConfig`; `cmd/sage-did` drops its default tables; add an Ethereum/Sepolia preset; rename the `SageRegistryV2` JSON key. Caveat: `deployments/config` imports `go-ethereum/ethclient` (validator.go:30), so do the conversion in `cmd` or in `did/ethereum`, not in `did` root.
8. **Remove dead code**: `toKeyHashes`, `computeAgentID`, `SageRegistryV2.abi.json` + `SageRegistryABI` (if item 3), `did.Network` enum, package-level global `Manager`/factory wrappers once callers are confirmed absent.

Uncertain / not verified: whether any external SDK (`sdk/`) or downstream repo relies on `did.Client`, `CreateClient`, or the global `did.Configure` API; whether a V2 `SageRegistry` deployment is still live anywhere.
