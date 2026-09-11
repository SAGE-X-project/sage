# SAGE feature map (from entry points)

Generated 2026-09-11 from `docs/refactoring/graph/entrypoints.md` (call/ref reachability from every `main`, cobra handler, cgo export and example) plus the three Go repositories in the SAGE-X-project organisation that import this module. Purpose: know what the code actually delivers before deciding what to keep, deprecate or move.

## 1. What the module is used for, by whom

| Consumer | How it uses SAGE | Evidence |
|---|---|---|
| `cmd/sage-crypto` (operator CLI) | key generation, listing, signing, verification, rotation, chain address derivation | reaches `crypto/{keys,formats,storage,rotation,chain,chain/ethereum,chain/solana}` |
| `cmd/sage-did` (operator CLI) | 3-phase registration (`commit`, `register`, `activate`) through `AgentCardClient`; resolve/list/verify/update/deactivate/card/key subcommands through `did.Manager` | `commit/register/activate` reach `did/ethereum` + bindings; the other 13 subcommands reach `did.Manager`, which has no wired client (defect D1) |
| `cmd/sage-verify`, `cmd/deployment-verify` | health probes, deployment config sanity | reach `pkg/health`, `deployments/config` only |
| `sage-a2a-go` (external, v1.5.2, builds against current main) | the real "SDK": A2A agents with DID identity, HPKE 1-RTT sessions, RFC 9421 signed HTTP, key files | imports `did` (27 files), `crypto` (14), `transport` (7), `hpke`, `session`, `rfc9421`, `did/ethereum` (`EthereumClient`, `AgentCardClient`), `crypto.Set*Constructors` |
| `sage-multi-agent` (external, pinned v1.3.1 via replace, builds against current main) | multi-agent demo: HPKE server/client, sessions, HTTP transport, AgentCard registration | imports `hpke`, `session`, `transport`, `transport/http`, `did`, `did/ethereum`, `crypto/formats` |
| `sage-adk` (external, 2025-10) | stale: imports the pre-v1.0 flat layout (`sage/crypto`, `sage/did`, `sage/config`) and no longer compiles | go.mod `sage v0.0.0` + replace |
| `examples/mcp-integration/*` | RFC 9421 request signing/verification, A2A card generation | reach `rfc9421`, `crypto/keys`, `did` |
| `lib/` (cgo) | nothing: exports a version string and two no-ops | no consumer in repo or organisation |

Packages no binary, example or external consumer reaches: `pkg/agent/handshake` (4-phase), `core/message/{dedupe,order,validator}`, `crypto/vault`, `pkg/storage/*`, `pkg/oidc/*`, `pkg/version`, `internal/logger`, `internal/cryptoinit` (reached only through the blank import in `core`).

## 2. Feature inventory (what works today)

| Feature | Entry | Works? | Notes |
|---|---|---|---|
| Key lifecycle (Ed25519, secp256k1, P-256, X25519, RSA), JWK/PEM, file/memory storage, rotation | `sage-crypto`, library | Yes | 4 copies of key-file loading in cmd; `sage-did update/deactivate` loads a random key instead of the file (D3) |
| Chain address derivation (Ethereum, Solana) | `sage-crypto address` | Yes | duplicated in `did/utils.go` |
| DID 3-phase registration on AgentCardRegistry (commit-reveal, KEM key, multi-key) | `sage-did commit/register/activate`, `AgentCardClient` | Yes | the only working write path; not an implementer of any `did` interface |
| DID resolution / listing / verification / update / deactivation via `did.Manager` | `sage-did resolve …`, `core.ConfigureDID` | No | `Manager.Configure` never installs a client (D1). External consumers bypass `Manager` and use `EthereumClient` directly as `did.Resolver` |
| Legacy `EthereumClient` (V2-shaped) | library, `sage-did` | Reads yes, writes no | packs V2 signatures against the AgentCard ABI (D4); still imported by both external consumers |
| A2A agent card generate/validate/proof, key proof-of-possession | `sage-did card/key`, library | Yes (library) | CLI path depends on `Manager` (D1) |
| HPKE 1-RTT handshake + directional AEAD sessions | library (`hpke`, `session`), external consumers | Yes | `SignCovered/EncryptAndSign` on HPKE sessions use a zero key (D2); nonce store is O(n) |
| 4-phase handshake (Invitation/Request/Response/Complete) | library only | Unused | no replay check, cleanup goroutine never stopped (D7) |
| RFC 9421 HTTP message signatures | library, examples, external | Yes | HTTP path has no replay check (D5); P-256 detection wrong (D8) |
| Transport (HTTP, WebSocket) with `SecureMessage` envelope | library, external | Yes | wire codec duplicated per transport |
| Health checks, Prometheus metrics | `sage-verify`, `metrics-demo` | Partly | `pkg/health` fails on Windows (D6); `/metrics` in `health.Server` serves an unfed collector |
| Persistence (`pkg/storage` memory/postgres), OIDC/Auth0, build version | library only | Untested / unread | zero consumers anywhere |
| Solidity contracts (AgentCardRegistry, hooks, ERC-8004, governance) | Hardhat | Yes, 202 tests | Go bindings regenerated manually; drift already present |
| Multi-language SDKs (Python, TS, Rust, Java) | standalone | No interop | target `/debug/*` and `/v1/a2a:sendMessage` routes that no server in the repo or organisation serves; no RFC 9421; TS ships zero session keys |

## 3. Reading

SAGE's delivered value is a Go library for DID-authenticated, HPKE-encrypted, RFC 9421-signed agent communication, consumed by `sage-a2a-go` and `sage-multi-agent`, plus operator CLIs and the AgentCardRegistry contracts. Everything else in the tree (4-phase handshake, cgo lib, storage, OIDC, four SDKs, random-test harness) is either unreachable or non-functional today. Decisions in `DECISIONS.md` follow from this: protect the library API that the two external consumers use, fix the CLI paths that are broken, and cut what nothing reaches.
