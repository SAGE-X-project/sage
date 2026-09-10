# Architecture Analysis: `pkg/agent/crypto/**` and `internal/cryptoinit`

Read-only review; line numbers refer to the current working tree.

## 1. Package roles and role leakage

| Package | Intended role | Respected? / Leakage |
|---|---|---|
| `pkg/agent/crypto` (root) | Interface + type contract layer (`KeyPair`, `KeyStorage`, `KeyExporter/Importer`, `KeyRotator`, `KeyType`, sentinel errors). | Partially. Leaks: (a) a concrete `Manager` (`manager.go:26`) that needs implementations, which forces (b) the late-bound function-pointer table in `wrappers.go:24-48`; (c) a mutable global algorithm registry with RFC 9421 name mapping (`algorithm_registry.go:58-64`), which is an RFC 9421 concern living below `core/rfc9421`; (d) `GetKeyTypeFromPublicKey` (`algorithm_registry.go:202-214`) maps every `*ecdsa.PublicKey` to `KeyTypeSecp256k1`, so P-256 keys are mis-typed (no test covers P-256 here). |
| `crypto/keys` | Key pair implementations + generation. | Leaks: `x25519.go` bundles ECIES-style AES-GCM encryption (`x25519.go:129-176`), Ed25519-to-X25519 conversion (`x25519.go:296-333`), and a full HPKE layer over `cloudflare/circl` (`x25519.go:355-568`) even though `pkg/agent/hpke` exists (and calls back into `keys.HPKEDeriveSharedSecretToPeer`, `hpke/client.go:225`, `hpke/server.go:254`). `algorithms.go:28-98` registers RFC 9421 names via `init()` with `log.Fatalf`. |
| `crypto/chain` | Chain abstraction: provider interface, registry, chain->key-type mapping. | Mostly. Address heuristics (`utils.go:169-206` `ParseAddress`) and a second, hard-coded chain->key mapping (`utils.go:104-116`) duplicate `key_mapper.go`. Blank-imports `keys` for its `init()` side effect (`key_mapper.go:25`). |
| `chain/ethereum` | Ethereum `ChainProvider`. | Two unrelated things in one package: the stateless `Provider` (`provider.go`) and `EnhancedProvider` (`enhanced_provider.go`), an RPC client wrapper with retry/gas/receipt-polling that depends on `deployments/config` (`enhanced_provider.go:33`). `SignTransaction`/`VerifySignature` are stubs (`provider.go:131,152`). |
| `chain/solana` | Solana `ChainProvider`. | Respected; `SignTransaction` is a stub (`provider.go:129`). |
| `crypto/formats` | JWK/PEM import/export. | Respected. Owns the JWK struct and a JWK thumbprint (`jwk.go:472-509`, unused). |
| `crypto/storage` | `KeyStorage` backends (file, memory). | Respected; file backend hard-wires JWK (`file.go:59-60`). |
| `crypto/vault` | Passphrase-encrypted byte vault. | Isolated: no non-test consumer anywhere in the module; does not use `crypto.KeyPair`/`KeyStorage`; redefines `ErrKeyNotFound` (`secure_storage.go:40`). `MemoryVault` "encrypts" with XOR (`secure_storage.go:326-330`) and panics on empty passphrase (modulo by zero at `:329`). |
| `crypto/rotation` | `KeyRotator` implementation. | Respected; only Ed25519/Secp256k1 rotate (`rotator.go:80-87`), P-256/RSA/X25519 rejected. `RotationInterval`/`MaxKeyAge` are unused (`rotator.go:45`). |
| `internal/cryptoinit` | Wire concrete constructors into root `crypto` at init. | Exists solely to break the `crypto` <-> `keys/formats/storage` cycle created by `Manager`. |

## 2. Exported API surface (names only)

**crypto (root)** — Types: `KeyType`, `KeyFormat`, `KeyRotationConfig`, `KeyRotationEvent`, `AlgorithmInfo`, `Manager`. Interfaces: `KeyPair`, `KeyExporter`, `KeyImporter`, `KeyStorage`, `KeyRotator`, `KeyManager`. Funcs: `NewManager`, `Set{KeyGenerators,StorageConstructors,FormatConstructors}`, `New/Generate{Ed25519,Secp256k1,P256}KeyPair`, `NewMemoryKeyStorage`, `New{JWK,PEM}{Exporter,Importer}`, `RegisterAlgorithm`, `GetAlgorithmInfo`, `ListSupportedAlgorithms`, `ListRFC9421SupportedAlgorithms`, `GetRFC9421AlgorithmName`, `GetKeyTypeFromRFC9421Algorithm`, `Supports{RFC9421,KeyGeneration,Signature}`, `IsAlgorithmSupported`, `GetKeyTypeFromPublicKey`, `ValidateAlgorithmForPublicKey`. Errors: 7 in `types.go:154-162` + 2 in `algorithm_registry.go:62-63`.

Should be unexported/removed (no non-test consumer outside the defining file): `KeyManager` (no implementer; `Manager` lacks `GetExporter/GetImporter/GetStorage/GetRotator`), `ErrKeyExists`, `Set*Constructors` (only `cryptoinit`), `GetAlgorithmInfo`, `ListSupportedAlgorithms`, `Supports*`, `IsAlgorithmSupported` (rfc9421 defines its own, `rfc9421/types.go:98`), `GetKeyTypeFromPublicKey`, `Manager.SetStorage/ExportKeyPair/ImportKeyPair`. `Manager` itself is only held by `core.Core` (`core/core.go:37,47,106`) and referenced from `lib/export.go:54`; `Core.GenerateKeyPair`/`GetCryptoManager` have no non-test callers.

**keys** — Types: `X25519KeyPair` (exported struct; all other pairs unexported). Funcs: `Generate{Ed25519,Secp256k1,P256,X25519,RSA}KeyPair`, `New{Ed25519,Secp256k1,P256,X25519,RSA}KeyPair(priv, id)`, `EncryptWithEd25519Peer`, `DecryptWithEd25519Peer`, `HPKE*` (8 funcs). Unused anywhere including tests: `HPKESealAndExportToX25519Peer`, `HPKEOpenAndExportWithX25519Priv`, `X25519KeyPair.DecryptWithX25519`. `publicKeyOnlyEd25519/RSA` (`constructors.go:105-183`) are dead, nolint-suppressed.

**chain** — Types: `ChainType`, `Network`, `Address`. Interfaces: `ChainProvider`, `ChainRegistry`, `PublicKeyResolver`, `ChainKeyTypeMapper`. Funcs: `NewRegistry`, `RegisterProvider`, `GetProvider`, `ListProviders`, `GenerateAddresses`, `NewChainKeyTypeMapper`, `GetRecommendedKeyType`, `GetSupportedKeyTypes`, `ValidateKeyTypeForChain`, `GetRFC9421Algorithm`, `AddressFromKeyPair`, `GetSupportedChainsForKey`, `GetKeyTypeForChain`, `ValidateKeyForChain`, `FormatAddress`, `ParseAddress`. Unused: `PublicKeyResolver` (no implementer, no consumer), `FormatAddress`, `GetKeyTypeForChain` (internal use only), `NewRegistry` outside package.

**chain/ethereum** — `Provider`, `NewProvider`, `EthClient`, `EnhancedProvider`, `NewEnhancedProvider[WithClient]`, `ExecuteWithRetry`, `EstimateGas`, `SuggestGasPrice`, `GetTransactionOpts`, `WaitForTransaction`, `HealthCheck`, `GetClient`, `GetConfig`, `Close`. `EnhancedProvider` is used only by `tests/integration/blockchain_test.go`.

**chain/solana** — `Provider`, `NewProvider`.

**formats** — `JWK`, `JWK.ComputeKeyIDRFC9421` (unused), `New{JWK,PEM}{Exporter,Importer}`.

**storage** — `NewFileKeyStorage`, `NewMemoryKeyStorage`.

**vault** — `SecureVault`, `EncryptedKeyData`, `FileVault`, `MemoryVault`, `NewFileVault`, `NewMemoryVault`, 5 errors. Entire package test-only.

**rotation** — `NewKeyRotator` (consumer: `cmd/sage-crypto/rotate.go:81`).

## 3. Interfaces

| Interface | Defined | Implementers | Consumers (non-test) | Notes |
|---|---|---|---|---|
| `crypto.KeyPair` | `types.go:47` | 7 in `keys` (+2 dead) | module-wide | Consumer-side; fine. |
| `crypto.KeyStorage` | `types.go:86` | `storage.fileKeyStorage`, `memoryKeyStorage` | `rotation`, `Manager`, `cmd/*` | Test double: `wrappers_test.go:51`. |
| `crypto.KeyExporter/Importer` | `types.go:68,77` | `formats.jwk*/pem*` | `storage/file.go`, `internal/session_creator.go`, `handshake`, `oidc/auth0` | Doubles in `wrappers_test.go:89,99`. |
| `crypto.KeyRotator` | `types.go:116` | `rotation.keyRotator` | `cmd/sage-crypto/rotate.go` | Single impl, no double. |
| `crypto.KeyManager` | `types.go:136` | none | none | Dead interface. |
| `chain.ChainProvider` | `chain/types.go:74` | ethereum, solana | `chain` registry/utils, `cmd/sage-crypto/address.go` | Double: `registry_test.go:190`. |
| `chain.ChainRegistry` | `chain/types.go:100` | `defaultRegistry` (same package) | only via package-level globals `registry.go:116-136` | Interface defined next to its only impl; no double. |
| `chain.ChainKeyTypeMapper` | `key_mapper.go:29` | `defaultKeyMapper` (same file) | `did/factory.go:45` | Next-to-impl; no double. |
| `chain.PublicKeyResolver` | `chain/types.go:115` | none | none | Dead. |
| `ethereum.EthClient` | `enhanced_provider.go:37` | `*ethclient.Client` | `EnhancedProvider` | Consumer-side; double `MockEthClient` (`enhanced_provider_test.go:37`). |
| `vault.SecureVault` | `secure_storage.go:48` | `FileVault`, `MemoryVault` | none | Next-to-impl. |

## 4. Dependency direction problems

1. **`pkg -> internal`**: `pkg/agent/core/core.go:29` blank-imports `internal/cryptoinit`. Any external module importing `pkg/agent/core` transitively depends on an `internal/` package, and any external user of `crypto.Manager` without importing `core` gets a `panic("... not initialized")` (`wrappers.go:73,80,99,112-146`). Seven `examples/` and five test files also blank-import `cryptoinit`.
2. **Mechanism of the cycle hack**: `keys`, `formats`, `storage` import `crypto` for the interfaces. `crypto.Manager` (`manager.go:43-103`) wants to call `keys.Generate*`, `formats.New*`, `storage.NewMemoryKeyStorage`, which would form `crypto -> keys -> crypto`. Instead `wrappers.go` holds nil function variables, `cryptoinit.init()` (`init.go:28-47`) assigns them, and `crypto.New*` wrappers panic if unset. The linkage is by import side-effect and is invisible to the compiler.
3. **`init()` global registries**: `keys/algorithms.go:28` registers algorithms into a process-wide map (`algorithm_registry.go:59`), and `chain/key_mapper.go:25` blank-imports `keys` purely to trigger it. `chain/ethereum/provider.go:165` and `chain/solana/provider.go:168` self-register into `chain.globalRegistry` (`registry.go:116`); `cmd/sage-crypto/main.go:28-29` blank-imports both. `RegisterProvider` errors are discarded (`_ =`).
4. **Lower layer -> config layer**: `chain/ethereum/enhanced_provider.go:33` imports `deployments/config` (a deployment/config package) from a crypto primitive package.
5. **Name collisions across layers**: `crypto.NewEd25519KeyPair()` generates a key (`wrappers.go:71`) while `keys.NewEd25519KeyPair(priv, id)` wraps an existing one (`constructors.go:35`); same for Secp256k1/P256.

## 5. Duplication / SSOT violations

| Concern | Location A | Location B |
|---|---|---|
| Chain -> key type mapping | `chain/key_mapper.go:55-75` | `chain/utils.go:104-116` (`GetKeyTypeForChain`); `did/factory.go:78-101` repeats the did.Chain->chain.ChainType switch twice. |
| Ethereum address derivation | `chain/ethereum/provider.go:70-80` | `did/utils.go:236-247` (identical Keccak code); `did/ethereum/client.go:132` uses `ethcrypto.PubkeyToAddress`. |
| Key ID derivation (`sha256(pub)[:8]` hex) | `keys/ed25519.go:46-47`, `secp256k1.go:50-52`, `x25519.go:60-62`, `rs256.go:48-50`, `p256.go:54-59` | repeated again in `keys/constructors.go:40-41,57-59,75-77,92-93` and `p256.go:79-84`. |
| P-256 uncompressed-point marshal | `keys/p256.go:54-57` | `keys/p256.go:79-82`; `did/utils.go:60-64`. |
| Algorithm name strings | registry: `keys/algorithms.go:34,48,62,90` (`ed25519`, `es256k`, `ecdsa-p256-sha256`, `rsa-pss-sha256`) | `rfc9421/types.go:85-88` (`EdDSA`, `ES256K`, `ECDSA-secp256k1`); `formats/jwk.go:86,99,112,127,135`; `cmd/sage-crypto/sign.go:234-241`. Note registry advertises RSA as `rsa-pss-sha256` but `keys/rs256.go:79` signs PKCS#1 v1.5. |
| `KeyType` enums | `crypto/types.go:30-36` (strings) | `did/types_v4.go:37-39` (uint8, `KeyTypeECDSA`); mapping spread over `cmd/sage-did/helpers.go:189-207`, `cmd/sage-did/key.go:261-266`. |
| JWK parsing | `formats/jwk.go:236-281` | `cmd/sage-did/helpers.go:180-207` (hand-rolled `map[string]interface{}` parse). |
| secp256k1 coordinate padding | `formats/pem.go:166-179` pads X/Y to 32 bytes | `formats/jwk.go:96-97,180-181` does not pad (`big.Int.Bytes()`), so JWK `x`/`y` may be shorter than 32 bytes for some keys. |
| secp256k1 sign/verify | `keys/secp256k1.go:77-116` (go-ethereum) | `did/key_proof.go:68,153`, `did/a2a_proof.go:112,252`, `hpke/signature_verifier.go:83-95` call `ethcrypto` directly instead of `KeyPair.Sign/Verify`. |
| `ErrKeyNotFound` | `crypto/types.go:155` | `vault/secure_storage.go:40` (distinct value; `errors.Is` across packages fails). |
| Chain->config | `chain.Network` constants (`chain/types.go:42-56`) | `deployments/config.BlockchainConfig` (`deployments/config/blockchain.go:31`) carries `ChainID`/`NetworkRPC` separately; no link between the two. |

## 6. Call/response conventions

- **Errors**: Sentinels in `crypto/types.go` and `chain/types.go`, wrapped with `%w` in `chain`, `storage`, `vault`, `enhanced_provider`. Inconsistent elsewhere: bare `errors.New`/`fmt.Errorf` without sentinels in `formats` (`jwk.go:78,91`), `keys/constructors.go:127`, `solana/provider.go:142` (`"invalid signature"` instead of `crypto.ErrInvalidSignature`), `manager.go:52` (`"unsupported key type"` instead of `ErrInvalidKeyType`), `algorithm_registry.go:212,239`. `registry.go:103` and `utils.go:58` use `==` instead of `errors.Is`.
- **Context**: `ChainProvider.GetPublicKeyFromAddress` takes `ctx` (`chain/types.go:86`) but `GenerateAddress`/`SignTransaction` do not; `EnhancedProvider` accepts `ctx` but `retryWithBackoff` sleeps without honoring it (`enhanced_provider.go:293-313`), and `WaitForTransaction` hard-codes 5 min / 3 s (`:219,222`). Nothing else in scope uses `context`.
- **Constructors**: mixed. `NewX(...)` returning interface (`storage`, `formats`, `rotation`, `chain.NewRegistry`, `NewProvider`), returning concrete pointer (`NewManager`, `NewEnhancedProvider`, `NewFileVault`), plus setter-style config (`Manager.SetStorage`, `KeyRotator.SetRotationConfig`) and config-struct injection (`NewEnhancedProvider(cfg)`). No functional options anywhere. Zero-value structs are usable for `ethereum.Provider`/`solana.Provider` but not for `Manager`.
- **Panics**: `wrappers.go` panics on missing init; `keys/algorithms.go:40-96` `log.Fatalf` at init. The only logging in scope is that `log` import; no `internal/logger` or `internal/metrics` coupling in scope.
- **Concurrency**: `sync.RWMutex` used consistently in registries/storage; `Manager` has no lock but delegates to locked storage.

## 7. Refactoring candidates

| # | What | Why | Risk | Files |
|---|---|---|---|---|
| 1 | Delete `crypto.Manager` + `wrappers.go` + `internal/cryptoinit`; give `core.Core` a `KeyStorage`/generator func injected by the caller, or move `Manager` to a new package that imports `keys/formats/storage` directly. | Removes the pkg->internal violation, the panic-on-missing-init trap, and the 14 blank imports. `Manager` has no real callers beyond `core.Core` fields. | Low-Mid: `core.Core.GenerateKeyPair`, `lib/export.go:54`, examples that call `crypto.Generate*` (4 example files) must switch to `keys.Generate*`. | `pkg/agent/crypto/{manager.go,wrappers.go}`, `internal/cryptoinit/init.go`, `pkg/agent/core/core.go`, `lib/export.go`, `examples/a2a-integration/*` |
| 2 | Replace `init()`-based algorithm registration with a static table in root `crypto` (no `keys` dependency needed: it is pure metadata) and drop the blank import in `chain/key_mapper.go:25`. | Removes `log.Fatalf` at init and the hidden ordering dependency. | Low. | `pkg/agent/crypto/algorithm_registry.go`, `pkg/agent/crypto/keys/algorithms.go`, `pkg/agent/crypto/chain/key_mapper.go` |
| 3 | Fix `GetKeyTypeFromPublicKey` to distinguish P-256 via `Curve` (and align RSA registry name with the PKCS#1 v1.5 implementation, or switch signing to PSS). | Current behavior rejects valid `ecdsa-p256-sha256` signatures in `rfc9421/verifier_http.go:194` and misadvertises RSA. | Low for P-256; RSA change is wire-visible. | `pkg/agent/crypto/algorithm_registry.go:202-214`, `pkg/agent/crypto/keys/{algorithms.go,rs256.go}` |
| 4 | Move `EnhancedProvider` + `EthClient` out of `chain/ethereum` into the DID/ethereum layer or a new chain-client package; drop the `deployments/config` import from crypto. | Separates stateless address logic from RPC client behavior; fixes layering. | Low: only `tests/integration/blockchain_test.go` uses it. | `pkg/agent/crypto/chain/ethereum/enhanced_provider.go` |
| 5 | Collapse `chain/utils.go` `GetKeyTypeForChain`/`ValidateKeyForChain` onto `ChainKeyTypeMapper`; add a `did.Chain -> chain.ChainType` helper to remove the duplicated switches in `did/factory.go`. | One mapping source. | Low. | `pkg/agent/crypto/chain/{utils.go,key_mapper.go}`, `pkg/agent/did/factory.go` |
| 6 | Extract a `keys.KeyID(pubBytes []byte) string` helper and reuse from all `Generate*`/`New*` constructors; add `MarshalP256Uncompressed`. | 9 copies of the same hash/hex snippet. | Very low. | `pkg/agent/crypto/keys/*.go` |
| 7 | Split `keys/x25519.go`: keep `X25519KeyPair` + `DeriveSharedSecret` in `keys`; move ECIES (`Encrypt*/Decrypt*`) and all `HPKE*` funcs into `pkg/agent/hpke` (which already owns HPKE). Remove unused `HPKESealAndExport*`, `HPKEOpenAndExport*`, `DecryptWithX25519`. | `keys` should not depend on `circl/hpke`; clarifies ownership. | Mid: `handshake/{client,server}.go` and `hpke/{client,server}.go` import paths change. | `pkg/agent/crypto/keys/x25519.go`, `pkg/agent/hpke/*`, `pkg/agent/handshake/*` |
| 8 | Route `did/utils.go:236-247` Ethereum address derivation through `chain/ethereum.Provider.GenerateAddress` (or export a `ethereum.AddressFromPublicKey`). | Single implementation of address derivation. | Low. | `pkg/agent/did/utils.go`, `pkg/agent/crypto/chain/ethereum/provider.go` |
| 9 | Either delete `crypto/vault` or make it implement `crypto.KeyStorage` with passphrase-protected serialization; at minimum reuse `crypto.ErrKeyNotFound`, replace XOR `MemoryVault`, guard empty passphrase. | Zero consumers; `MemoryVault` gives a false sense of encryption and can panic. | Low (test-only today). | `pkg/agent/crypto/vault/secure_storage.go` |
| 10 | Remove dead exports: `KeyManager`, `PublicKeyResolver`, `ErrKeyExists`, `publicKeyOnly*` types, `JWK.ComputeKeyIDRFC9421`, `FormatAddress`; unify `IsAlgorithmSupported` (crypto vs rfc9421). | API surface hygiene. | Very low. | `pkg/agent/crypto/{types.go,algorithm_registry.go}`, `chain/{types.go,utils.go}`, `keys/constructors.go`, `formats/jwk.go` |
| 11 | Zero-pad secp256k1 `x`/`y`/`d` to 32 bytes in JWK export (match `pem.go:166-179`). | RFC 7518 requires fixed-length coordinates; interop with external JWK consumers. | Low; round-trip inside SAGE unaffected. | `pkg/agent/crypto/formats/jwk.go:96-98,180-181` |

Uncertainties: whether external SDK consumers (`sdk/*`) depend on `crypto.Generate*` wrappers or `internal/cryptoinit` behavior was not checked (outside scope); item 1 should be validated against `sdk/` and `lib/` before removal.
