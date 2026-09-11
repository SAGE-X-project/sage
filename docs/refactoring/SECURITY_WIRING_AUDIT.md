# Security Wiring Audit — actual request paths

Scope: verify which security controls are on the request paths real consumers use (not the package inventory), per `analysis/01-crypto.md`, `03-handshake-hpke-session-transport.md`, `04-core-storage-health-oidc-internal.md`. `docs/refactoring/FEATURE_MAP.md` referenced in the task does not exist in the tree; consumers were identified by grep of non-test callers instead.

Line numbers refer to the working tree at commit `e98b42b`. Every claim marked **[tested]** was confirmed by a throwaway Go test module (created under the scratchpad, run with `go test`, then deleted; results quoted in §8). Everything else is by reading. Severity: `[치명]` / `[중요]` / `[권장]`. Confidence: `[High]` / `[Mid]` / `[Low]`.

## 0. Headline findings

| # | Finding | Sev | Conf |
|---|---|---|---|
| 1 | RFC 9421 HTTP path (`HTTPVerifier.VerifyRequest`) — the only signature path the MCP examples use — has **no replay check, no forward-skew bound, and does not require body coverage**. With the example component set a request body can be swapped after signing and still verifies. **[tested]** | [치명] | [High] |
| 2 | HPKE-derived sessions (`NewSecureSessionFromExporterWithRole`) have an **all-zero HMAC signing key and nil legacy AEAD**; `SignCovered/VerifyCovered` produce keyless MACs, `EncryptAndSign/DecryptAndVerify` nil-deref. **[tested]** | [치명] | [High] |
| 3 | Session layer (`Encrypt/Decrypt`) has **no ciphertext replay / ordering protection**; identical ciphertext decrypts repeatedly. `Manager.ReplayGuardSeenOnce` and `NonceCache` exist but have zero callers. **[tested]** | [중요] | [High] |
| 4 | `secp256k1` `KeyPair.Sign` hashes with Keccak-256; the rfc9421 envelope verifier (`Verifier.VerifySignature`, `core.QuickVerify`) hashes with SHA-256. Signatures made with the SDK key pair **cannot verify on the envelope path** for Ethereum DIDs (and vice versa on the HPKE path). **[tested]** | [중요] | [High] |
| 5 | `hpke.Server` (P1) is **not wired by any non-test code** in the module; the only consumer is `tests/integration/basic_test.go:172-193`. All of P1/P2's controls are therefore "implemented, exercised only in tests". | [중요] | [High] |
| 6 | Ethereum DID resolution picks the **first ECDSA key regardless of on-chain `verified` flag**, never surfaces Ed25519 keys, and has no per-key revocation concept in Go (`ethereum/client.go:380-410`). Active flag is checked. | [중요] | [High] |
| 7 | A2A card proof verification (`VerifyA2ACardProof`) uses the **key embedded in the card itself**, no DID resolution; `cmd/sage-did card verify` (`card.go:262`) calls only this. Self-signed forgeries pass. | [중요] | [High] |
| 8 | `core.VerificationService.VerifyAgentMessage(ctx, msg, nil)` **panics** (`verification_service.go:63`). **[tested]** | [권장] | [High] |

---

## 1. P1 — HPKE 1-RTT handshake server (`pkg/agent/hpke/server.go`)

Entry: `Server.HandleMessage` (`server.go:115`). Non-test callers of `hpke.NewServer`: **none** (grep). Test caller: `tests/integration/basic_test.go:172-177` (no `Binder`, no `Cookies`).

| Control | Implemented where | On path? | Gap / defect | Sev | Conf |
|---|---|---|---|---|---|
| Signature over exact payload bytes | `server.go:213-217` → `common.go:128-152` (`CompositeVerifier`: Ed25519 / secp256k1-Keccak) | YES | Verifies `msg.Payload` as-is (good). `msg.ContextID`, `msg.ID`, `msg.Role`, `Metadata` are **not signed**; ctxID is bound only indirectly via `info` string compare (`server.go:234-237`). | [권장] | [High] |
| DID ↔ payload binding | `server.go:224-226` (`senderDID == pl.InitDID`) | YES | `pl.RespDID` is **never compared to `s.DID`** — server accepts an init addressed to another DID, creates a session and binds a kid before the client's ackTag check would fail. Resource waste; not key compromise. | [권장] | [High] |
| DID active check | via `resolver.ResolvePublicKey` → `did/resolver.go:107-109` / `ethereum/resolver.go:42-44` (`ErrInactiveAgent`) | YES | Depends on resolver impl (see P6). | — | [High] |
| Key-type/algorithm binding | `signature_verifier.go:196-203` selects by Go key type | YES | No explicit `alg` field; key type from chain decides. ECDSA verifier tries 3 encodings (`:95-117`) — accepts DER and raw. Acceptable. | 결함 없음 | [High] |
| Timestamp / skew window | `server.go:227-230`, default ±2 min (`:95-97`) | YES | Symmetric window, fine. | 결함 없음 | [High] |
| Nonce uniqueness | `server.go:231-233` → `common.go:154-178` (`nonceStore`, key = `ctxID|nonce`, TTL 10 min) | YES | Check-and-mark is atomic (under mutex). **O(n) sweep on every call** and unbounded growth within TTL → CPU/memory DoS by flooding valid-signature inits. TTL (10 m) > skew (2 m): bound is correct. | [중요] | [High] |
| Replay window bound | nonce TTL 10 m ≥ 2×skew | YES | OK. | 결함 없음 | [High] |
| DoS cookie | `server.go:137-142`, `types.go:34-42` | NO: dead code | `Cookies` never set by any caller. Cookie check runs **after** DID resolution + signature verify (`:125` precedes `:137`), so it does not protect the expensive step it is meant to. | [권장] | [High] |
| Suite whitelist | `server.go:243-245` | PARTIAL | Compares a compile-time constant to the allow-list; the client cannot negotiate a suite, so this is config validation, not negotiation. | 결함 없음 | [High] |
| Session creation / binding | `server.go:290-308` → `session.Manager.EnsureSessionFromExporterWithRole` (`manager.go:81`) | YES | `sid = SHA256(label‖secret)[:16]` — identical on both peers; kid random UUID. `KeyIDBinder` (`Binder`) never wired (only impl is `internal.Creator`, which returns `ok=false` for hpke, see analysis 03 §3). | [권장] | [High] |
| Key zeroisation | `server.go:158,164-165,173,193` (`zeroBytes`) | YES | `combined` zeroed after use; `session.sessionSeed` keeps a copy until `Close` (`session.go:131`). | 결함 없음 | [High] |
| Constant-time compare | `common.go:92-98` (`isAllZero32`), client `hmac.Equal` | YES | Info/exportCtx compared with `string(a)!=string(b)` (`server.go:235,239`) — non-secret, fine. | 결함 없음 | [High] |
| Error information leak | `server.go:225` returns generic "authentication failed" for DID mismatch, but `:229,232,236,240` leak "ts out of window" / "replay detected" / "info mismatch"; transports forward `err.Error()` verbatim (`http/server.go:206`, `websocket/server.go:277`). | YES | Oracle for skew/replay state. Low impact. | [권장] | [High] |
| Request size limit | none in hpke; transport-level (see P5) | NO | — | see P5 | [High] |

## 2. P2 — HPKE client (`pkg/agent/hpke/client.go`)

Entry: `Client.Initialize` (`client.go:79`). Non-test callers: none (test: `basic_test.go:191-193`).

| Control | Implemented where | On path? | Gap / defect | Sev | Conf |
|---|---|---|---|---|---|
| Peer KEM key resolved via DID + active check | `client.go:195-221` → `ResolveKEMKey` | YES | KEM key is raw bytes from chain (`ethereum/client.go:422`); no proof-of-possession for the KEM key exists in Go (`key_proof.go:74-77` explicitly excludes X25519). | [권장] | [High] |
| ackTag (key confirmation, HMAC over transcript) | `client.go:156-163` → `common.go:208-234`; `hmac.Equal` | YES | Transcript binds info, exportCtx, enc, ephC, ephS, both DIDs, ctx, nonce, kid. Good. | 결함 없음 | [High] |
| Server signature over canonical envelope | `client.go:180,418-470` | YES | Client rebuilds struct with **its own** `serverDID` and recomputed hashes, so a response signed by a different DID fails. `r.Ctx` is taken from the response (`:449`, comment says "prefer local ctxID" but uses `r.Ctx`); harmless because ackTag already bound ctx. | 결함 없음 | [High] |
| Response timestamp check | `serverSigEnvelope.Ts` signed (`server.go:316`) | NO | Client never checks `r.Ts` (`client.go:350-360` ignores errors, `:418-470` never reads it). Response replay is prevented by the fresh `ephC` in the transcript, so impact is nil. | 결함 없음 | [High] |
| TOFU key pinning | `client.go:53,431-436` | NO: dead code | `c.pins` is only read; **no code path writes it** (grep: single hit at `:431`). | [권장] | [High] |
| Lenient response parsing | `client.go:350-360` `v, _ := get(...)` | PARTIAL | Missing fields become "" and are caught later by `:419` (v/task) or the signature; `kid==""` would still bind an empty kid — but ackTag covers kid, so server must have sent it. | [권장] | [Mid] |
| Nonce generation | `client.go:104` `uuid.NewString()` (v4, 122-bit random) | YES | Fine. | 결함 없음 | [High] |
| Zeroisation | `client.go:99-190` | YES | Consistent. `ephCpriv` (X25519 private) is not zeroed — Go `ecdh.PrivateKey` has no zeroise API. | [권장] | [High] |
| `sessMgr` nil deref before nil check | `client.go:501` vs `:510` | — | Programming defect (documented in analysis 03). | [권장] | [High] |

## 3. P3 — Encrypted session use (`pkg/agent/session`)

Entry points: `SecureSession.Encrypt/Decrypt` (`session.go:465,503`), `EncryptAndSign/DecryptAndVerify` (`:540,568`), `SignCovered/VerifyCovered` (`:643,650`). Non-test callers outside the package: `cmd/metrics-demo/main.go:138,145` (`Encrypt/Decrypt` only). **No production code encrypts application traffic with these sessions.**

| Control | Implemented where | On path? | Gap / defect | Sev | Conf |
|---|---|---|---|---|---|
| AEAD confidentiality/integrity | ChaCha20-Poly1305, `session.go:488-489,663-680` (directional), `:505` (legacy) | YES | Random 96-bit nonce per message; with `MaxMessages` default 1000 (`manager.go:50`) collision risk negligible. | 결함 없음 | [High] |
| Directional keys | `deriveDirectionalKeys` `:251-291`, HKDF salt = session id, info `"sage-directional-keys-v1"` | YES | Correct direction split. | 결함 없음 | [High] |
| Legacy `signingKey` / `aead` on HPKE sessions | `:255-265` zero-fills `[0:64]`, never derives; `NewSecureSessionFromExporterWithRole` never sets `aead` (`:120-141`) | **PARTIAL (broken)** | `SignCovered` = HMAC with 32 zero bytes; `EncryptAndSign` nil-deref **[tested §8.5]**. These are `session.Session` interface methods, so any consumer of the interface hits it. | [치명] | [High] |
| Message counter / ordering | none | NO | No sequence number, no AAD. Reordering and duplicate delivery are undetectable. **[tested §8.6]** | [중요] | [High] |
| Replay guard (`NonceCache`, `ReplayGuardSeenOnce`) | `nonce.go:46-62`, `manager.go:345-350` | NO: dead code | Zero callers of `ReplayGuardSeenOnce`; zero callers of `GetByKeyID` outside session (grep). Designed for "RFC-9421 `nonce` per keyid" (`manager.go:343`) — i.e. it was meant to be called from an RFC 9421 verifier with `keyid`, which never happened. | [중요] | [High] |
| Session expiry | `IsExpired` `:341-367` (MaxAge 1 h, idle 10 m, MaxMessages 1000; `manager.go:47-51`) | YES | Checked on every Encrypt/Decrypt. `Close()` writes `closed` without the lock (`:447`); `IsExpired` reads it under RLock → data race. | [권장] | [High] |
| Key rotation | none in `pkg/agent/session` (grep `rotat|rekey` → only `cmd/sage-crypto/rotate.go`) | NO: missing | CLAUDE.md claims "자동 키 로테이션"; there is no rekey. Session ends at MaxMessages and must be re-handshaken. | [권장] | [High] |
| AAD binding (`EncryptWithAAD*`) | `:616-658,720-756` | YES (concrete type only) | Not on the `Session` interface (`types.go:28-49`); no caller passes AAD. | [권장] | [High] |
| Key zeroisation | `Close` `:429-451`, `Reset` `:362-395` | YES | `Close` zeroes; pooled `Reset` also zeroes. Fine. | 결함 없음 | [High] |
| Constant-time MAC compare | `hmac.Equal` `:594,671` | YES | — | 결함 없음 | [High] |
| Session ID secrecy | `ComputeSessionIDFromSeed` `:211-220`: SHA-256(label‖secret)[:16] | YES | sid is derived from the secret; it is used as HKDF salt (`:235`) and never sent on the wire (kid is). Fine. | 결함 없음 | [High] |
| `SetDefaultConfig` race | `manager.go:382-384` no lock | — | — | [권장] | [High] |

## 4. P4 — RFC 9421 (`pkg/agent/core/rfc9421`) and `core.VerificationService`

Two verifiers with different semantics:

* **HTTP path**: `HTTPVerifier.SignRequest/VerifyRequest` (`verifier_http.go:52,111`). Consumers: `examples/mcp-integration/basic-demo/main.go:80`, `basic-tool/calculator_tool.go:99`, plus signing in `client/sage_client.go:104`, `simple-standalone/main.go:169`. **This is the path real MCP integrations use.**
* **Envelope path**: `Verifier.VerifySignature/VerifyWithMetadata` (`verifier.go:59,97`), wrapped by `core.VerificationService` (`verification_service.go:51,107,140`). Consumers: `core.Core` (`core.go:49`) only; `lib/export.go` references but does not call.

### HTTP path

| Control | Implemented where | On path? | Gap / defect | Sev | Conf |
|---|---|---|---|---|---|
| Signature over covered components + `@signature-params` | `canonicalizer.go:36-53` | YES | Covered set is **chosen by the signer** and the verifier cannot require anything: `HTTPVerificationOptions.RequiredComponents` (`verifier_http.go:268-269`) is declared and **never read** (grep: 2 hits, both the declaration). | [치명] | [High] |
| Content-Digest / body binding | `body_integrity.go:66-94`; only if `content-digest` is covered | PARTIAL | None of the four MCP examples cover `content-digest` (`basic-demo/main.go:210-216`, `client/sage_client.go:91-98`, `simple-standalone/main.go:157-164`). Body swap after signing verifies **[tested §8.3]**. | [치명] | [High] |
| Nonce uniqueness / replay | `params.Nonce` parsed (`parser.go:236-237`) but **never checked**; `HTTPVerifier` has no nonce store (`verifier_http.go:40-42`). `Verifier.VerifyHTTPRequest` (`verifier.go:234`) delegates without using its `nonceManager`. | NO | Same signed request accepted repeatedly **[tested §8.1]**. Examples don't even send a nonce. | [치명] | [High] |
| Timestamp window | `created` age ≤ MaxAge 5 m (`:161-167`), `expires` (`:169-171`) | PARTIAL | No lower bound: `created` = now+24 h accepted **[tested §8.2]**. Combined with no nonce → a captured request is replayable for 5 minutes, or indefinitely if the signer puts `created` in the future. `expires` is signer-controlled. | [중요] | [High] |
| Key-type/algorithm binding | `sagecrypto.ValidateAlgorithmForPublicKey` (`verifier_http.go:194`; `algorithm_registry.go`) | PARTIAL | Empty `alg` is allowed (`ValidateAlgorithmForPublicKey` returns nil). `GetKeyTypeFromPublicKey` maps every `*ecdsa.PublicKey` to secp256k1 → P-256 keys with `ecdsa-p256-sha256` are rejected; secp256k1 keys are verified with **SHA-256** (`:199-201,217`) not Keccak, so `alg="es256k"` here ≠ Ethereum `es256k` produced by `keys.secp256k1KeyPair.Sign` (`keys/secp256k1.go` Keccak). Only Go↔Go with `SignRequest` interoperates. | [중요] | [High] |
| DID → key resolution / active check | not in rfc9421; example does `didManager.ResolvePublicKey` (`calculator_tool.go:93`), which rejects inactive agents | YES (example) / N/A (library) | Library gives no helper that ties `keyid` to the resolved DID: `params.KeyID` is never compared to the `X-Agent-DID` header or the DID used for resolution. | [권장] | [High] |
| Signature selection with multiple signatures | `verifier_http.go:143-147` picks "first" by **Go map iteration** | YES | Non-deterministic which signature is verified when `SignatureName` is unset and >1 signature present → attacker can add a second, valid-by-own-key signature… only if the verifier resolves keys per-signature, which it does not (single `publicKey` arg). Still non-deterministic failure. | [권장] | [High] |
| Request size limit | none; `readBodyAndRestore` `body_integrity.go:164` does `io.ReadAll` | NO | Unbounded memory if `content-digest` covered. | [권장] | [High] |
| Error information leak | `body_integrity.go:90` echoes actual and expected digests; `verifier_http.go:165` echoes ages | YES | Low. | [권장] | [High] |
| RSA | `verifier_http.go:221-225` PKCS#1 v1.5; registry name `rsa-pss-sha256` (`keys/algorithms.go:90`) | YES | Name/implementation mismatch (analysis 01 §5). | [권장] | [High] |

### Envelope path and `core.VerificationService`

| Control | Implemented where | On path? | Gap / defect | Sev | Conf |
|---|---|---|---|---|---|
| Signature base | `verifier.go:143-172` `ConstructSignatureBase` | YES | `SignedFields` comes from the **message** (`X-Signed-Fields` header in `VerifyMessageFromHeaders`, `message_builder.go:150-156`); no required set. `SignedFields=[""]` → empty base. Only exploitable with a signature over the empty/short string by the real key. | [중요] | [High] |
| Nonce replay | `verifier.go:74-78,89-91` via `nonce.Manager` | YES | Check-then-mark is non-atomic (`nonce/manager.go:64-89`; two lock scopes) → two concurrent replays both pass. Global namespace (not per DID/keyid). Optional: only when `Nonce != ""`; `QuickVerify` never sets one. Goroutine leak per `NewVerifier` (`verifier.go:46`). Positive control **[tested §8.4]**. | [중요] | [High] |
| Timestamp window | `verifier.go:65-71` symmetric ±5 m | YES | Disabled when `MaxClockSkew==0` (`QuickVerify`, `verification_service.go:178-180`). `Timestamp` formatted with `time.RFC3339` (seconds) in the base (`:155`) while parsed from `X-Timestamp` — sub-second precision silently dropped; signer must format identically. | [권장] | [High] |
| Algorithm binding | `verifier.go:175-215` string switch + Go key type assertion | PARTIAL | ECDSA: SHA-256 over base, raw r‖s. **Incompatible with `keys.secp256k1KeyPair.Sign` (Keccak)** — `QuickVerify` for `did:sage:ethereum:*` selects `ECDSA-secp256k1` (`verification_service.go:160-161`) and will reject SDK-produced signatures **[tested §8.7]**. No low-S check. | [중요] | [High] |
| DID active check | `verification_service.go:62-69` (`RequireActiveAgent`) | YES | `opts == nil` → nil deref **[tested §8.8]**. | [권장] | [High] |
| Capability check | `verifier.go:124-137`; capabilities injected from resolver (`verification_service.go:81`) | YES | Compares on-chain caps to required; not attacker-influenced. OK. | 결함 없음 | [High] |
| Metadata check (endpoint/name) | `verifier.go:116-122,217-231` vs `X-Metadata-*` headers | PARTIAL | Compared values are **unsigned** unless the signer also lists `header.X-Metadata-Endpoint` in `SignedFields`. It is a consistency check, not an integrity control. | [권장] | [High] |
| Key revocation | none | NO | Only the agent-level `IsActive`. | [중요] | [High] |
| Constant-time compare | `ed25519.Verify`, `ecdsa.Verify` (library) | YES | `compareValues` (`:258-263`) string compare on JSON — non-secret. | 결함 없음 | [High] |

## 5. P5 — Transport servers

| Control | HTTP (`transport/http/server.go`) | WebSocket (`transport/websocket/server.go`) | Sev | Conf |
|---|---|---|---|---|
| Verification before dispatch | none: presence of `ID`, `DID`, `Payload` only (`:109-120`) | same (`:209-220`) | — (by design; handler verifies) | [High] |
| Request size limit | **none**: `io.ReadAll(r.Body)` (`:86`), no `http.MaxBytesReader` (grep: 0 hits) | **none**: no `conn.SetReadLimit` (grep: 0 hits); `ReadJSON` unbounded (`:197`) | [중요] | [High] |
| Timeouts | none in package; depends on the caller's `http.Server` (`ReadHeaderTimeout` etc. not set anywhere in repo — not verified beyond grep of this package) | read 60 s / write 30 s (`:80-81,191,285`) | [권장] | [Mid] |
| Origin / host allowlist | N/A | **off by default** (`:83`); when on, **empty `Origin` is allowed** (`:137-141`); `allowedOrigins` map mutated without lock (`:150-153,157`) vs read in upgrader → race | [중요] | [High] |
| Header override of identity | `X-SAGE-DID` / `X-SAGE-Context-ID` / `X-SAGE-Task-ID` **override the JSON body** (`:153-164`) | none | HPKE server re-binds DID via payload (`server.go:224`), so override changes nothing there; for any handler that trusts `msg.DID` without a payload binding this is a spoof vector. | [권장] | [High] |
| Error information leak | handler `err.Error()` verbatim, HTTP 200 (`:201-209`) | verbatim (`:272-280`) | [권장] | [High] |
| TLS | none in server (caller's listener); client uses `http.Client{Timeout:30s}` default transport, TLS verify on (`client.go:59-60`) | `Dialer` default (verify on) | 결함 없음 | [High] |
| Non-test consumers | `NewHTTPServer` / `NewWSServer*`: **none** outside tests/examples (grep) | same | — | [High] |

## 6. P6 — DID resolution (`pkg/agent/did`, `did/ethereum`)

Path used by P1/P4: `did.Manager.ResolvePublicKey` (`manager.go:190`) → `MultiChainResolver` (`resolver.go:101-112`) → `EthereumClient.Resolve` (`ethereum/client.go:203-425`).

| Control | Implemented where | On path? | Gap / defect | Sev | Conf |
|---|---|---|---|---|---|
| Chain-of-trust DID → on-chain record | `client.go:342-365` `getAgentByDID(did)`; existence check `:368-370` | YES | The DID string is a lookup key; Go never checks the identifier part of `did:sage:ethereum:0x…` against `Owner` or any key. Whether the contract enforces it was **not verified** (Solidity out of scope). | [권장] | [Mid] |
| Active flag | `:421` `IsActive: on.Active`; enforced in `ResolvePublicKey`/`ResolveKEMKey` (`ethereum/resolver.go:42-44,56-58`; `did/resolver.go:107,121`) | YES | `Resolve` itself returns inactive agents; `VerificationService.VerifyAgentMessage` re-checks (`:63`). OK. | 결함 없음 | [High] |
| Key selection / revocation | `:380-410`: loops `keyHashes`, calls `getKey`, takes the **first `KeyType==0` (ECDSA)**; ignores `k.Verified` (`:223` field read at `:246,256`, never tested) | PARTIAL | (a) unverified keys are trusted; (b) Ed25519 (`KeyType==1`) keys are never returned → Ed25519-only agents on Ethereum resolve to `PublicKey==nil`; (c) no revoked-key concept in Go — if the contract keeps revoked hashes in `keyHashes`, a revoked ECDSA key is still used. | [중요] | [High] |
| Key type binding to DID chain | `did.UnmarshalPublicKey(k.KeyData,"secp256k1")` (`:403`) | YES | Fine for ECDSA. | 결함 없음 | [High] |
| KEM key | `:422` raw 32 B from chain, no PoP | YES | `UpdateKEMKey` takes a signature (`agentcard_client.go:517-531`) — verification is contract-side, not checked here. | [권장] | [Mid] |
| Caching | **none** in `EthereumClient.Resolve` (1 + N RPC calls per verification) | NO | The cached `ethereum.Resolver` (`ethereum/resolver.go:251-355`) returns a **hard-coded mock document** (`"mock-public-key"`, `:340-347`) and is used by `cmd/sage-did/debug.go:77`. It must never be wired into a verifier. DoS on verifier: every HTTP request in `basic-tool` triggers two full resolutions (`calculator_tool.go:93,105`). | [중요] | [High] |
| Chain extraction | `resolver.go:181-200` prefix `"eth"`/`"sol"`; `manager.go:280-297` | YES | Consistent. | 결함 없음 | [High] |

## 7. P7 — A2A card proof and key proof-of-possession

| Control | Implemented where | On path? | Gap / defect | Sev | Conf |
|---|---|---|---|---|---|
| A2A card signature | `a2a_proof.go:63-139` sign SHA-256(`json.Marshal(baseCard)`); verify `:154-261` | YES (`cmd/sage-did/card.go:262` via `ValidateA2ACardWithProof`) | Verification key is looked up **inside the card** (`:162-168`); no resolver, no on-chain cross-check. `ValidateA2ACardWithDID` (`a2a.go:271`) exists but `card.go:262` does not call it. A forger generates a key, lists it in `publicKey`, signs — passes. | [중요] | [High] |
| ECDSA hex key decoding | `:215-216` decodes `PublicKeyHex` with **base58** | YES | Bug: hex-encoded ECDSA keys never verify. | [권장] | [High] |
| Ed25519 prehash | `:104,204` signs/verifies `hash[:]` (Ed25519 over SHA-256) | YES | Non-standard for `Ed25519Signature2020` (which signs the canonicalised document directly); interop with other DID libraries will fail. | [권장] | [High] |
| Key PoP | `key_proof.go:46-82` sign SHA-256(`"SAGE-PoP:<did>:<hex key>"`); verify `:97-166`; used `cmd/sage-did/key.go:610` | YES | Domain-separated, deterministic. X25519 excluded (`:74-77`) — KEM keys have no PoP. Whether the contract verifies PoP on `addKey` was not checked. | [권장] | [High] |
| Signature domain separation across protocols | HPKE init signs raw JSON payload (`hpke/client.go:263`); rfc9421 envelope signs `"body: …"` lines; PoP signs `"SAGE-PoP:…"`; A2A signs SHA-256 of card JSON | — | All four use the **same DID key**. No shared prefix collision found by reading (raw JSON starts with `{`; others with `agent_did:`/`body:`/`SAGE-PoP:`), but there is no explicit domain tag on the HPKE and rfc9421 paths. | [권장] | [Mid] |

## 8. Throwaway test results (module deleted after run)

Scratch module `scratchaudit` with `replace github.com/sage-x-project/sage => <repo>`; `go test -v -count=1`, all 8 tests PASS (i.e. each claim held):

1. `TestHTTPReplayAccepted` — request signed with `nonce="nonce-1"` accepted 3× by `HTTPVerifier.VerifyRequest` and 3× by `Verifier.VerifyHTTPRequest`.
2. `TestHTTPCreatedInFutureAccepted` — `created = now+24h` accepted.
3. `TestHTTPBodyNotBoundWithExampleComponents` — signed with the basic-demo component set, body replaced with `{"op":"transfer","amount":1000000}`, `VerifyRequest` returned nil.
4. `TestEnvelopeReplayRejected` (control) — second `Verifier.VerifySignature` with same nonce returned `nonce replay attack detected`.
5. `TestHPKESessionZeroSigningKeyAndNilAEAD` — `SignCovered("hello") == HMAC-SHA256(key=32×0x00,"hello")`; `EncryptAndSign` panicked with nil pointer dereference.
6. `TestSessionCiphertextReplayAccepted` — ciphertext from initiator `Encrypt` decrypted 3× by responder.
7. `TestSecp256k1HashMismatchAcrossPaths` — `keys.secp256k1KeyPair.Sign(base)` → rfc9421 envelope verify `ECDSA signature verification failed`; `hpke.CompositeVerifier.Verify` on the same bytes → nil.
8. `TestVerifyAgentMessageNilOptsPanics` — nil pointer dereference at `opts.RequireActiveAgent`.

Not tested (by reading only): P1 `RespDID` not checked; WS origin/size; DID key selection (needs chain).

---

## 9. Determinism of signing and verification

### Algorithms

| Algorithm | Signing deterministic? | Where | Notes |
|---|---|---|---|
| Ed25519 | Yes (inherent, RFC 8032) | `keys/ed25519.go` `Sign`; `rfc9421/verifier_http.go:65`; `a2a_proof.go:104`; `key_proof.go:60` | A2A/PoP sign a SHA-256 prehash — deterministic but non-standard. |
| secp256k1 via go-ethereum | Yes (`ethcrypto.Sign` uses RFC 6979 deterministic nonce) | `keys/secp256k1.go` `Sign`; `a2a_proof.go:112`; `key_proof.go:68` | Output 65 B (r‖s‖v), low-S normalised by libsecp256k1. Hash = Keccak-256 of the input. |
| P-256 via `crypto/ecdsa` | **No** — `ecdsa.Sign(rand.Reader, …)` (Go ≥1.20 hedged: deterministic nonce mixed with randomness) | `keys/p256.go` `Sign`; `rfc9421/verifier_http.go:73` (also used for secp256k1 `*ecdsa.PrivateKey` on the HTTP path) | Different signature bytes per call; verification unaffected. No low-S normalisation → malleable; without a nonce/replay store this means the same message can appear with two distinct valid signatures. |
| RSA PKCS#1 v1.5 | Yes (inherent) | `keys/rs256.go` `Sign`; `verifier_http.go:93` (via `crypto.Signer`) | Registry advertises `rsa-pss-sha256` (`keys/algorithms.go:90`); PSS would be randomised. |

### Canonicalisation steps and cross-implementation determinism

| Step | Location | Same logical message → same base bytes? | Risk |
|---|---|---|---|
| RFC 9421 component lines | `canonicalizer.go:36-53,152-167` | Go↔Go yes. Cross-language: header values joined with `", "` and outer-trimmed only (`:160-163`); RFC 9421 §2.1 also requires collapsing obs-fold and does not lowercase values — matches for simple headers. Component identifiers are emitted **exactly as the signer wrote them** (`buildSignatureParams :196-198`): `"@method"` vs `@method` produce different `@signature-params` lines; the parser (`parser.go:183-189`) rejects unquoted, so Go signers must quote. `@query-param` value is URL-decoded via `req.URL.Query()` (`:178-185`), RFC requires the encoded form → interop break for encoded params. `@request-target` (`:116-125`) is a draft-cavage component, not RFC 9421. | [중요] cross-impl |
| `@signature-params` parameter order | `canonicalizer.go:200-215` fixed order keyid, alg, created, expires, nonce; the `Signature-Input` header is emitted in the same order (`verifier_http.go:239-255`) but the **verifier rebuilds from parsed params**, so a foreign signer that orders `created;keyid` gets a different base → fails. | Go↔Go yes; foreign no | [중요] |
| Multiple signatures | `verifier_http.go:143-147` map iteration | **Non-deterministic** choice when `SignatureName==""` and ≥2 signatures | [권장] |
| `time.Now()` inside verification | `verifier.go:66`, `verifier_http.go:161`, `server.go:227` | Only for window checks, not in the base | 결함 없음 |
| Envelope base (`ConstructSignatureBase`) | `verifier.go:143-172` | Lines joined with `\n`; timestamp `RFC3339` (seconds, zone as parsed — `Z` vs `+00:00` differ!); body embedded raw; header lookup is **case-sensitive map key** (`msg.Headers[headerName]`, `:164`) while `ParseMessageFromHeaders` copies keys verbatim — `X-Agent-DID` vs `x-agent-did` differ. Not an RFC 9421 base at all. | [중요] cross-impl |
| HPKE init payload | `hpke/client.go:247-258` `json.Marshal(map[string]any)` — Go sorts map keys, HTML-escapes `<>&` | Signed bytes travel with the message and the server verifies the **received bytes** (`server.go:213-217`), so canonical form is irrelevant for verification. Deterministic given identical inputs; `ts` uses `RFC3339Nano` which drops trailing zeros (`2025-…T00:00:00Z` vs `.000Z`), fine because bytes are shipped. | 결함 없음 |
| HPKE signed response | signed: `serverSigEnvelope` struct (`server.go:73-86,335`); shipped: `map[string]any` (`:344-360`); client **rebuilds the struct** (`client.go:446-463`) | Deterministic in Go (struct field order, no maps, all strings). A non-Go client must reproduce Go's `encoding/json` struct encoding exactly (no spaces, field order as listed, `RawURLEncoding` no padding). Shipping a map but signing a struct means the wire form is **not** the signed form — a foreign implementation cannot just verify "what it received". | [권장] cross-impl |
| A2A card JSON | `a2a_proof.go:86,182` `json.Marshal(A2AAgentCard)` | Struct order fixed; `Capabilities []string` order as stored; `time.Time` → `RFC3339Nano` (round-trips only if the incoming JSON used the same precision and zone formatting); `omitempty` on `capabilities`/`publicKeyHex` means a card that arrives with `"capabilities":[]` re-marshals differently. Go HTML-escaping of `<>&` in `description`/endpoint URLs (`?a=1&b=2` → `&`) breaks cross-language verification. No JCS (RFC 8785). | [중요] cross-impl |
| PoP challenge | `key_proof.go:218-221` `fmt.Sprintf("SAGE-PoP:%s:%x")` | Deterministic, lowercase hex. | 결함 없음 |
| HPKE `info`/`exportCtx` | `common.go:40-60` fixed string concatenation | Deterministic. | 결함 없음 |
| Session ID / keys | `session.go:186-220,235-291` SHA-256/HKDF over fixed-order inputs; `canonicalOrder` for ephemerals | Deterministic. | 결함 없음 |

Concrete non-determinism / cross-implementation risks (file:line): `verifier_http.go:143-147` (map iteration), `canonicalizer.go:178-185` (decoded query param), `canonicalizer.go:196-198` (verbatim component identifiers), `verifier.go:155` (`RFC3339` zone formatting), `verifier.go:164` (case-sensitive header map), `a2a_proof.go:86,182` (Go JSON escaping / omitempty / time precision), `server.go:344-360` vs `:73-86` (map shipped, struct signed), `verifier_http.go:73` (randomised ECDSA, no low-S).

---

## 10. Threat coverage for agent/MCP messaging

| Threat | Addressed today (on-path) | Implemented but unwired | Missing |
|---|---|---|---|
| Message tampering in transit | HPKE init/response: full payload signed + ackTag (P1/P2, test-only consumers). rfc9421 HTTP: only covered components; MCP examples do **not** cover the body. Session AEAD (no production caller). | `BodyIntegrityValidator` (works only if signer covers `content-digest`); `RequiredComponents` field. | Verifier-side mandatory component policy; AAD/sequence in session frames. |
| Replay | HPKE init: nonce store + ±2 m window (test-only consumer). Envelope path: `nonce.Manager` when a nonce is present. | `session.NonceCache`/`ReplayGuardSeenOnce`; `storage.NonceStore.CheckAndStore` (atomic, persistent; zero importers); `core/message/validator` (zero importers); hpke `nonceStore` is O(n). | HTTP-path replay check (the path MCP uses); forward skew bound; session-layer counter; 4-phase handshake nonce/timestamp (`handshake/server.go:142-340` never reads them — grep confirms). |
| Impersonation of an agent DID | DID → on-chain key via `EthereumClient.Resolve`; active flag; HPKE `InitDID==msg.DID`. | `ValidateA2ACardWithDID` (`a2a.go:271`, not called by CLI). | Binding of rfc9421 `keyid` to the resolved DID; A2A card verification against chain; `Verified` flag enforcement on keys; Ed25519 keys on Ethereum never resolvable. |
| Key compromise / rotation | Agent deactivation (`IsActive`) stops all verification. `sage-crypto rotate` rotates local storage keys. | `Manager.RevokeKey` (`did/manager.go:321`, CLI only). | Per-key revocation check at resolve time; session rekey; KEM key PoP; TOFU pins never written (`hpke/client.go:53`). |
| Downgrade to unsigned | HPKE: signature mandatory (`common.go:129-131`). | — | rfc9421: signer chooses covered set and may omit `alg` (`ValidateAlgorithmForPublicKey` accepts empty); no minimum policy; HTTP transports dispatch without any signature requirement (`http/server.go:109-123`). |
| Malicious tool result injection into an agent | Nothing: rfc9421 examples verify **requests** only; responses are never signed (`verifier_http.go` has no response support; `@status` rejected `canonicalizer.go:140-142`). | `session.EncryptAndSign` (broken on HPKE sessions). | Response signing / channel binding for tool results; MCP example returns plain JSON. |
| Confused deputy via MCP | Capability check exists (`verifier.go:124-137`; example `calculator_tool.go:105`). | `did.MetadataVerifier.ValidateAgentForOperation` (`verification.go:158`). | Audience/target binding: no `@authority`/`@target-uri` in example covered set; HPKE `RespDID` unchecked by server; no per-tool scope in capabilities model. |
| Metadata / endpoint spoofing | `VerifyMetadata` compares on-chain vs supplied (`ethereum/resolver.go:64-123`). | `validateEndpoint` (`verification.go:243`). | Endpoint is not part of any signature; `X-Metadata-*` unsigned; header override of DID/ctx in HTTP transport (`http/server.go:153-164`). |
| Denial of service on verifier | HPKE optional cookie (unwired, and placed after the expensive work). | `CookieVerifier`. | Body size limits (HTTP/WS), `SetReadLimit`, `MaxBytesReader`; DID resolution cache (every request = ≥2 RPC round-trips); hpke `nonceStore` O(n) sweep and unbounded; `nonce.Manager` goroutine per `NewVerifier`. |

---

## 11. Recommended wiring fixes (prioritised)

### (a) Connect existing code

1. **Replay check on the HTTP path** — in `HTTPVerifier.VerifyRequest` (`verifier_http.go:111`) after parsing, when `params.Nonce != ""`, call an injected guard. Reuse `session.NonceCache.Seen(keyid, nonce)` (`session/nonce.go:46`) keyed by `params.KeyID`, or `storage.NonceStore.CheckAndStore` (`storage/interface.go:56`) for multi-instance. Make `Verifier.VerifyHTTPRequest` (`verifier.go:234`) pass its existing `nonceManager`. Also reject `params.Nonce == ""` when `opts.RequireNonce` (new bool).
2. **Enforce `RequiredComponents`** — read `opts.RequiredComponents` (`verifier_http.go:269`) in `VerifyRequest` and fail if any is missing from `params.CoveredComponents` (use `IsComponentCovered`, `body_integrity.go:108`). Set `DefaultHTTPVerificationOptions` to require `@method`, `@authority`, `@path`, `content-digest` for requests with a body; update the four examples to cover `content-digest` and `@authority` and to set `Nonce`.
3. **Fix HPKE session key derivation** — in `deriveDirectionalKeys` (`session.go:251`) also HKDF-fill `[0:64]` and call `chacha20poly1305.New(encryptKey)` in `NewSecureSessionFromExporterWithRole` (`:120-141`), or make `SignCovered/VerifyCovered/EncryptAndSign/DecryptAndVerify` return an error when `s.aead == nil`. Add the missing test (`FromExporterWithRole` + `SignCovered`).
4. **A2A card: verify against chain** — `cmd/sage-did/card.go:262` should call `ValidateA2ACardWithDID` (`a2a.go:271`) after `ValidateA2ACardWithProof`, or `VerifyA2ACardProof` should take a `Resolver` and require the verification key to match a `Verified` on-chain key.
5. **Use `Verified` when selecting keys** — `ethereum/client.go:402` add `&& k.Verified`; also return Ed25519 keys (`KeyType==1`) so `ResolvePublicKey` works for Ed25519 agents, choosing by the DID chain or by a requested key type.
6. **Wire `core/message/validator`** (`validator.go:55`) into `handshake.Server.HandleMessage` (`handshake/server.go:142`) for Invitation/Request/Complete, or delete `handshake` (analysis 03 §7 #8). The message types already implement `ControlHeader`.
7. **Move the HPKE cookie check before DID resolution** — `server.go:137-142` above `:125`; cheap check first is the whole point.

### (b) Small additions

1. Forward skew bound in `VerifyRequest`: reject `params.Created > now + opts.MaxSkew` (`verifier_http.go:161-167`).
2. `hpke.Server`: check `pl.RespDID == s.DID` in `validateInitEnvelope` (`server.go:222`).
3. Replace hpke `nonceStore` (`common.go:154-178`) with `session.NonceCache` (has GC loop, no per-call sweep) keyed by `ctxID`.
4. Size limits: `http.MaxBytesReader(w, r.Body, N)` in `http/server.go:86`; `conn.SetReadLimit(N)` in `websocket/server.go:189`; `io.LimitReader` in `body_integrity.go:164`.
5. WebSocket origin: default `checkOrigin=true`; treat empty `Origin` as reject unless explicitly allowed (`:137-141`); guard `allowedOrigins` with `connMu` or a dedicated mutex.
6. Drop the `X-SAGE-*` header override of DID/context (`http/server.go:153-164`) or require equality with the body.
7. `nonce.Manager`: make check+mark atomic (`nonce/manager.go:64-89` → single `CheckAndMark`), add `Close()`, key by DID/keyid.
8. `VerifyAgentMessage`: `if opts == nil { opts = rfc9421.DefaultVerificationOptions() }` (`verification_service.go:55`).
9. `ValidateAlgorithmForPublicKey`: reject empty `alg` when a policy requires it; fix `GetKeyTypeFromPublicKey` to detect P-256 via `key.Curve` (`algorithm_registry.go`).
10. Fix `a2a_proof.go:216` (hex decode, not base58); sign the JCS-canonical card without prehash for Ed25519 if interop with `Ed25519Signature2020` is intended.
11. `SecureSession.Close` take the write lock (`session.go:446-447`); `Manager.SetDefaultConfig` take the lock (`manager.go:382`).
12. Pick the signature deterministically when several are present (`verifier_http.go:143-147`): require `opts.SignatureName` or sort names.
13. Generic error strings for auth failures at the transport boundary (`http/server.go:206`, `websocket/server.go:277`); log details server-side.

### (c) Missing designs

1. **Single signing convention per key type.** Decide whether secp256k1 signs Keccak-256 (Ethereum) or SHA-256 (RFC 9421 `es256k` is Keccak per registry? — the registry name says `es256k`, the HTTP verifier does SHA-256). Today `keys.secp256k1KeyPair.Sign`, `hpke.ECDSAVerifier`, `did/*_proof.go` use Keccak; `rfc9421` (both paths) use SHA-256. One `SignatureVerifier` (analysis 03 §7 #4) used by hpke, handshake and rfc9421 would remove the split.
2. **Session-layer replay/ordering.** Add a per-direction 64-bit counter used as the AEAD nonce (or as AAD with a random nonce), sliding-window receive check, and a `Rekey()` at `MaxMessages` to make "automatic key rotation" true.
3. **Response/tool-result authentication.** rfc9421 response signing (`@status`, `content-digest` of the response), or wrap tool results in the session AEAD with the request id as AAD.
4. **Audience binding.** Require `@authority` (HTTP) / `RespDID` (HPKE) so a message for agent A cannot be forwarded to agent B; add a `did` component that the verifier compares with the resolved DID and `keyid`.
5. **Key lifecycle at resolution.** Define the on-chain revocation semantics (`revoked` flag or removal from `keyHashes`) and enforce it in `Resolve`; add PoP for KEM keys; give `hpke.Client` a way to populate `pins`.
6. **Canonical JSON (RFC 8785) for A2A cards and the HPKE response** so non-Go implementations can verify what they receive rather than re-encoding Go structs.
7. **Verifier DoS budget**: per-DID resolution cache with TTL fed by chain events (replace the mock `ethereum.Resolver`), and rate limiting before signature verification.

---

<Fact-based Answer>
**Fact**
- `hpke.NewServer`/`NewClient`, `NewHTTPServer`, `NewWSServer*`, `ReplayGuardSeenOnce`, `GetByKeyID`, `EncryptAndSign/SignCovered/DecryptAndVerify/VerifyCovered`, `storage.NonceStore`, `core/message/validator`, `hpke.ServerOpts.Binder/Cookies`, `HTTPVerificationOptions.RequiredComponents` have zero non-test callers/readers in the module (grep, this tree).
- `HTTPVerifier.VerifyRequest` accepted the same nonce-bearing request 3×, accepted `created=now+24h`, and accepted a swapped body under the example component set (tests §8.1-8.3).
- `NewSecureSessionFromExporterWithRole` sessions have `signingKey` = 32 zero bytes and `aead == nil` (test §8.5); identical ciphertext decrypts repeatedly (test §8.6).
- `keys.secp256k1KeyPair.Sign` output fails `rfc9421.Verifier.VerifySignature` and passes `hpke.CompositeVerifier.Verify` (test §8.7).
- `VerifyAgentMessage(ctx, msg, nil)` panics (test §8.8).
- `handshake/server.go` contains no read of `Nonce` or `Timestamp` (grep).
- `ethereum.Resolver.Resolve` returns `PublicKey: "mock-public-key"` (`ethereum/resolver.go:343`) and is used by `cmd/sage-did/debug.go:77`.
- MCP examples cover `@method`, `@path`, `content-type`, `date`, `x-agent-did` (+`content-length` in `client/sage_client.go`); none cover `content-digest`; none set `Nonce`.

**Opinion**
- [High] The rfc9421 HTTP path is the only production-relevant verification path today, and it is the least protected; fixes (a)1-2 are the highest-value changes.
- [High] `hpke`/`session` controls are sound in design but unreachable from any shipped consumer; their defects (zero key, no counter) will surface as soon as someone wires them.
- [Mid] The contract may enforce DID↔owner and key revocation server-side; the Go layer does not, so an off-chain caller of `Resolve` cannot rely on it without checking Solidity.
- [Mid] Cross-language verification of A2A cards and the HPKE response will fail on Go-specific JSON encoding unless a canonical form is adopted.
- [Low] The `X-SAGE-DID` header override is exploitable only by handlers that trust `msg.DID` without a payload binding; none exist in-tree today.
</Fact-based Answer>
