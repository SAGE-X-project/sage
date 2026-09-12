# 05. Progress assessment: where SAGE stands against its purpose and protocol

Status: review synthesis, 2026-09-12. Read-only; no code changed. Tree: `sage` `878932d`, `sage-spec` `a34dd49`, `rs-sage-core` `206bbbb`, `sage-gateway` `4e72668`, `sage-inspector` `05b890d`, `sage-contracts` `d9f313b` (all 2026-09-12).

Sources: the review set in this directory (`01` documentation graph, `02` code graph, `03` history, `04` purpose, `07` literature, `08` implementation evaluation, `09` protocol flows, `10` inspector matrix), `docs/refactoring/BACKLOG.md`, `STRATEGY.md` §2-§6, `v2/REPO_PLAN.md`, `v2/VERSION_POLICY.md`. Every code fact that carries a headline judgement was re-checked with `grep`/`sed` on 2026-09-12; those are marked "verified" with `file:line`. Confidence: [High] read in code or document; [Mid] one inference; [Low] not checked. Severity: [critical] / [major] / [minor] as in `08`.

Yardstick. The purpose (`04` §1) is: agent-to-agent and agent-to-MCP calls that cannot be forged, replayed or altered, with chain-anchored identity and E2E-encrypted sessions, adoptable by unmodified agent clients, deterministic across implementations (`STRATEGY.md:19`). The protocol (`09` §1) is eight spec layers: crypto (01), JCS (02), RFC 9421 (03), HPKE handshake (04), session (05), did:sage (06), A2A card (07), transport envelope (08).

---

## 1. Capability matrix

Columns: **Spec** = normative text in `sage-spec`; **Go** = implemented in `sage`; **Rust** = implemented in `rs-sage-core`; **Live** = reachable from a shipped binary (`cmd/*`, `sage-gateway`, `sage-inspector`) on a real request path, not only tests or examples; **Vec** = covered by `sage-spec/vectors`; **Insp** = checkable by `sage-inspector`; **Docs** = a current (non-MIXED/STALE) document exists in `sage` besides the spec. Values: Y / P (partial) / N / n/a (does not apply by design; excluded from scoring).

| # | Layer / component | Spec | Go | Rust | Live | Vec | Insp | Docs | Evidence (one pointer each) |
|---|---|---|---|---|---|---|---|---|---|
| 1 | Crypto primitives (Ed25519, secp256k1-Keccak, P-256, X25519) | Y | Y | Y | Y | Y | P | P | `08` §1 (mainstream libs; RSA id mismatch [major]); `sage-crypto verify`, gateway signing; `crypto.json` 4 (P-256 verify-only); inspector infers key type by length (`10` a-n27); `docs/crypto/*` last 2025-10-11 (`03` §3) |
| 2 | JCS (RFC 8785) | Y | Y | Y | P | Y | P | P | `08` §5 (Go accepts what Rust rejects [minor]); live only via `sage-did card` proof, HPKE envelope tests only; `jcs.json` 4; no `jcs` inspector command (`10` §3); no guide in `sage` (`01` §4 "no guide") |
| 3 | RFC 9421 request signing/verification | Y | Y | Y | Y | Y | P | P | `09` §2a; gateway `serve`/`client` and five MCP examples (`02` §2e); `rfc9421.json` 3 positive requests; inspector `ExpectedDID` tautological (`10` §1 item 2); `rfc9421-en.md` MIXED (`01` §6 row 12); `@request-target` deviation verified `canonicalizer.go:232-241` [major] |
| 4 | RFC 9421 response signing (`;req` binding) | Y | Y | P | Y | Y | P | N | `09` §2a steps 7-8; Rust accepts any `;req` (`09` §4 a11); gateway signs/verifies; `response-ed25519` vector; inspector own check weak (`10` r-n1/r-n2); `rfc9421-en.md:307,443` still "pending" (`01` §6 row 12) |
| 5 | Replay protection (nonce guard, HPKE ctx guard, session window) | Y | Y | Y | P | P | N | N | one `session.ReplayGuard` verified `verifier_http.go:46-64`; only the HTTP guard is on a live path (gateway); all guards in-memory, unbounded (`09` §3); only negative vector is the session replay (`10` §2d d-n1); inspector `DisableReplayCheck` (`10` §1 item 3); ARCHITECTURE/handshake docs name the wrong mechanism (`01` §6 rows 6, 10) |
| 6 | HPKE 1-RTT handshake | Y | Y | Y | N | P | N | P | no non-test caller of `hpke.NewServer/NewClient` in `pkg cmd examples internal` (verified grep; `02` §2e); gateway excludes HPKE (`sage-gateway/README.md:57-58` verified); 6 derivation vectors, no init/ack envelope vector (`10` §3); cookie after signature verify (`server.go:147` vs `:159` verified, O-6 [major]); `hpke-based-handshake-*.md` MIXED with six wrong mechanism claims (`01` §6 rows 5, 7-11) |
| 7 | Session records (ChaCha20-Poly1305, seq, window, rekey) | Y | Y | Y | N | Y | N | P | record API called outside `pkg/agent/session` only by `pkg/vectors/session.go` and `examples/metrics-demo` (verified grep; `08` §4); directional entry points skip `IsExpired` (verified: called only at `session.go:646,663,682,704`) [major]; `session.json` 3, no stale vector; `session/README.md` misuses label as salt (`01` §6 row 4) |
| 8 | did:sage resolution and key policy | P | Y | P | Y | P | N | P | spec silent on key selection, `#key-n`, cache; no DID Document projection (O-8) (`09` §2b, `08` 6.3); Rust parses only, no chain resolver (`02` §3); `sage-did resolve` and gateway `resolve` cache (`09` §1); `did/parse`, `chain-aliases` vectors only; inspector has no resolver (`10` a-n31); `did-{en,ko}.md` MIXED (`01` §3) |
| 9 | Key proof of possession | Y | Y | Y | P | Y | N | N | verified only by `cmd/sage-did/key.go:610` and vectors, never in `key_policy.go` (verified grep; `08` 6.1-6.2 [major]); on-chain Ed25519 "verified" after a length check (`AgentCardRegistry.sol:439-442` verified); PoP not bound to chain id (O-1); `pop-*` vectors; no `pop` command; E-06 open |
| 10 | A2A agent card and proof | Y | Y | Y | Y | P | P | P | `sage-did card validate --with-proof --verify-did` (BACKLOG B-07); Go misses two spec MUSTs (`09` §4 b7, `10` b-n9/b-n10); one verify-only Ed25519 vector; inspector checks self-attested proof only (`10` §1); `examples/a2a-integration/*` STALE/MIXED (`01` §3) |
| 11 | On-chain registration (commit-reveal-activate) | n/a | Y | n/a | P | n/a | N | P | out of spec scope (`00-overview.md` §1); `AgentCardClient` implements all three phases (`09` §2e); `sage-did commit` sends `Keys: [][]byte{}` and DID `TBD` (verified `cmd/sage-did/commit.go:120-128`) so the CLI path cannot complete against the pinned contract (`08` 6.5); `docs/cli/*` lack the commands (`03` §3) |
| 12 | Transport / gateway | P | P | n/a | P | N | N | P | spec 08 has no vector (`01` §5); Go HTTP/WS transports sign nothing, verify nothing, configure no TLS (verified grep of `pkg/agent/transport`, `pkg cmd internal` for `tls.Config`; `08` 7.1 [major]); gateway = RFC 9421 proxies only, no HPKE, no stdio MCP, no TLS (`sage-gateway/README.md:57-58`, `main.go` grep); `docs/API.md` STALE, gateway README CURRENT |
| 13 | Inspector | n/a | Y | n/a | P | P | P | Y | `request`/`response`/`card`/`vectors` commands exist (`sage-inspector/pkg/inspect/{http,card,vectors}.go` verified); a pass is not evidence for replay, expected DID or `;req` set (`10` §1 items 2-4); vector run compares Go with Go (`10` §1 item 5); no HPKE/session/registry input; README CURRENT (`01` §3) |
| 14 | Rust core (`rs-sage-core`) | Y | n/a | Y | P | Y | N | P | 26 vectors pass in its CI (`ci.yml:30-35` verified); C header + WASM artefacts, 0.3.0; no consumer, no live Go exchange (F-03b open); FFI exposes no HPKE, no `catch_unwind` (verified grep; `08` 7.2 [major]); README MIXED (AES-GCM, 207 tests, `01` §6 row 15) |
| 15 | Contracts (`sage-contracts`) | n/a | Y | n/a | P | n/a | n/a | Y | bindings pinned to `2ff992e` with drift check (`.contracts-version` verified; BACKLOG C-02); Sepolia `ERC8004ValidationRegistry` still runs pre-fix open-admin code (`REPO_PLAN.md` §5, C-01 redeploy open); first tag pending (VERSION_POLICY §5); README CURRENT |
| 16 | Language SDKs (`sdk/*`) | n/a | n/a | N | N | N | n/a | P | four reimplementations, none bound to the Rust core, TS secp256k1 over SHA-256 (`08` 7.4 [major]); call routes no server serves (`STRATEGY.md` §1); experimental banners only (D-08); F-04 deferred until F-03b |
| 17 | Documentation | Y | P | P | Y | n/a | n/a | N | spec: 9 chapters, all CURRENT (`01` §3); `sage`: 26 CURRENT / 46 MIXED / 33 STALE of 105 (`01` §2); `docs/INDEX.md` generated and CI-checked (E-03); 18 open contradictions (`01` §6); every implementation-side protocol document MIXED or STALE (`01` §5) |
| 18 | CI / supply chain | n/a | Y | P | Y | P | Y | P | SHA-pinned actions, blocking scanners, govulncheck, GoReleaser + cosign + SLSA + SBOM, depguard + codegraph gate, vectors job (`03` §4 CI; CHANGELOG Unreleased); Rust CI red on main, Miri cannot fail (BACKLOG F-08 note; `08` 8.6); spec checkout `ref: main` not a tag (`test.yml:182` verified); no fuzz job, `-tags=integration` matches nothing (verified grep; `08` 8.3, 8.6); `docs/CI-CD.md` MIXED |

---

## 2. Progress figures

Rubric (stated so it can be recomputed): each of the seven columns weighs 1; Y = 1, P = 0.5, N = 0; n/a cells are dropped from the denominator of that row; a row score is its sum divided by its applicable columns; a layer score is the unweighted mean of its rows; the overall score is the unweighted mean of the 18 rows. Two caveats: equal weighting treats "documented" and "on a live path" alike, and n/a exclusions favour rows whose scope is narrow by design (registration, contracts). A second figure, the **Live column alone**, is reported because the purpose depends on it.

| Layer | Rows | Row scores | Layer | Live only |
|---|---|---|---|---|
| Primitives | 1, 2 | 86 %, 79 % | **82 %** | 75 % |
| Signing (RFC 9421 + replay) | 3, 4, 5 | 86 %, 71 %, 57 % | **71 %** | 83 % |
| Session establishment (HPKE + records) | 6, 7 | 57 %, 64 % | **61 %** | 0 % |
| Identity (did:sage, PoP, A2A, registration) | 8, 9, 10, 11 | 57 %, 64 %, 79 %, 50 % | **62 %** | 75 % |
| Delivery (transport/gateway, inspector, Rust, contracts, SDKs) | 12, 13, 14, 15, 16 | 33 %, 70 %, 67 %, 83 %, 13 % | **53 %** | 40 % |
| Governance (documentation, CI) | 17, 18 | 60 %, 75 % | **68 %** | 100 % |
| **Overall** | 18 rows | | **64 %** | **61 %** |

Row arithmetic: 1 = 6/7; 2 = 5.5/7; 3 = 6/7; 4 = 5/7; 5 = 4/7; 6 = 4/7; 7 = 4.5/7; 8 = 4/7; 9 = 4.5/7; 10 = 5.5/7; 11 = 2/4; 12 = 2/6; 13 = 3.5/5; 14 = 4/6; 15 = 2.5/3; 16 = 0.5/4; 17 = 3/5; 18 = 4.5/6.

Reading. [High] The layers that carry the *authenticity and replay* half of the purpose (primitives, signing) are between 71 % and 82 % and are live through the gateway. The layers that carry the *confidentiality* half (HPKE, session) are specified, implemented twice and vectored, but 0 % live: nothing shipped opens a session. Identity is live but its trust flags are partly unearned (row 9). Delivery is the weakest layer because the SDKs (13 %) and the transport layer (33 %) are where the "adoptable by any client" claim would have to be met.

Items that block the next steps, by the rows they would move (each is already tracked; the ID is given):

| Would move | Item | Tracked as | Effort (from source) |
|---|---|---|---|
| 6, 7 Live: N -> Y | wire HPKE handshake and session records into one shipped path (gateway "HPKE session mode", `sage-gateway/README.md:110`) with a Go-Rust conformance run | F-03b, gateway roadmap | L (`08` §3 recommendation) |
| 5, 6, 13 Insp: N -> P/Y | inspector P1 set: `-did`/`-authority` inputs, replay across a capture set, exact `;req` set, `init`/`ack`/`record` commands | `10` §3 P1 | S + M + S + M + M |
| 3-7 Vec: P -> Y | `rejected` vector schema for rfc9421, hpke, jcs; init-payload and ack-envelope verify vectors; stale/order session vector | `10` §3 vectors P1-P2 | M + M + S + S |
| 9 Live: P -> Y | verify PoP in `key_policy.go`, bind it to `chainid || registry` | `08` §6 recommendation, O-1 | M |
| 11 Live: P -> Y | `sage-did commit` loads the key file and commits the real key set | `08` 6.5 | S |
| 12 Live: P -> Y | TLS policy (`https`/`wss` required unless `AllowInsecure`) or documented reliance on the session layer | `08` §7 recommendation | S |
| 14 Live: P -> Y | live Go-Rust exchange in CI; HPKE in the FFI/WASM surface; `catch_unwind` | F-03b, `08` 7.2 | M |
| 16 all | bind SDKs to `rs-sage-core` or archive `sdk/*` | F-04 (deferred), D-08 | L |
| 17 Docs | rewrite `docs/handshake/*`, `docs/did/*`, `docs/cli/*`, `pkg/agent/{did,session}/README.md` from spec and `--help`; write E-06 | E-04, E-06 | M |
| 18 CI | pin `sage-spec` checkout to a tag; fuzz job with a budget; `-tags=integration` fixed or removed; Rust CI green | VERSION_POLICY §5, `08` 8.3, 8.6, F-08 | S each |

---

## 3. The wiring gap

Features that exist in the module but sit on no live request path, ranked by how much of the stated security claims depend on them. "Claim" cites the document that makes it. Reachability is from `02` §2e (graph walk) and re-verified by grep where marked.

| Rank | Feature (where it exists) | Live path today | Claim that depends on it | Severity | Conf. |
|---|---|---|---|---|---|
| 1 | HPKE 1-RTT handshake + session record layer (`pkg/agent/hpke`, `pkg/agent/session`) | none: no `cmd/`, example or gateway caller (verified grep; `02` §2e; `08` §3-§4) | "end-to-end encrypted ... channels" (`README.md:12`, verified); forward secrecy, MITM resistance (`ARCHITECTURE.md:461-476` per `04` §1) | [major] for the claim; not exploitable by itself | [High] |
| 2 | Transport confidentiality (TLS) | no `tls.Config` in `pkg`, `cmd`, `internal` (verified); `http://` registered as a transport; gateway has no TLS (`08` 7.1) | `ARCHITECTURE.md` assumes "endpoints are authenticated via TLS" (`04` §1); with rank 1 unwired, every live payload is plaintext | [major] | [High] |
| 3 | Key proof of possession on the resolve path (`did.VerifyKeyProofOfPossession`) | CLI `key verify-pop` and vectors only; `key_policy.go` trusts the on-chain `verified` flag, which the contract sets for Ed25519 after a length check (verified) | "DID-based identity with on-chain verification" against impersonation (`04` §1); `verified` semantics that the resolver and card verifier rely on (`08` 6.1) | [major] | [High] |
| 4 | Response/tool-result authentication end to end | `SignResponse`/`VerifyResponse` live in the gateway; nothing signs MCP results over stdio (gateway is HTTP only) | "malicious tool result injected into an agent" and "confused deputy" rows of `STRATEGY.md` §4 | [major] for the MCP claim | [High] |
| 5 | DoS cookie hooks (`hpke.CookieSource`, `CookieVerifier`) | no implementer in tree (`02` §1 interfaces); Go checks the cookie after DID resolution and signature verification (verified `server.go:147,159`; O-6) | "Denial of service on the verifier" row of `STRATEGY.md` §4 | [major] | [High] |
| 6 | Session lifetime limits on directional entry points | `IsExpired` called only by `Encrypt`/`Decrypt`/`EncryptAndSign`/`DecryptAndVerify` (verified `session.go:646,663,682,704`); the handshake-created role sessions use `EncryptOutbound`/`DecryptInbound` | session `MaxAge`/`IdleTimeout`/`MaxMessages` (`ARCHITECTURE.md`, `session/README.md`) | [major] (moot while rank 1 is unwired) | [High] |
| 7 | Persistent replay store (`pkg/storage` `NonceStore`, memory + postgres) | fan-in 0 from `pkg/agent` (`02` §1); every live guard is in-memory and unbounded (`09` §3) | replay protection across restarts; bounded verifier memory (`STRATEGY.md` §4 "bounded, TTL-evicting nonce store") | [minor] | [High] |
| 8 | Registration from the binary (`sage-did commit`) | sends empty keys and DID `TBD` (verified `commit.go:120-128`) | "on-chain trust" reachable by an operator without writing Go | [minor] | [High] |
| 9 | Envelope verifier (`core.VerificationService`, `rfc9421.Verifier.VerifySignature`) | no binary or example reaches it (`02` §2e) | none current; a second verifier surface with its own replay scope (agent DID) | [minor] | [High] |
| 10 | HPKE traffic keys / channel binding (`DeriveTrafficKeys`, spec 04 §4) | vector generator only (`09` §2c) | none; specified and vectored but unused (`08` 3.2) | [minor] | [High] |
| 11 | Encrypted key vault (`crypto/vault`), `X-SAGE-DID == keyid` in the core verifier, `ValidateA2ACardWithProofAndDID`, `hpke.KeyIDBinder` (only the deprecated `sessioninit` implements it) | vault: no importer; header check: gateway only; the two functions: no callers (`02` §1 dead code) | key-at-rest protection (`pkg/agent/crypto/README.md` per `01` §3); spec 03 §3 MUST | [minor] | [High] |

[High] Ranks 1-2 together mean the confidentiality claim is carried by tests and examples only; ranks 3-5 mean the identity and MCP claims are enforced at the gateway with weaker trust inputs than the documents state. Ranks 6-11 are hygiene until rank 1 is wired.

---

## 4. The 2026-09 cycle: delivered versus promised

Promised: `STRATEGY.md` §5 migration steps 0-7 with their gates. Delivered: BACKLOG status and `03` §4 (99 commits on 2026-09-11/12; CHANGELOG `[Unreleased]`).

| Step (STRATEGY §5) | Promised output and gate | Delivered | Not delivered / deviation |
|---|---|---|---|
| 0 Governance | rulesets, `security.yml`, secret scanning, SHA pins, digest pins, blocking scanners; gate "P0 list green" | A-01..A-05, A-08..A-11, A-13, A-15, A-18, A-19 Done/PR; signed, attested releases | signed tags (F08) pending; A-12 `tools/codegraph` Dependabot entry; A-14 lint exclusions; A-17 single version source Open |
| 1 Wire controls (v1.6.0) | HTTP replay/components/digest, HPKE session keys, secp256k1 alignment, resolver `verified`, `did.Manager` wiring, A2A card vs chain, contracts access control; gate "audit §11 a+b closed, consumers build" | B-01..B-13 merged; C-01, C-05, C-06 code fixed; `[Unreleased]` "Security" entries confirm each | B-14 cookie order Open (O-6); B-15 DoS budget Open; B-16 Decision; C-01 Sepolia redeploy Open; `v1.6.0` not tagged (VERSION_POLICY §5); external consumer builds not re-verified in this review |
| 2 Spec and vectors | profiles, MCP binding, negative vectors; gate "Go passes 100 % of its own vectors; negative vectors present" | `sage-spec` 1.0.0-draft.1, 9 chapters, 26 vectors, Go and Rust CI runs (F-01 Done) | MCP binding chapter missing; negative vectors only `did/parse.rejected` (`10` facts); O-1..O-8 open; spec untagged, CI at `ref: main` |
| 3 Contracts extraction | subtree split, ABI publishing, bindings on tag; gate "regenerated bindings match" | F-02 Done: `sage-contracts` with history, `abi/`, drift check | first tag `v1.5.0` pending; pin is a commit |
| 4 Go cleanup (v1.7.0) | phases 1-3, 5, 6; remove `sdk/`, `lib/`, `handshake`, `tests/random`; AES-GCM suite; gate "0 layer violations, 0 duplicates" | D-01..D-05 done with deviations; `lib/`, `tests/random`, `core/message/{dedupe,order,validator}` removed; layer violations 0 (`02` §1) | `handshake` deprecated, not removed (D-06); `sdk/` kept as experimental (D-08) instead of removed; AES-GCM suite dropped by decision (ChaCha20-Poly1305 mandatory, `REPO_PLAN.md` §4); "0 exact duplicates" not met (the `reports/bindings` copy, `02` §1); `v1.7.0` not tagged |
| 5 Rust core alignment | session realigned, vectors in CI, C header, `uniffi` + WASM, provenance; gate "passes vectors; ABI documented" | F-03 Done at vector level: 26/26, cbindgen header drift-checked, WASM build | live interoperability F-03b Open; `uniffi` and provenance not started; CI red on main (F-08 note); HPKE not in FFI |
| 6 SDKs | `sage-sdk-python` first; gate "Python SDK talks to gateway end to end" | nothing (deferred by plan, `REPO_PLAN.md` §1) | as planned; the four in-tree SDKs remain non-interoperable |
| 7 Gateway | MCP wrapper + HTTP proxy + A2A endpoint + recipes; gate "unmodified MCP client exchanges signed calls and results" | F-05 skeleton: verifying reverse proxy, signing forward proxy, resolver cache, response signing, three recipes, tests | stdio MCP bridge, A2A/HPKE endpoint, capability checks, rate limiting not done (`sage-gateway/README.md:57-58`); gate not automated (`04` §4) |
| not in the plan | | `sage-inspector` (F-07 skeleton), version policy and matrix (F-06), licence decision (A-16 closed, LGPL kept), Actions policy (F-08), `docs/archive/2026-09`, generated `docs/INDEX.md`, `pkg/vectors`, five review documents | |

Deviations from the strategy text, all recorded: mandatory AEAD reversed (`STRATEGY.md:62` vs `REPO_PLAN.md` §4); Apache-2.0 relicensing rejected (`STRATEGY.md:176` vs `LICENSING.md` §5); Rust repository named `rs-sage-core` not `sage-core`; `sdk/` marked rather than archived; MCP binding not prototyped in the gateway before the spec (STRATEGY §10 asked for that order; neither happened).

[High] Measured against the gates, steps 0, 2, 3 are met or met except for tagging; step 1 is met on the Go side and open on chain; steps 4, 5, 7 are partially met; step 6 was not started by decision. The cycle changed the delivery form (spec, five repositories, CI) more than it changed what a client can use today: the only new live capability for an unmodified client is the RFC 9421 gateway.

---

## 5. Risks to this assessment

| Risk | What was not done | Effect on the figures | Conf. |
|---|---|---|---|
| Nothing executed | no `go test`, no vector run, no gateway or inspector run, no contract suite; `08` ran one JCS scratch program; CI results are taken from workflow files and BACKLOG text | Y in Go/Rust/Vec columns means "code and CI configuration present", not "observed passing" | [High] |
| Sibling checkouts lag | gateway pins `sage` `b04477b`, inspector `5ab9c7e` (verified `go.mod`), both older than `878932d`; `RS_SAGE_CORE_ALIGNMENT.md` §2 predates Rust PRs #23-#25 (`02` §3) | the Live column reflects Go code a few PRs behind main | [High] |
| Rubric | equal column weights; n/a exclusions; a security-weighted rubric would lower Delivery and Identity further | overall 64 % is a coverage figure, not a security score | [High] |
| Reachability tooling | `02` §2a documents a codegraph over-approximation (`core/message/nonce` credited to binaries); negatives here were re-checked by grep, positives were not all re-walked | Live cells for rows 3, 8, 10 rely on `02` §2e | [Mid] |
| External consumers | `sage-a2a-go` and `sage-multi-agent` build status not re-checked after #316-#322 | step 1 gate "consumers build" is asserted by `STRATEGY.md` §1 at 2026-09-11 only | [Mid] |
| Unresolved counts | contracts 219 vs 209 `it(`; Rust 207 vs 481 `#[test]` (`01` §8) | Docs column for rows 14, 15 | [Low] |
| Not measured at all | coverage percentage (no threshold), replay-guard memory growth, registration latency and gas, revocation-to-rejection latency, formal properties of the handshake (`07` §6-§7) | no row can claim a quantitative security property | [High] |
| Moving spec target | CI checks out `sage-spec` at `main` (`test.yml:182` verified) | a spec change on main can move the Vec column without a `sage` commit | [High] |

---

## Facts vs Opinions

**Facts** (verified in the tree on 2026-09-12 or read in the cited documents)

- `cmd/` contains `sage-crypto`, `sage-did`, `sage-vectors`, `sage-verify`; no `cmd/` package, example or gateway package calls `hpke.NewServer`/`NewClient`; the session record API is called outside its package only by `pkg/vectors/session.go` and `examples/metrics-demo`.
- `pkg/agent/transport` contains no RFC 9421 call; `pkg`, `cmd`, `internal` contain no `tls.Config`, `TLSConfig` or `ListenAndServeTLS`.
- `pkg/agent/hpke/server.go:147` verifies the sender before the cookie check at `:159`; `pkg/agent/core/rfc9421/canonicalizer.go:232-241` emits `"METHOD path?query"` for `@request-target`.
- `cmd/sage-did/commit.go:120-128` commits with `Keys: [][]byte{}` and DID `did:sage:<chain>:TBD`.
- `session.IsExpired` is called only at `session.go:646,663,682,704`.
- `sage-spec` has 9 chapters and 26 vectors in 6 suites; `sage`'s CI checks them out at `ref: main`; `rs-sage-core` CI checks out the same repository; `rs-sage-core/src/ffi` has no HPKE file and `src/` has no `catch_unwind`.
- `sage-gateway` pins `sage` `b04477b`, `sage-inspector` pins `5ab9c7e`; the gateway README lists HPKE sessions and stdio MCP bridging as not done.
- No workflow runs fuzzing; no Go file carries `go:build integration`.
- `README.md:12` still opens with "A blockchain-based security framework ... end-to-end encrypted"; `docs/handshake/README.md:31` rates the deprecated handshake "Stable".
- BACKLOG: B-14, B-15, B-16, C-01 redeploy, C-04, D-06 removal, D-07 rewrite, E-06, F-03b, F-04 are Open or Decision; `v1.6.0`, `v1.7.0`, the first `sage-contracts` and `sage-spec` tags do not exist.

**Opinions**

- [High] Progress is real but lopsided: authenticity and replay protection are live through the gateway at roughly three quarters of the matrix; confidentiality is at zero on the live path, so the README's first sentence is not yet true of any shipped binary.
- [High] The single change with the largest effect on the figures is to give the HPKE handshake and session layer one shipped caller with a Go-Rust conformance run (F-03b plus the gateway session mode); until then rows 6, 7 and rank 1 of the wiring gap stay where they are regardless of how many vectors pass.
- [High] The 2026-09 cycle met its governance and specification gates and half of its wiring gate; the delivery gates (steps 6-7) are where the strategy's promise to "Codex, Claude Code, Hermes" still has no measurement.
- [Mid] The inspector and vector work (`10` §3 P1) is the cheapest way to convert "implemented" into "verified" for rows 3-7, and should precede any further repository split.
- [Mid] The overall figure would drop to roughly 55 % under a security-weighted rubric (Live and Vec at double weight); the direction of every recommendation above is unchanged by the weighting.
- [Low] External consumers still build against main; this was true on 2026-09-11 and six interface-changing PRs merged since.
