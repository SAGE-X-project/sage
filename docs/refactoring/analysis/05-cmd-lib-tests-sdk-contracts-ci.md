# SAGE Peripheral Architecture Analysis (cmd, lib, tests, tools, examples, sdk, contracts, build/CI)

Read-only analysis. Line numbers cite the working tree as of this run.

## 1. cmd/

| Binary | Role | pkg deps | Notes |
|---|---|---|---|
| `sage-crypto` (1,332 LOC, cobra) | keygen/sign/verify/address/rotate/list | `pkg/agent/crypto{,/keys,/formats,/storage,/rotation,/chain,/chain/ethereum,/chain/solana}` | No chain defaults; no version flag wiring to `pkg/version` |
| `sage-did` (3,115 LOC, cobra) | commit/register/activate, resolve/list/update/deactivate, key add/list/revoke/approve/verify-pop, card generate/validate/show, debug, verify | `pkg/agent/crypto{,/keys,/storage}`, `pkg/agent/did{,/ethereum}` | Only cmd with a `_test.go` (`register_test.go`), which re-tests `did.GenerateAgentDIDWithAddress` already covered by `pkg/agent/did/utils_test.go:30` |
| `sage-verify` (305 LOC, hand-rolled arg switch) | health/blockchain/system checks | `pkg/health`, `deployments/config` | `const version = "1.0.0"` (`main.go:30`); Korean UI strings (`main.go:154-218`) |
| `deployment-verify` (159 LOC) | print config + deployment JSON, dial RPC, check contract code | `deployments/config`, go-ethereum directly | Single 127-line `main`; reads `SageRegistryV2` field names (`main.go:63-64`), a pre-AgentCard naming |
| `metrics-demo` (160 LOC) | serves `/metrics` on :9090, simulates sessions | `internal/metrics`, `pkg/agent/session` | Demo, not a product binary; imports `internal/` |
| `random-test` (280 LOC) | CLI over `tests/random` fuzz harness | `tests/random` only | `//go:build integration` (`main.go:19`) but `Makefile:739` builds without `-tags` -> `make random-test*` fails ("build constraints exclude all Go files") |

**Business logic living in cmd/ that belongs in a library package**

- Key loading, triplicated in `sage-crypto`: `sign.go:106-169 loadKey` and `address.go:258-320 loadKeyForAddress` are byte-identical (JWK wrapper unwrapping + PEM/JWK import); `verify.go:140-197 loadPublicKey` is a third variant with a public-key fallback. The "wrapper format from `sage-crypto generate`" (`{private_key, public_key, key_id, key_type}`) is a file format defined only in cmd (`generate.go:107-138`) and only parseable by cmd.
- `sage-did/helpers.go:55-93 loadKeyPair` is a defective fourth variant: for `--key-format jwk|pem` it **generates a fresh random key** instead of importing (`helpers.go:78-84`: "simplified implementation"). It is reached by `update` (`update.go:86`) and `deactivate` (`deactivate.go:148`), so those commands cannot succeed with a file key. `pkg/agent/crypto/formats.NewJWKImporter/NewPEMImporter` already exist and are what sage-crypto uses.
- `sage-did/helpers.go:98-316` (`detectKeyType`, `detectPEMKeyType`, `detectJWKKeyType`, `parseKeyFile`, `parseJWKKey`, `parsePEMKey`, `isValidRawKey`): public-key format sniffing and JWK/PEM/raw -> `did.KeyType` + raw bytes. Used by `key add` (`key.go:272,280`). This duplicates the concern of `pkg/agent/crypto/formats` but returns `did.KeyType`; it belongs in `pkg/agent/did` or `formats`.
- `sage-did/card.go:343-434 validateCardWithDID` is a near-verbatim copy of `pkg/agent/did.ValidateA2ACardWithDID` (`pkg/agent/did/a2a.go:271-340`), differing only in constructing a `did.Manager` rather than accepting a `Resolver`. Card generation (`card.go:159-229`) correctly calls `did.GenerateA2ACard`.
- `sage-did/verify.go:64-161 runVerify` compares name/endpoint by hand and builds a `did.VerificationResult`; the comment at `verify.go:105-106` says "we'd add a VerifyMetadata method to the manager".
- `sage-did/commit.go:178-219` persists commitment state to `~/.sage/commitments/<hash>.json`; `createAgentCardClient` (`commit.go:162-176`) returns "key file loading not implemented yet" when `--key-file` is passed.
- `sage-crypto/verify.go:255-286 verifyWithPublicKey` re-implements Ed25519/ECDSA(SHA-256, r||s) verification even though `crypto.KeyPair.Verify` exists (`pkg/agent/crypto/types.go:61`). Whether it matches `KeyPair.Sign` hashing for secp256k1 was not verified here.
- `sage-verify/main.go:84-181` duplicates the env -> config -> `localhost:8545` fallback chain twice (`:91-101`, `:131-139`).

**Every "table of chain defaults" location**

| Location | Content |
|---|---|
| `cmd/sage-did/register.go:171-180 getDefaultRPCEndpoint` | Ethereum -> `https://eth-mainnet.g.alchemy.com/v2/your-api-key` (placeholder), Solana -> mainnet-beta |
| `cmd/sage-did/register.go:182-199 getDefaultContractAddress` | Ethereum -> `0x000…000`, Solana -> `1111…` (placeholders); comment points to `contracts/DEPLOYED_ADDRESSES.md`, which does not exist. Called from 13 sites across resolve/list/update/deactivate/key/card/verify |
| `cmd/sage-did/{register,commit,activate}.go:71,85,67` | cobra flag default `http://localhost:8545` (3-phase commands, inconsistent with the mainnet default above) |
| `cmd/sage-verify/main.go:100,138` | `http://localhost:8545` |
| `deployments/config/blockchain.go:43-71 NetworkPresets` | `local`(31337), `kairos`(1001), `mainnet`(**8217 Kaia Cypress**, not Ethereum) — no Sepolia despite `pkg/agent/did/types.go:87 NetworkEthereumSepolia` and CLAUDE.md |
| `deployments/config/blockchain.go:172-185 readContractAddress` | "placeholder — actual implementation would parse JSON"; returns `0x4Ba6Fc82…` for any path containing "kairos" |
| `deployments/config/deployment_loader.go:108-135 GetContractAddress` | second copy of the kairos `0x4Ba6Fc82…` fallback |
| `contracts/ethereum/hardhat.config.js:36-220` | 15 networks with env-first public RPC fallbacks (the most complete table) |
| `examples/a2a-integration/0{1..4}/main.go`, `examples/agent-initialization/main.go:406-412` | `0xDc64a140…`, `localhost:8545`, Hardhat account-0 private key, 5 copies |
| `examples/config.yaml:7,18-22` | `0x5FbDB231…` / `0xe7f1725E…` (differs from the Go examples) |
| `.env.example:149`, `.env.e2e.example:17` | `0x5FbDB231…`, Sepolia `0xb25D5f59…` |
| `tools/scripts/test/sage-cli-workflow.sh:51-53` | RPC + Hardhat key/address |

The Sepolia addresses in CLAUDE.md (`0xC7eCF7Ad…` etc.) appear in **no** code, config, or deployment JSON; `contracts/ethereum/deployments/` holds only `hardhat-latest.json` and `deployments/*.json` is gitignored (`contracts/ethereum/.gitignore:208`).

Layering note: `pkg/agent/crypto/chain/ethereum/enhanced_provider.go:33` imports `deployments/config` (pkg -> deployments), in addition to the pkg -> internal violations already in the graph summary.

## 2. lib/

- Single file `lib/export.go` (60 LOC), `package main`, `import "C"`. Exports three symbols: `SageVersion` (returns hard-coded `"1.3.1"`, `:32`), `SageInit` (returns 0, no-op), `SageCleanup` (no-op). Blank-identifier references to `core.NewVerificationService`, `crypto.NewManager`, `did.NewManager` (`:52-56`) exist only to force linking; **no crypto, signing, DID or session function is exposed to C callers**. No header file, no C tests, no README.
- Build: `Makefile:84-213` (`build-lib`, static/shared, 5 platform targets); `Dockerfile:30` runs `make build-lib || true` (failure swallowed). Release bundle target `package: build-all-platforms build-lib-all` (`Makefile:797`) exists but `release.yml:65` only runs `make build-platform`.
- CI: no workflow references `build-lib`, `c-shared`, or `lib/`. `go vet ./...` / `go test ./...` compile it as a normal package, that is all.

## 3. tests/ and tools/

**Real tests vs harnesses**

| Path | Kind |
|---|---|
| `tests/integration/*_test.go` (14 files, ~5.3k LOC) | Real tests. **None carries `//go:build integration`** — only `e2e_sepolia_test.go:1` has a tag (`e2e`). The `-tags=integration` in `Makefile:473` and `integration-test.yml:96` is a no-op; these tests also run under plain `go test ./...` (`test.yml:38`) and self-skip via `SAGE_RPC_URL`/`SKIP_BLOCKCHAIN_TESTS` (`test_helper.go:59-78`). CI exports `ETHEREUM_RPC_URL` (`integration-test.yml:90`), which they do not read. |
| `tests/blockchain_verification_test.go` (587 LOC) | Real test, hardcodes `localhost:8545` (`:45,58,217`). |
| `tests/helpers/testhelpers.go` (206 LOC) | Logging/assert sugar + `SaveTestData` writing `testdata/verification/*.json` (gitignored). **Imported by 32 `pkg/**/_test.go` files** (e.g. `pkg/agent/hpke/hpke_test.go:28`) — the library's unit tests depend on `tests/`. |
| `tests/testutil/environment.go` (271 LOC) | `MockEthereumServer` returns constant `"0x1"` for every JSON-RPC call (`:239-250`); `CreateTestContract` returns `0x…0001`. One importer (`tests/integration/did_integration_enhanced_test.go:30`). Overlaps `test_helper.go` skip logic but uses different env vars (`SKIP_INTEGRATION`/`ETHEREUM_RPC_URL` vs `SKIP_BLOCKCHAIN_TESTS`/`SAGE_RPC_URL`). |
| `tests/random/` (4 files, 1,564 LOC, package `random`) | Library for `cmd/random-test` (`main.go:35`). Imports **zero** SAGE packages; every executor is a simulation returning `Passed = Expected.ShouldPass` (`executor.go:161-178`). No `_test.go`. Layering oddity confirmed (cmd depends on tests/), and the harness cannot detect a real defect. |
| `tools/analyze` | `package main`, stdlib only; benchmark JSON -> markdown. Invoked only by `tools/scripts/run-benchmarks.sh:143`. |
| `tools/benchmark` | Test-only package (3 `_bench_test.go`); reached via `run-benchmarks.sh:15`; not in CI. |
| `tools/codegraph` | Separate module (`tools/codegraph/go.mod`); produced `docs/refactoring/graph/`. Not in Makefile/CI. |
| `tools/loadtest` | k6 JS; `loadtest.yml:107,216,299` launches `tests/handshake/server/main.go`, **which does not exist**; trigger path `loadtest/**` (`:10`) no longer matches `tools/loadtest/**`. |
| `tools/scripts` (20 scripts + 4 in `test/`) | Makefile glue. `verify_rfc9421_ed25519.sh:17` reads a fixture path that does not exist. `test/sage-cli-workflow.sh:52` embeds the Hardhat account-0 private key. |

Repo-root `testdata/verification/` (193 JSON, gitignored) is written by `helpers.SaveTestData` and read by nothing.

## 4. examples/

All examples are in the root module. `a2a-integration/01-04`, `agent-initialization`, `simple-agent-init` carry `//go:build ignore` and are compiled by nothing. `mcp-integration/{multi-agent,performance-benchmark}` are README-only. `make build` compiles the 7 mcp-integration binaries (`Makefile:300-303`); no workflow builds or runs any example.

APIs exercised: `rfc9421.NewHTTPVerifier().SignRequest` (basic-demo `:207-224`, client `sage_client.go:88-105`, simple-standalone `:155-172`); `did.GenerateA2ACard/ValidateA2ACard` (02/03); `crypto.Generate*KeyPair` (01); `did.NewManager`/`RegistryConfig`.

Copies of library logic:
- `agent-initialization/main.go:77-158` and `simple-agent-init/main.go:87-188`: hand-rolled x509/PEM key persistence and `ecdsa.GenerateKey`/`ed25519.GenerateKey` instead of `storage.NewFileKeyStorage` / `formats.NewPEMExporter` / `keys.Generate*`; near-duplicates. `agent-initialization/main.go:269` uses deprecated `elliptic.Marshal`.
- `vulnerable-vs-secure/secure-chat/main.go:42-61` and `simple-standalone/main.go:46-67`: "verification" only checks header presence and accepts everything; neither imports `rfc9421`. The secure-chat output claims "identity verified via blockchain DID" (`:120`).
- `basic-tool/calculator_tool.go:67`: `RPCEndpoint: "https://eth-mainnet.example.com"`.

## 5. Non-Go inventory

**SDKs** (none built or tested in CI; Makefile touches `sdk/` only in `clean`, `Makefile:662-667`)

| SDK | Files / LOC | Version | Claims vs reality | Tests |
|---|---|---|---|---|
| Python `sdk/python/sage_client/{client,crypto,did,session,types,exceptions}.py` | ~1,344 | `0.1.0` | httpx client to `/debug/kem-pub`, `/debug/server-did`, `/debug/register-agent`, `/v1/a2a:sendMessage` (`client.py:136-372`) — **no Go server in this repo serves these routes**. "HPKE" is X25519+HKDF(no salt)+AES-GCM with counter nonce (`crypto.py:154-156,234-240,265`). DID resolver is an in-memory dict (`did.py:93-131`). No RFC 9421. | 7 (`tests/test_crypto.py`) |
| TypeScript `sdk/typescript/src/{client,crypto,session,types,index}.ts`, `react/hooks.tsx` | ~1,244 | `1.0.0` | README claims RFC 9421 (`README.md:10`) — only an unused `HTTPSignature` type (`types.ts:62-66`). No network layer, no HPKE. Session keys are all-zero placeholders (`session.ts:52-53`) and derived keys are discarded (`client.ts:142-151`); server signature never verified (`client.ts:138-139`). `ethers`/`jose` deps unused. License MIT vs LGPL elsewhere. | **0** (vitest configured) |
| Rust `sdk/rust/sage-client/src/{client,crypto,session,types,did,error,lib}.rs` | ~1,128 | `0.1.0` | reqwest client to the same `/debug/*` routes (`client.rs:69-230`); ad-hoc "HPKE" matching Python; DID parse only, no resolver. | 7 inline |
| Java `sdk/java/sage-client/src/main/java/com/sage/client/*` (17 files) | ~1,519 | `0.1.0` | OkHttp client to same routes; "HPKE" uses raw X25519 secret as AES key with no KDF and random nonces (`Crypto.java:150-152,210-226`) -> **incompatible with Python/Rust**. `target/` build output checked in. | 2 classes |

Signing in Python/Rust/Java is an ad-hoc `"{client}|{server}|{msg}|{ts}"` Ed25519 string (`client.rs:166`, `SageClient.java:204`), not RFC 9421, so none interoperates with the Go core.

**contracts/ethereum** (`pragma solidity 0.8.20` everywhere, Hardhat v3, OpenZeppelin)

| Contract | Role | LOC |
|---|---|---|
| `AgentCardRegistry.sol` | main registry: multi-key, commit-reveal, ERC-8004 | 793 |
| `AgentCardStorage.sol` (abstract) | storage layout | 354 |
| `AgentCardVerifyHook.sol` | registration hook | 273 |
| `erc-8004/standalone/ERC8004{Identity,Reputation,Validation}Registry.sol` | ERC-8004 registries | 255/373/693 |
| `governance/TEEKeyRegistry.sol`, `SimpleMultiSig.sol`, `TimelockController.sol` | governance | 827/342/19 |
| interfaces (5) | | 617 |
| `AgentCardStorageTest.sol` (in production dir), `test/ReentrancyAttacker.sol` | test helpers | 301/97 |

"202 tests" = exact `it(` count across 9 JS test files (5,419 LOC); `foundry.toml` exists but `test/foundry/` and any `.t.sol` do not; `forge` is never run. `package.json:28,42,55` reference network `arbitrumOne`, defined as `arbitrum` (`hardhat.config.js:171`).

**Go bindings — SSOT**: only one location exists, `pkg/blockchain/ethereum/contracts/agentcardregistry/` (`AgentCardRegistry.go` 3,770, `AgentCardStorage.go` 1,395, `IRegistryHook.go` 223; ABI-only, no `Bin`). `contracts/ethereum/bindings/go` **does not exist**, yet `test.yml:117` and `security.yml:86-90` still exclude it. Sole importer: `pkg/agent/did/ethereum/agentcard_client.go:40`. No `abigen` invocation anywhere — regeneration is manual, and drift exists: `AgentCardStorage.go` lacks the `KEMKeyUpdated` event present in the compiled artifact. No bindings exist for VerifyHook, ERC-8004, or governance contracts.

**contracts/solana**: 2 Anchor 0.29 programs (`sage-registry/src/lib.rs` 486, `sage-verification-hook/src/lib.rs` 281, plus 951 lines of Rust tests). Program IDs are placeholders. No `Anchor.toml`, so `scripts/deploy.sh` cannot run. Never built/tested in CI.

## 6. Build / CI

| Concern | Makefile | Workflows | Gap |
|---|---|---|---|
| Go unit | `test` (`go test -v ./...`) | `test.yml:38` `go test -race ./...` | tests/integration runs here too (no tag) |
| Integration | `test-integration` (Hardhat via `setup_test_env.sh`) | `integration-test.yml:94-98` with Hardhat node | tag is no-op; env var mismatch (`ETHEREUM_RPC_URL` vs `SAGE_RPC_URL`) |
| E2E | `test-e2e*` -> `./tests/integration/... -tags=e2e` | `integration-test.yml:146-148` runs `./test/e2e/...` — **path does not exist** | broken job |
| Binaries | `build` (3 bins + 7 examples) | `test.yml:158`, `release.yml:65` (`build-platform`) | ok |
| cgo lib | `build-lib*` | none; Dockerfile `|| true` | never verified |
| Contracts | none | `test.yml:75-77` npm test; `security.yml` Slither (`|| true`), solhint | no `make` entry; coverage disabled |
| Solana, SDKs, examples, random-test, codegraph | — | — | none anywhere |
| Docker | `docker-build` (script) | `docker.yml` builds on push to main/dev + PR | PR step rebuilds image a second time (`docker.yml:75-80`) |
| Loadtest | `loadtest` | `loadtest.yml` | server path missing; trigger path stale |
| Go toolchain | `go.mod` 1.25.2, CI 1.25.2 | `Dockerfile:5` `golang:1.26.3` (after PR 210) | mismatch, relies on `GOTOOLCHAIN=auto` |

**Version SSOT**: `VERSION`=1.5.2 -> injected via ldflags (`Makefile:24-32`) into `pkg/version.Version` (default also `"1.5.2"`). Divergent copies: `lib/export.go:32` `"1.3.1"`, `cmd/sage-verify/main.go:30` `"1.0.0"`, `contracts/ethereum/package.json:3` `1.5.0`, SDKs 0.1.0/1.0.0/0.1.0/0.1.0, Solana 0.1.0. `tools/scripts/update-version.sh` updates only VERSION, README, contracts package.json/lock, `version.go`, `lib/export.go` — and has evidently not been run since 1.3.1 for lib. No cmd binary calls `pkg/version`, so the ldflags injection reaches nothing user-visible. `docs/INDEX.md:120-127` describes `deployment-verify` as "RFC 9421 HTTP signature verification", which is wrong; `docs/ARCHITECTURE.md` has no section on cmd/, lib/, or sdk/.

## 7. Refactoring candidates (this scope)

| # | What | Why | Risk | Files |
|---|---|---|---|---|
| 1 | Extract a key-file loading helper (or extend `storage`): one `LoadKeyPair(path, format, storageDir, keyID)` handling the sage-crypto wrapper JSON, JWK, PEM, storage; make `sage-did` use it | 4 copies, one of which silently generates random keys (`helpers.go:78-84`) — a correctness/security defect affecting `update`/`deactivate` | Low | `cmd/sage-crypto/{sign,address,verify}.go`, `cmd/sage-did/{helpers,deactivate,update}.go` |
| 2 | Single chain-defaults table keyed by `did.Network`, with `NetworkPresets` gaining Ethereum mainnet/Sepolia; delete `getDefaultRPCEndpoint/ContractAddress` and both `0x4Ba6…` fallbacks | 13 call sites depend on placeholders; `deployments/config` "mainnet" is Kaia; CLAUDE.md addresses exist nowhere in code | Medium | `cmd/sage-did/register.go:171-199`, `deployments/config/{blockchain,deployment_loader}.go` |
| 3 | Replace `cmd/sage-did/card.go:343-434` with `did.ValidateA2ACardWithDID`; move `runVerify` comparison into `did.Manager.VerifyMetadata`; move `helpers.go:98-316` key sniffing into `pkg/agent/did` | Exact duplicate of library code; untested in cmd | Low | `cmd/sage-did/{card,verify,helpers,key}.go`, `pkg/agent/did/a2a.go` |
| 4 | Move `tests/helpers` -> `internal/testutil`; merge `tests/testutil` env-skip logic with `tests/integration/test_helper.go` on one env-var set | 32 pkg unit tests import from `tests/`; two incompatible skip conventions | Low | `tests/helpers/testhelpers.go`, `tests/testutil/environment.go`, `tests/integration/test_helper.go`, 32 `_test.go` |
| 5 | Add `//go:build integration` to blockchain-dependent `tests/integration` files, or drop the tag from Makefile/CI; fix `Makefile:739` to pass `-tags=integration` for random-test, or delete `tests/random`+`cmd/random-test` (simulation only) | Tag is currently meaningless; random-test cannot build; harness tests nothing | Low | `tests/integration/*.go`, `Makefile:465-479,733-786`, `.github/workflows/integration-test.yml:96` |
| 6 | Add `make bindings` running `abigen` from Hardhat artifacts, plus a CI drift check; remove stale `contracts/ethereum/bindings` exclusions | Manual regeneration already produced ABI drift (`KEMKeyUpdated`); 7 of 9 contracts have no bindings | Low | `Makefile`, `.github/workflows/{test,security}.yml`, `pkg/blockchain/ethereum/contracts/agentcardregistry/` |
| 7 | Either give `lib/` a real C API (sign/verify/resolve wrappers + header + CI build on Linux) or remove `lib/`, `build-lib*` targets, and `Dockerfile:30` | Exports nothing usable; version string stale; never built in CI | Low either way | `lib/export.go`, `Makefile:79-213,797`, `Dockerfile:30` |
| 8 | Make `pkg/version` the only version source: read it in `sage-verify`/`lib`, add `--version` to cobra roots, extend `update-version.sh` to SDK manifests | 9 distinct version values today; ldflags injection reaches no binary output | Low | `cmd/sage-verify/main.go:30`, `lib/export.go:32`, `cmd/{sage-crypto,sage-did}/main.go`, `tools/scripts/update-version.sh` |
| 9 | Fix CI dead paths: `./test/e2e/...`, `tests/handshake/server/main.go`, `loadtest/**` trigger; align Dockerfile Go version with go.mod | Jobs cannot pass as written | Low | `.github/workflows/{integration-test,loadtest}.yml`, `Dockerfile:5` |
| 10 | Either compile the `//go:build ignore` examples in CI or move them to docs; fix the two "secure" examples to call `rfc9421.VerifyRequest`; dedupe `agent-initialization`/`simple-agent-init` | Six examples rot silently; two examples demonstrate no-op verification under a "secure" label | Low | `examples/**` |
| 11 | SDKs: decide scope. Minimum: mark TS/Java as experimental, fix TS zero-key session and Java missing KDF, add test jobs. Interop with the Go core requires RFC 9421 plus the missing server routes, or removing those claims | Four SDKs advertise features they do not have; TS ships as 1.0.0 with placeholder encryption keys | High effort if implemented; low if scoped down | `sdk/**` |

Uncertain items (not verified): whether `sage-crypto verify`'s ECDSA path is hash-compatible with `secp256k1KeyPair.Sign`; whether the CLAUDE.md Sepolia addresses correspond to a real deployment.
