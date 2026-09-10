# Architecture Analysis: handshake / hpke / session / transport / internal (sessioninit)

## 0. Two findings that dominate

[Critical][High] `pkg/agent/session/session.go:251-291` (`deriveDirectionalKeys`) never fills `keyMaterial[0:64]`, so every session created through the HPKE path (`NewSecureSessionFromExporterWithRole`) has an all-zero `signingKey` and a nil `aead`; `SignCovered`/`VerifyCovered` (660-676) then HMAC with a zero key, and `EncryptAndSign`/`DecryptAndVerify` (568, 605) nil-dereference. All four are `session.Session` interface methods. No test combines `FromExporterWithRole` with these methods (verified by grep). Established by reading, not executed.

[Major][High] `handshake.NewServer` starts a cleanup goroutine (`server.go:136`) with no non-test stop; `StopCleanupLoop` exists only in `helper_test.go:54`. `session.Manager.SetDefaultConfig` (`manager.go:382`) writes without the lock. The 4-phase handshake verifies signatures but never checks `MessageControlHeader.Nonce`/`Timestamp`, so Invitation/Request/Complete are replayable within `pendingTTL` (15 min).

## 1. Package roles and the two handshakes

| Package | Role (one sentence) | Respected? |
|---|---|---|
| `core/message` | 42-LOC envelope header (`MessageControlHeader`, `BaseMessage`, `ControlHeader` iface). | Yes, but only `handshake` and `core/message/*` use it; `hpke` uses ad-hoc maps. |
| `transport` | Protocol-agnostic `MessageTransport` + `SecureMessage`/`Response` + URL-scheme selector + mock. | Mostly; `RegisterHTTPFactory` is an empty placeholder (`selector.go:73-76`), `TransportGRPC` has no impl. |
| `transport/http`, `transport/websocket` | Client+server adapters for `SecureMessage`. | Yes, but they each own an identical wire codec (Section 5). |
| `session` | AEAD session (`SecureSession`), lifecycle `Manager`, replay `NonceCache`. | Yes, plus dead `Metadata`/`GenerateSalt` and metrics coupling. |
| `handshake` | 4-phase (Invitation/Request/Response/Complete) Ed25519-bootstrap key agreement that emits `Events` and does not create sessions (`server.go:52-53`). | Yes, but effectively unused: only non-test importer is `internal/session_creator.go`. |
| `hpke` | 1-RTT HPKE-Base(X25519)+E2E-DH key agreement; server creates and binds the session itself (`server.go:290-308`). | Yes; used by `tests/integration/basic_test.go:172-193`. |
| `internal` (file `session_creator.go`, `package sessioninit`) | Adapter implementing `handshake.Events`+`KeyIDBinder` that turns handshake callbacks into `session.Manager` calls. | Only user is `handshake/server_test.go:37`. |

**Relationship.** The two handshakes share consumers, not code: both take `transport.MessageTransport`, `did.Resolver`, `sagecrypto.KeyPair`, `*session.Manager`, and emit `transport.SecureMessage` with a `TaskID` discriminator (`"handshake/<n>"` at `handshake/utils.go:27`, `"hpke/complete@v1"` at `hpke/types.go:21`). They diverge at session creation: handshake -> `session.Params` -> `NewSecureSession` (single AEAD, labelled "legacy single-AEAD path" at `session.go:491,529`); hpke -> `EnsureSessionFromExporterWithRole` (directional AEADs). They do not import each other. Docs call 4-phase "Traditional, mature" and HPKE "recommended for new projects" (`docs/handshake/README.md:11-12`); top-level `README.md:153` documents only HPKE. Nothing in code marks handshake `Deprecated`. Conclusion: handshake is de facto legacy, undeclared.

## 2. Exported API surface (grouped) and suspicious exports

**handshake**: `Phase` + consts, `Events`, `NoopEvents`, `KeyIDBinder`, four `*Message` types, `KeyInfo`, `Client`/`NewClient` + 4 phase methods, `Server`/`NewServer`/`HandleMessage`, `GenerateTaskID`.
Flag: `KeyInfo` (`types.go:184`) used only by `tests/helpers`; `GenerateTaskID` exported while `parsePhase` is not (asymmetric); `Server.sessionCfg` stored (`server.go:121`) but never read.

**hpke**: `Client`/`NewClient`/`Initialize`/`WithCookieSource`, `Server`/`ServerOpts`/`NewServer`/`HandleMessage`, `InfoBuilder`/`DefaultInfoBuilder`, `KeyIDBinder`, `CookieSource`, `CookieVerifier`, `SignatureVerifier`+`ECDSAVerifier`/`Ed25519Verifier`/`CompositeVerifier`, `TrafficKeys`/`DeriveTrafficKeys`, `MakeAckTag`, `HPKEInitPayload`/`Parse...WithEphC...`, `HPKEBaseInitPayload`/`ParseHPKEBaseInitPayloadFromJSON`, `DefaultInfo`/`DefaultExportContext`, `TaskHPKEComplete`.
Flag (no reference outside defining file, non-test): `HPKEBaseInitPayload` + parser (`common.go:320-375`), `DeriveTrafficKeys`/`TrafficKeys` (`common.go:100-117`), `DefaultInfo`/`DefaultExportContext` (`common.go:63-70`). `MakeAckTag`, `ParseHPKEInitPayloadWithEphCFromJSON` are intra-package only. Public struct fields `Client.DID`, `Server.DID`.

**session**: `Session` (14 methods), `Config`, `Status`, `Params`, `SecureSession` + 4 constructors, `DeriveSessionSeed`, `ComputeSessionIDFromSeed`, `Manager` (Create/Ensure*/Bind/Unbind/Get*/Remove/List/ReplayGuardSeenOnce/Stats/Close), `NonceCache`, `Metadata`/`MetadataBuilder`/`GenerateSalt`/`GeneralPrefix`.
Flag: `SecureSession.Reset`/`InitializeSession` are pool internals (`session.go:379,416`); `NewSecureSessionFromExporter` (145), `EnsureAndBindFromExporterWithRole` (`manager.go:135`), `UnbindKeyID` (248), whole `metadata.go` — no non-test callers. `EncryptWithAAD*`, `EncryptOutbound`, `DecryptInbound` exist only on the concrete type, not on `Session`.

**transport**: `MessageTransport`, `SecureMessage`, `Response`, `TransportType` consts, `TransportFactory`, `TransportSelector` (+`RegisterHTTPFactory` no-op), `DefaultSelector` (mutable global), `SelectByURL`/`Select`, `MockTransport`.
**transport/http**: `HTTPTransport` (+`WithClient`), `HTTPServer`, `MessageHandler`. **transport/websocket**: `WSTransport` (+`WithTimeouts`, `Connect`, `Close`), `WSServer` (+`WithOrigins`, `WithTimeouts`, origin mutators), `MessageHandler`.
**internal**: `Creator`, `NewCreator` (doc comment says "New", `session_creator.go:47`).

## 3. Interfaces

| Interface | Defined | Implementers | Consumers |
|---|---|---|---|
| `handshake.Events` (5) | `handshake/types.go:59` | `NoopEvents` (80), `internal.Creator` | `handshake.Server` (`server.go:217,266,280,308,320`); errors from `On*` are discarded |
| `handshake.KeyIDBinder` (1) | `types.go:98` | `internal.Creator` | runtime assertion `any(s.events).(KeyIDBinder)` (`server.go:322`) |
| `hpke.KeyIDBinder` (1) | `hpke/types.go:29` | none wired in repo (no `Binder:` in any test) | `ServerOpts.Binder` -> `server.go:301` |
| `hpke.InfoBuilder` (2) | `types.go:23` | `DefaultInfoBuilder` | Client (87-88), Server (234-238) |
| `hpke.CookieSource`/`CookieVerifier` | `types.go:34-42` | none | Client (112), Server (137) |
| `hpke.SignatureVerifier` | `signature_verifier.go:39` | ECDSA/Ed25519/Composite | `hpke.verifySignature` (`common.go:146`) |
| `session.Session` (14) | `session/types.go:28` | `SecureSession` only | zero non-test consumers outside `session`; `Manager` stores `Session` then type-asserts back to `*SecureSession` for pooling (`manager.go:313,442`) |
| `transport.MessageTransport` (1) | `interface.go:42` | Mock, HTTP, WS | `handshake.Client/Server`, `hpke.Client` |
| `core/message.ControlHeader` (3) | `message/types.go:38` | 4 handshake messages via 12 identical getters (`types.go:111-181`) | only `core/message/{order,dedupe,validator}`, which neither handshake nor hpke use |

The two `KeyIDBinder`s are textually identical; `internal.Creator` satisfies both structurally, but its `IssueKeyID` depends on `sidByCtx` populated by `OnComplete` (`session_creator.go:74-78,107-113`), which only the 4-phase server calls, so wiring it into `hpke` would always return `ok=false`. The server-side handler signature `func(ctx, *SecureMessage) (*Response, error)` is an undeclared interface: it appears as `http.MessageHandler` (`http/server.go:36`), `websocket.MessageHandler` (`websocket/server.go:36`), `handshake.Server.HandleMessage`, and `hpke.Server.HandleMessage`.

**The `internal/session_creator.go` + `internal/sessioninit` mechanism, precisely.** There is no registration and no global. `internal/session_creator.go` declares `package sessioninit` but lives in directory `internal/`; its import path is `github.com/sage-x-project/sage/internal`, aliased `sessioninit` in `handshake/server_test.go:37`. `internal/sessioninit/` contains only a README whose "File Structure" section claims the Go file lives there (stale). `Creator` is constructed explicitly (`NewCreator(sm)`) and passed as `events` to `handshake.NewServer`. It (a) mints an X25519 keypair in `AskEphemeral` and retains the private key per `ctxID` (`session_creator.go:85-103`), (b) in `OnComplete` derives the shared secret, sets `Params.SharedSecret`, calls `Manager.EnsureSessionWithParams`, deletes the ephemeral key and records `ctxID -> sid` (56-83), (c) in `IssueKeyID` generates `"session:"+rand` and calls `Manager.BindKeyID` (107-121). It is not an import-cycle workaround: it cannot live in `session` (`handshake/types.go:28` imports `session`, so `session -> handshake` would cycle), but it could live in `handshake`, which already imports `session`, `keys`, and `formats`. Placing it under `internal/` makes the 4-phase flow unusable by external module consumers, and in-module fan-in is 0 (only a test).

## 4. Dependency direction

- pkg -> internal: `handshake/client.go:29`, `handshake/server.go:33`, `session/manager.go:26`, `session/session.go:34` import `internal/metrics`, whose values are `promauto` globals on a package `Registry` (`internal/metrics/handshake.go:28-61`, `session.go:28-81`, `crypto.go:28`). Any importer of `session` links Prometheus and cannot substitute a recorder.
- In-scope graph: `core/message <- handshake`; `transport <- handshake, hpke, transport/http, transport/websocket`; `session <- handshake, hpke, internal`; `handshake` and `hpke` are siblings. `transport/http|websocket -> transport` only (clean). `session` imports nothing from `pkg` (clean apart from metrics).
- Globals/`init()`: `transport.DefaultSelector` (`selector.go:148`) is mutated by `init()` in `http/register.go:26` and `websocket/register.go:26`; registration therefore depends on blank imports. No other globals in scope.
- Constructors spawn goroutines: `handshake.NewServer` (`server.go:136`, no stop), `session.NewManager` (`manager.go:65`, stopped by `Close`), `NewNonceCache` (`nonce.go:41`).
- Inverted flow: `handshake.Server` pushes the Response back through a client-side `MessageTransport` (`server.go:342-378`) while also returning it; `hpke.Server` returns inline only.

## 5. Duplication / SSOT violations (verified)

| Item | Locations |
|---|---|
| Wire codec: `wireMessage`/`wireResponse` structs, `toWireMessage`, `fromWireResponse`, `fromWireMessage`, `toWireResponse`, `sendErrorResponse`, `MessageHandler`, `init()` | `http/client.go:172-230`, `http/server.go:36,135-209`, `websocket/client.go:297-355`, `websocket/server.go:36,235-280`, both `register.go`. Only difference: HTTP `fromWireMessage` overlays `X-SAGE-*` headers (`http/server.go:152-172`). |
| AEAD seal/open framing (`nonce||ct`) repeated 8 times | `session.go:482,520,557,585,616,639,680,701,720,740`; only `Encrypt`/`Decrypt` emit metrics. |
| handshake send path x5 | `client.go:48,90,138,186` + `server.go:342`; vary only in `TaskID`, `Role`, and bootstrap encryption. |
| Signature verification | `handshake/server.go:382` (ed25519 or `Verify` iface) vs `hpke/common.go:128` (adds secp256k1 via `CompositeVerifier`); handshake cannot verify Ethereum-keyed agents. |
| Nonce/replay stores (5) | `session.NonceCache` (`nonce.go:27`, per-keyid, GC goroutine, caller `ReplayGuardSeenOnce` has no in-scope user); `hpke.nonceStore` (`common.go:154`, O(n) sweep per call); `core/message/nonce.Manager` (used by `rfc9421/verifier.go:39` and `message/validator`); `core/message/dedupe.Detector` (content hash); `pkg/storage.NonceStore` (`storage/interface.go:54`, memory+postgres, unused by all of the above). handshake uses none. |
| Session config defaults | `handshake/server.go:106-110` (MaxMessages 10000, never applied) vs `session/manager.go:47-51` and `withDefaults` (1000). |
| `zeroBytes` | `hpke/common.go:86`, `session.go:449` (closure), `session.go:389-398` loops. |
| Random-ID helpers | `internal.randBase64URL:123`, `session.GenerateSalt:85`, `nonce.GenerateNonce:53`, `uuid.NewString` as nonce (`hpke/client.go:104`). |
| KDF idioms | `hpke.hkdfExpand` (`common.go:206`) is HMAC counter-mode, not RFC 5869 Expand (no `T(i-1)` chaining), while `session` uses `x/crypto/hkdf`. Not a break, but misnamed. |

## 6. Call/response conventions and inconsistencies

- **Envelope layering**: `transport.SecureMessage` (no JSON tags) -> per-transport `wireMessage` (snake_case) -> `Payload` bytes. handshake payloads are JSON structs embedding `MessageControlHeader` (camelCase). hpke payloads are `map[string]any` built by hand (`client.go:247-256`), parsed back as `map[string]string` (`common.go:265-318`); the signed response uses a struct for canonical bytes but a map for output (`server.go:320-360`).
- **Error surfacing**: HTTP server maps handler errors to `{success:false,error}` with HTTP 200 (`http/server.go:208`). HTTP/WS clients return both a non-nil `Response` and non-nil `error` for transport failures (`http/client.go:119-124`, `websocket/client.go:150-176`) but `(Response{Success:false}, nil)` for non-200 (`http/client.go:145-153`). `hpke.Client` checks `resp.Success` (`client.go:300`); `handshake.Client` ignores it and counts `HandshakesCompleted{success}` on any non-error `Send` (`client.go:85`).
- **Swallowed errors**: `_ = s.events.On*` (`server.go:217,280,308,320`), best-effort unmarshal (304), `v, _ := get("v")` chain (`hpke/client.go:350-360`). `hpke.Client.createAndBindSession` dereferences `sessMgr` before its nil check (`client.go:501` vs `510`).
- **Logging**: `fmt.Printf` to stdout in `session/manager.go:307,400,435,459` and seven transport sites, while `internal/logger` exists.
- **Context**: transports, handlers, `Events` take `ctx`; `session.Manager`, `Session`, `KeyIDBinder`, `CookieVerifier` do not.
- **Constructors/config**: `handshake.NewServer` = 6 positional params incl. `*session.Config` (unused); `hpke.NewServer` = positional + `ServerOpts`; `hpke.Client` = fluent `WithCookieSource`; transports = `NewX`/`NewXWithClient`/`WithTimeouts`/`WithOrigins`; `session.NewManager()` no options.
- **Metrics**: inline Prometheus globals in handshake and session; none in hpke; handshake increments "completed" once per phase per side.

## 7. Refactoring candidates (this scope)

| # | What | Why | Risk | Files |
|---|---|---|---|---|
| 1 | Fix `deriveDirectionalKeys` to derive `encryptKey`/`signingKey` (or make legacy methods error when `aead==nil`); add a test using `FromExporterWithRole` + `SignCovered`. | Zero HMAC key / nil-deref on HPKE sessions. | Medium (only affects currently-broken path). | `session/session.go`, `session_test.go` |
| 2 | Move wire codec + `Handler` func type into `transport` (`wire.go`); http/ws call it. | Exact duplicates; wire JSON is one contract. | Low. | `transport/*`, `http/*.go`, `websocket/*.go` |
| 3 | Single `KeyIDBinder` in `session` (both packages already import it); keep aliases. | Duplicate interface. | Low. | `handshake/types.go`, `hpke/types.go` |
| 4 | Move `SignatureVerifier`/`CompositeVerifier` to `pkg/agent/crypto`; use from both servers. | handshake lacks secp256k1; two verify paths. | Medium (handshake gains ECDSA acceptance). | `hpke/signature_verifier.go`, `hpke/common.go`, `handshake/server.go` |
| 5 | Inject a small metrics recorder interface (nop default) or move metrics to `pkg/`. | pkg -> internal; Prometheus forced on library users. | Low-Medium; also touches `pkg/health`, `cmd/metrics-demo`. | `session/*.go`, `handshake/*.go`, `internal/metrics` |
| 6 | Private `seal/open` helpers in session; one `send()` in handshake used by 4 client methods + server. | Structural duplicates. | Low. | `session/session.go`, `handshake/client.go`, `server.go` |
| 7 | Move `internal/session_creator.go` into `pkg/agent/handshake` (no cycle) or into `internal/sessioninit/`; fix README. | Path/package mismatch; unusable outside module. | Low. | `internal/session_creator.go`, `internal/sessioninit/README.md` |
| 8 | Decide handshake status: mark `Deprecated` or harden (nonce/timestamp replay check, use `sessionCfg`, exported `Stop()`, drop `ControlHeader` getters if validator unused). | Undeclared legacy with replay gap and goroutine leak. | Medium. | `handshake/*` |
| 9 | Remove dead exports listed in Section 2; delete `RegisterHTTPFactory` placeholder. | Surface hygiene. | Low (breaking only for unknown external users). | as listed |
| 10 | Lock in `SetDefaultConfig`; replace `fmt.Printf` with a logger; consolidate replay stores on `session.NonceCache` or `storage.NonceStore`. | Race; five replay implementations. | Low-Medium. | `session/manager.go`, `hpke/common.go`, transports |

## Facts vs. opinions

**Fact**
- `deriveDirectionalKeys` (`session.go:251-291`) allocates a zeroed 192-byte buffer, points `signingKey` at `[32:64]`, and only HKDF-fills `[64:192]`; `NewSecureSessionFromExporterWithRole` (120-141) never calls `deriveKeys` or sets `aead`; `EncryptAndSign`/`DecryptAndVerify`/`SignCovered`/`VerifyCovered` use `s.aead`/`s.signingKey` (568, 605, 661, 668).
- No test file references both `FromExporterWithRole` and `SignCovered|EncryptAndSign|DecryptAndVerify|VerifyCovered`.
- `StopCleanupLoop` is defined only in `handshake/helper_test.go:54`.
- Only non-test importer of `pkg/agent/handshake` is `internal/session_creator.go`; `internal/session_creator.go` declares `package sessioninit`; `internal/sessioninit/` contains only `README.md`.
- No `Binder:` field is set in any test or non-test file for `hpke.ServerOpts`.
- `transport/http` and `transport/websocket` wire structs and four codec functions are textually identical (Section 5 line refs).
- `handshake.Server` never reads `Nonce`/`Timestamp` from decoded messages; `Server.sessionCfg` is written at `server.go:121` and never read.

**Opinion**
- [High] The zero-`signingKey`/nil-`aead` path will panic or produce non-secret MACs at runtime for any caller using those interface methods on an HPKE session; not executed to confirm.
- [High] `handshake` is de facto legacy; removing or deprecating it removes ~950 LOC plus `internal/session_creator.go` and the `core/message` coupling with no in-module functional loss.
- [Mid] `internal.Creator` was placed under `internal/` for layering intent rather than to break a cycle, since `handshake` already imports everything `Creator` needs.
- [Mid] `hpke.hkdfExpand` is interoperable only with itself (both sides use it), so the misnomer is a naming/audit issue rather than a protocol break.
