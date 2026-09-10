# Architecture Analysis: core / rfc9421 / message / storage / health / oidc / version / logger / metrics

## 1. Package roles and actual consumers

| Package | Role (one sentence) | Respected? | Module-internal consumers (non-test) |
|---|---|---|---|
| `pkg/agent/core` | Facade (`Core`) + `VerificationService` composing DID resolution with rfc9421 verification. | Mostly; `core.go:33 const Version = "0.1.0"` is a stale third version source. | `lib/export.go` (only `_ = core.NewVerificationService`), `examples/mcp-integration/basic-tool`. |
| `core/rfc9421` | RFC 9421 HTTP Message Signatures (sign/verify/canonicalize/parse) plus a non-RFC "SAGE Message" envelope. | Two distinct models coexist in one package (see §6). | `core`, 4 `examples/mcp-integration/*`, `tests/integration`. |
| `core/message` | Shared control-header contract (`ControlHeader`, `MessageControlHeader`, `BaseMessage`). | Yes. | `handshake/types.go:107-168` (embeds), `dedupe`, `order`, `validator`. |
| `core/message/nonce` | In-memory TTL nonce set. | Yes. | `rfc9421/verifier.go:43-56`, `validator`. |
| `core/message/dedupe` | In-memory TTL set of SHA-256(header) hashes. | Yes. | `validator` only. |
| `core/message/order` | Per-session monotonic sequence/timestamp check; `ResultBuilder` is unused outside its own test. | Partially. | `validator` only. |
| `core/message/validator` | Composes nonce+dedupe+order+timestamp. | Yes, but **no non-test consumer anywhere** (grep of `core/message/validator"` -> zero importers). | none |
| `pkg/storage` (+memory, postgres) | Persistence abstraction for sessions/nonces/DIDs. | Interface respected. **Zero Go importers outside itself**; referenced only in `docs/DATABASE.md:226-280`, `api/examples/sessions.md:372-398`; no `_test.go` at all. Schema matches `deployments/migrations/000001_initial_schema.up.sql:5-45`. | none (library-only surface) |
| `pkg/health` | Health checks + HTTP health/metrics server. | Partially: `system.go:56-57` uses `syscall.Statfs` without build tag, so `GOOS=windows go build ./pkg/health` fails (`undefined: syscall.Statfs_t`) while `Makefile:53 PLATFORMS=linux darwin windows` builds `sage-verify` via `build-platform` (Makefile:244-247). | `cmd/sage-verify/main.go:105-282` (Checker/CheckBlockchain/CheckSystem only; **never `NewServer`/`StartHealthServer`**), `tests/integration/health_test.go`. |
| `pkg/oidc` (+auth0) | Sentinel errors + Auth0 client-credentials agent and JWKS-backed verifier. | Yes; `oidc.ErrNoPublicKey/ErrInvalidSigningAlg/ErrInvalidIssuer` are defined but `auth0.go:253,259,237` re-create them inline. | none outside `pkg/oidc/auth0` (library-only). |
| `pkg/version` | Build-info holder populated by ldflags. | **Wired but never read**: `Makefile:30-34` injects `-X pkg/version.*`, but no Go file calls `version.Get/String/Short/...`. `cmd/sage-verify/main.go:30 const version = "1.0.0"` and `lib/export.go:33 "1.3.1"` are separate stale values. `MAIN_BUILD_LDFLAGS` (Makefile:38-40, used at :258/:269/:280) targets `main.Version` symbols that no `cmd/*` declares. | none (dead in-module; library-only) |
| `internal/logger` | Home-grown JSON structured logger + `SageError`. | Only consumer is `pkg/health/server.go:36,42,69,73,194`. `SageError`/`ErrCode*`/package-level `Debug/Info/...` are consumed by nothing. | `pkg/health` |
| `internal/metrics` | Two systems: (a) Prometheus vectors on a private `Registry` (`registry.go:33`, `promauto.With(Registry)` in crypto/handshake/message/session.go), (b) legacy `MetricsCollector` (`collector.go:27-274`). | (a) used; (b) `Record*` methods are **never called** outside the package, so `health/server.go:148-186 /metrics` always serves zeros. `MessagesProcessed/ReplayAttacksDetected/NonceValidations` (`message.go`) have no producers. | `handshake/{client,server}.go`, `session/{manager,session}.go`, `pkg/health/server.go`, `cmd/metrics-demo`. |

Verdict: **dead** (in-module) = `core/message/validator`, `order.ResultBuilder`, `metrics.MetricsCollector`, `logger.SageError`/global funcs, `pkg/version`, `health.Server`. **Library-only surface** (documented, plausible external use) = `pkg/storage/*`, `pkg/oidc/*`.

## 2. Exported API surface (grouped) and should-be-unexported flags

- **core**: `Core` + 12 methods (`core.go:36-118`), `VerificationService{VerifyAgentMessage,VerifyMessageFromHeaders,QuickVerify}`, `VerificationResult`, `DIDResolver`, `Version`. Flag: `Version` const; getters `GetCryptoManager/GetDIDManager/GetVerificationService` leak internals.
- **rfc9421**: types `Message`, `VerificationOptions`, `VerificationResult`, `SignatureAlgorithm` consts, `HTTPVerificationOptions`, `SignatureInputParams`; `Verifier` (`NewVerifier`, `NewVerifierWithNonceManager`, `VerifySignature`, `VerifyWithMetadata`, `ConstructSignatureBase`, `VerifyHTTPRequest`, `SignHTTPRequest`); `HTTPVerifier` (`SignRequest`, `VerifyRequest`); `Canonicalizer.BuildSignatureBase`; `BodyIntegrityValidator.ValidateContentDigest`, `IsComponentCovered`, `ComputeContentDigest`; `ParseSignatureInput`, `ParseSignature`; `MessageBuilder` (12 fluent methods), `ParseMessageFromHeaders`; `GetSupportedAlgorithms`, `IsAlgorithmSupported`. Flag: `ConstructSignatureBase` and `Canonicalizer` are implementation details exposed for tests.
- **message**: `MessageControlHeader`, `BaseMessage`, `ControlHeader`.
- **nonce**: `Manager{IsNonceUsed,MarkNonceUsed,GetUsedNonceCount}`, `GenerateNonce`. **dedupe**: `Detector{IsDuplicate,MarkPacketSeen,GetSeenPacketCount}`. **order**: `Manager.ProcessMessage`, `GetNextSequence` (global atomic `manager.go:30`), `Result`, `ResultBuilder`. **validator**: `MessageValidator{ValidateMessage,GetStats}`, `ValidationResult`, `ValidatorConfig`, `DefaultConfig`.
- **storage**: `Session`, `Nonce`, `DID`, `SessionStore`, `NonceStore`, `DIDStore`, `Store`. **memory**: `Store` (+`Clear`), `SessionStore`, `NonceStore`, `DIDStore` types exported though only reachable via `Store` accessors (`store.go:61-73`) — should be unexported. **postgres**: same plus `Config`; `NonceStore.Get` (`nonces.go:119`) is outside the interface.
- **health**: `Status` consts, `HealthStatus`, `BlockchainHealth`, `SystemHealth`, `Checker`, `CheckBlockchain`, `CheckSystem`, threshold consts (`system.go:29-32`), `Server`, `NewServer`, `StartHealthServer`.
- **oidc**: 12 `Err*` vars. **auth0**: `Config`, `Agent.RequestToken`, `VerifierConfig`, `NewVerifier` returning **unexported** `*verifier` (`auth0.go:163`, golint violation), `LoadOrCreateKeyPair` (writes to `./testdata/` via package var `keyPath` `generate_keys.go:33` — test helper leaked into production API).
- **version**: `Version/GitCommit/GitBranch/BuildDate/GoVersion` vars, `Info`, `Get`, `String`, `Short`, `UserAgent`, `GetModuleVersion`, `PrintVersion`, `PrintVersionJSON`. Flag: `String()`/`Short()` slice `GitCommit[:7]` without length check (`version.go:74,91`).
- **logger**: `Level`, `Field` + 6 constructors, `Logger`, `StructuredLogger` (+`SetPrettyPrint`, `SetTimeFormat`), `NewLogger`, `NewDefaultLogger`, `WithRequestID/WithTraceID`, `SageError`, `NewSageError`, 12 `ErrCode*`, `SetDefaultLogger/GetDefaultLogger`, `Debug/Info/Warn/ErrorMsg/Fatal`. Flag: `SageError` family belongs in an errors package, not a logger; `SetDefaultLogger` silently ignores non-`*StructuredLogger` (`logger.go:380-384`).
- **metrics**: `Registry`, `Handler`, `StartServer`, ~20 promauto vectors, `MetricsCollector`, `MetricsSnapshot`, `NewMetricsCollector`, `GetGlobalCollector`. Flag: `MetricsCollector` fields are exported and mutated under a private mutex (`collector.go:31-44`).

## 3. Interfaces

| Interface | Defined | Implementers | Consumers | Notes |
|---|---|---|---|---|
| `storage.Store/SessionStore/NonceStore/DIDStore` | `storage/interface.go:27-103` | `memory.*` , `postgres.*` | none in module | Two impls, zero consumers. |
| `logger.Logger` | `logger.go:121-132` | `StructuredLogger` only | `health.Server` | Single-impl; 9 methods, health uses only `Info`/`Error`. |
| `core.DIDResolver` | `verification_service.go:31-34` | `did.Manager` (`manager.go:182,190`) | `VerificationService` | Single impl in non-test code; no test mocks found by grep. Good seam. |
| `message.ControlHeader` | `message/types.go:38-42` | 4 handshake message types (`handshake/types.go:111-179`) via embedded `MessageControlHeader` (which itself has no getter methods — the getters are hand-written per type), plus test mocks | `dedupe`, `order`, `validator` | Consumed only by packages nothing uses; handshake implements it but never passes it to a validator. |

Never consumed: `storage.*` (all four), `validator` outputs. Single-implementation: `logger.Logger`, `core.DIDResolver`.

## 4. Dependency-direction problems

- **`pkg/health -> internal/logger`** (`server.go:29,36,42,69,73,194`): uses only `Logger.Info(string)` and `Logger.Error(string)` with no fields, plus `NewLogger(os.Stdout, InfoLevel)` in `StartHealthServer`. A 2-method local interface removes the import entirely; `StartHealthServer` would take the logger as a parameter.
- **`pkg/health -> internal/metrics`** (`server.go:30,148`): only `GetGlobalCollector().GetSnapshot()` for the legacy collector that nothing feeds. Replacing `handleMetrics` with `metrics.Handler()` (or accepting an `http.Handler` in `NewServer`) removes the import and fixes the always-zero output.
- **`pkg/agent/core -> internal/cryptoinit`** (`core.go:29`, blank import): side-effect registration of key generators / storage / format constructors into `pkg/agent/crypto` (`cryptoinit/init.go:29-46`). This is not replaceable by an interface; it exists to break a `crypto <-> keys/formats/storage` cycle. `core` is the only `pkg` importer; 7 examples and 6 test files import it directly.
- **`internal/metrics`** is a Prometheus wrapper: private registry `metrics.Registry` (`registry.go:33`) with Go/process collectors registered in `init()` (`registry.go:36-40`); all vectors are `promauto.With(Registry)` at package init. It is *not* registered on `prometheus.DefaultRegisterer`; exposure is only via `metrics.Handler()`/`StartServer` (`server.go:29-50`) or `cmd/metrics-demo:45`. `pkg/agent/session` and `pkg/agent/handshake` also violate `pkg -> internal`.

## 5. Duplication / SSOT violations

**Replay protection — four implementations**

| Impl | Key | Store | TTL / GC | Check+mark atomic? | Consumer |
|---|---|---|---|---|---|
| `message/nonce.Manager` | nonce string (global) | `map` + RWMutex | ctor TTL; goroutine ticker, never stopped (`manager.go:46,99-106`) | No (`IsNonceUsed` then `MarkNonceUsed`) | rfc9421 `Verifier`, validator |
| `message/dedupe.Detector` | SHA-256(seq,nonce,ts) | `map` + RWMutex | same pattern, unstoppable goroutine (`detector.go:46,104-111`) | No | validator |
| `storage.NonceStore` | nonce string + sessionID | memory map / Postgres tx | caller-supplied `expiresAt`; explicit `DeleteExpired` | Yes (`CheckAndStore`) | none |
| `session.NonceCache` | (keyid, nonce) | nested `sync.Map` | ctor TTL; stoppable ticker (`nonce.go:70-75`) | Yes (`Seen` = check-and-record) | `session.Manager.ReplayGuardSeenOnce` (`manager.go:345`) |

Overlaps: `nonce.Manager` and `dedupe.Detector` are the same code with a different key; `dedupe` is redundant when a nonce is present (hash includes nonce). `rfc9421.HTTPVerifier.VerifyRequest` (`verifier_http.go:111-189`) parses `params.Nonce` but performs **no** replay check, while `Verifier.VerifySignature` (`verifier.go:74-90`) does — the two verifier paths differ in security semantics. `nonce.Manager` is a global namespace; `session.NonceCache` is per-keyid. No shared interface exists.

**storage/memory vs storage/postgres**: 1:1 method duplication across 3 stores each; error strings duplicated verbatim (`"session not found: %s"` in `memory/store.go:136,154,167,224` and `postgres/sessions.go:90,131,147,226`; same for nonce/DID). Postgres uses `err == pgx.ErrNoRows` (`dids.go:78,194`, `sessions.go:89`, `nonces.go:134`) instead of `errors.Is`. Memory `Update` (`store.go:149-160`, `dids.go:66-77`) replaces the whole record; postgres `Update` writes a subset of columns (`sessions.go:112-116`, `dids.go:90-94`, omits `updated_at`). `Count` semantics match (non-expired only).

**Version sources (5+)**: `VERSION` file (1.5.2) -> Makefile -> `pkg/version.Version` default `"1.5.2"` (`version.go:31`, never read); `core.Version "0.1.0"` (`core.go:33`); `did.Version "0.1.0"` (`did/did.go:27`); `cmd/sage-verify` `"1.0.0"`; `lib/export.go` `"1.3.1"`; `contracts/ethereum/package.json` 1.5.0; `sdk/typescript/package.json` 1.0.0.

**Logging**: no `log/slog` anywhere. `internal/logger` used only by health; `pkg/agent/crypto/keys/algorithms.go`, `pkg/agent/transport/selector.go`, `cmd/*` use stdlib `log`; `oidc/auth0/auth0.go:119,321` and `generate_keys.go:50` use `fmt.Printf/Println`.

**Errors**: `pkg/oidc` uses sentinels; `auth0.go` bypasses three of them; `storage`, `core`, `rfc9421`, `health` use only `fmt.Errorf` strings (no sentinels, not `errors.Is`-able); `logger.SageError` is a third, unused scheme; `validator` returns errors inside a result struct.

## 6. Call/response conventions

- **Errors**: `fmt.Errorf("...: %w")` wrapping is consistent in storage/oidc/core. rfc9421 `VerifyWithMetadata` returns `(*VerificationResult, nil)` with `Valid=false` for failures (`verifier.go:97-141`) while `VerifySignature` returns `error` — two failure channels; `core.VerificationService` inherits both (`verification_service.go:63-69` returns result, `:59` returns error).
- **Context**: storage, oidc, core all take `ctx` first. rfc9421 `Verifier`/`HTTPVerifier` take none. `health.CheckBlockchain` creates its own `context.Background()` with 10 s timeout (`blockchain.go:52`) rather than accepting one. `nonce`/`dedupe` goroutines have no ctx or `Close`.
- **Constructors**: `NewX(deps)` in most places; `rfc9421.NewVerifier()` hard-codes `nonce.NewManager(5*time.Minute, 1*time.Minute)` (`verifier.go:43-48`) and spawns a goroutine per verifier; `core.NewVerificationService` hard-codes `rfc9421.NewVerifier()` (`verification_service.go:46`) so nonce policy cannot be injected from `core`. `auth0.NewVerifier` returns an unexported type. `health.StartHealthServer` is a convenience wrapper that fixes stdout/Info.
- **Config structs**: `postgres.Config`, `auth0.Config/VerifierConfig`, `validator.ValidatorConfig` (pointer with `DefaultConfig()`), `rfc9421.VerificationOptions`/`HTTPVerificationOptions` (pointer, nil -> defaults, `verifier.go:60-62`, `verifier_http.go:112-114`). Inconsistent: `QuickVerify` builds `&VerificationOptions{MaxClockSkew: 0}` to *disable* time checks (`verification_service.go:178-180`), relying on `>0` semantics.
- **rfc9421 verifier parameterisation**: algorithm selection is by string in `Message.Algorithm` (`verifier.go:175-215`) for the envelope path, but by key type + `sagecrypto.ValidateAlgorithmForPublicKey` for the HTTP path (`verifier_http.go:192-232`); RSA is supported only on the HTTP path; ASN.1 ECDSA is "not implemented" (`verifier_http.go:294`). `VerificationOptions.RequireActiveAgent`/`RequiredCapabilities` are interpreted by `core`, not by rfc9421, though they live in rfc9421.
- **Composition in core**: `VerifyAgentMessage` resolves metadata via `DIDResolver`, mutates the caller's `message.Metadata["capabilities"]` (`verification_service.go:78-81`), then calls `VerifyWithMetadata` with `expectedMetadata={endpoint,name}`. `VerifyMessageFromHeaders` copies `X-Metadata-*` headers into metadata and then *also* adds every header via `AddHeader` (`message_builder.go:159-161`). `QuickVerify` infers algorithm from DID chain (`:158-166`), bypassing `Message.KeyID`.

## 7. Refactoring candidates (this scope)

1. **Unify replay guards behind one interface.** Define `type ReplayGuard interface{ Seen(scope, nonce string) bool }` in `core/message`; make `session.NonceCache` and `nonce.Manager` satisfy it; delete `dedupe`. Inject into `rfc9421.Verifier` and `core.NewVerificationService`. Add `Close()` to `nonce.Manager` (goroutine leak). Risk: low-medium. Files: `message/types.go`, `message/nonce/manager.go`, `message/dedupe/*`, `rfc9421/verifier.go`, `core/verification_service.go`, `session/nonce.go`.
2. **Delete or promote `core/message/validator` and `order.ResultBuilder`.** Nothing consumes them; handshake implements `ControlHeader` but never validates. Either wire `validator` into `handshake/server.go` or remove (~400 LOC + 1.6k test LOC). Risk: low.
3. **Break `pkg/health -> internal/*`.** Replace `handleMetrics` with `metrics.Handler()` passed in as `http.Handler`; take a minimal local logger interface; delete `MetricsCollector` (`collector.go`) since no producers exist. Add `//go:build !windows` + a stub for `system.go`, or use `golang.org/x/sys`. Risk: low.
4. **Single version source.** Delete `core.Version`, `did.Version`, `cmd/sage-verify` `version`, `lib/export.go` literal; have those read `pkg/version`. Drop `MAIN_BUILD_LDFLAGS` (Makefile:36-40). Guard `GitCommit[:7]`. Risk: low.
5. **Move `internal/cryptoinit` to a public path (or remove).** Fixes the only `pkg -> internal` edge in `core` and lets SDK users register implementations without importing an internal path. Risk: low.
6. **Storage tidy-up.** Unexport `memory.{SessionStore,NonceStore,DIDStore}` and postgres equivalents; introduce `storage.ErrNotFound/ErrAlreadyExists/ErrExpired` sentinels and use `errors.Is(err, pgx.ErrNoRows)`; add a shared conformance test run against both backends (currently zero tests). Risk: low.
7. **oidc/auth0 API hygiene.** Return an exported `*Verifier` (or `oidc.Verifier` interface) from `NewVerifier`; use the existing `oidc.Err*` sentinels at `auth0.go:237,253,259`; move `LoadOrCreateKeyPair` to a `_test.go` or `testutil`. Risk: low.
8. **Extract `logger.SageError`/`ErrCode*` out of the logger** (or delete: zero consumers). Risk: none. File: `internal/logger/logger.go:322-374`.

Uncertain / not verified: whether external SDK consumers rely on `pkg/storage`, `pkg/version`, or `validator` (no in-repo evidence either way); whether `handshake` was intended to call `validator` (no TODO found).
