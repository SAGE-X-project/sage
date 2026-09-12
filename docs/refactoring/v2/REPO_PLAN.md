# Repository plan (final, 2026-09-12)

## 1. Summary

`rs-sage-core` is the right project to serve as the Rust core; no new Rust
repository is needed. Its module layout, `cdylib + rlib` build and CI already
match the target described in `../STRATEGY.md` §2.3. What does not match is the
protocol layer: the crate ports an older revision of the Go core (four-phase
handshake, AES-256-GCM sessions with a counter nonce, secp256k1 over SHA-256)
and is therefore not wire-compatible with the current Go core. F-03 is defined
as "align `rs-sage-core` to `sage-spec` and shrink its scope", not as a rewrite
in a fresh repository.

Execution order: F-01 `sage-spec` → F-02 `sage-contracts` → F-05
`sage-gateway` → `sage-inspector` → F-03 `rs-sage-core` alignment → D-04 3.1 /
D-03 2.7 unification (approved 2026-09-12). SDK repositories are deferred until
`rs-sage-core` publishes a C header and a WASM artifact.

## 2. Assessment of `rs-sage-core`

Surveyed at `rs-sage-core` HEAD on 2026-09-12: crate `sage_crypto_core` 0.3.0,
`crate-type = ["cdylib", "rlib"]`, about 29,700 lines of Rust, features
`ffi` (libc), `wasm` (wasm-bindgen), `blockchain` (alloy 0.7, solana-sdk).
Modules: `crypto`, `rfc9421`, `hpke`, `session`, `handshake`, `transport`,
`did`, `ffi`, `wasm`, `blockchain`, `validation`, `formats`, `core`. CI runs
`cargo fmt`, `clippy -D warnings`, tests and a security workflow; a release
workflow publishes to crates.io.

### What already matches the target

- Module boundaries map one-to-one onto the Go clusters
  (`pkg/agent/{crypto,core/rfc9421,hpke,session,transport,did}`).
- The crate is built as a C-compatible shared library and has a WASM feature,
  which is exactly the surface the SDKs need.
- Licence metadata in `Cargo.toml` is `MIT OR Apache-2.0`, the SDK-friendly
  choice recommended in `../STRATEGY.md` §7 (but see `LICENSING.md` §3: the
  `LICENSE` file contradicts it).

### What diverges from the Go core and must change (F-03)

| Area | `rs-sage-core` today | Go core convention (fixed) | Severity | Confidence |
|---|---|---|---|---|
| secp256k1 signature hash | `ecdsa-secp256k1-sha256` (SHA-256) | Keccak-256, RFC 6979, low-S, `r‖s‖v` (`pkg/agent/crypto/keys/secp256k1_keccak.go`) | Critical: signatures do not cross-verify | High |
| Session AEAD and nonce | AES-256-GCM, nonce = hash(session id) ‖ 8-byte counter | ChaCha20-Poly1305, sequence header, 1024-entry replay window, rekey | Critical: session messages do not decrypt | High |
| Handshake | 4-phase Invitation / Request / Response / Complete | Removed from the Go core (B-16); HPKE 1-RTT only | Major: duplicates deleted code | High |
| JCS (RFC 8785) | Absent; `rfc9421/canonicalize.rs` only serialises signature components | `pkg/agent/crypto/jcs` is mandatory for JSON payloads | Major | High |
| Ed25519 in RFC 9421 | Not in the algorithm enum (`ecdsa-p256-sha256`, `ecdsa-secp256k1-sha256`, `rsa-v1_5-sha256`, `rsa-pss-sha512`) | Ed25519 is the default signing algorithm | Major | Mid (enum checked, signer internals not) |
| C FFI | Four exports: `sage_init`, `sage_version`, `sage_last_error`, `sage_clear_error` | SDKs need sign / verify / HPKE / session entry points | Major: effectively a stub | High |
| Blockchain module | Re-implements on-chain reads with alloy and solana-sdk | On-chain reads belong to the Go core and gateway; the Rust core is pure crypto and protocol | Recommended: drop | Mid |
| Dependency age | k256 0.11, http 0.2, reqwest 0.11, base64 0.21 | Two to three major versions behind | Recommended: supply-chain review | High |

The `rfc9421`, `session`, `hpke` and `handshake` modules carry most of the
divergence, so F-03 is close to a rewrite of those four modules even though
the repository, build and CI are kept.

## 3. Repositories and roles

| Repository | Role | Language | Published artifacts | State on 2026-09-12 |
|---|---|---|---|---|
| `sage` | Reference core: protocol implementation, CLIs, test-vector generator (`cmd/sage-vectors`) | Go | Go module, binaries, vectors | active |
| `rs-sage-core` | Rust core: spec-conformant implementation, C ABI, WASM; common substrate for SDKs | Rust | crate, C header, `.wasm` | needs F-03 alignment |
| `sage-spec` | Protocol specification and golden test vectors; the single interoperability reference | Markdown, JSON | profile documents, `vectors/` | draft 1.0.0-draft.1, 26 vectors (F-01 done 2026-09-12) |
| `sage-contracts` | Solidity and Anchor contracts, deployment scripts, ABI publishing; Go bindings generated on tag | Solidity, Rust | ABI JSON, address registry | imported with history, `abi/` published, CI green (F-02 done 2026-09-12; first tag pending) |
| `sage-gateway` | MCP / A2A wrapper, HTTP signing proxy, client recipes; imports the core, never the reverse | Go | binary, container image | skeleton merged: verifying and signing proxies, recipes (F-05, 2026-09-12) |
| `sage-inspector` | Spec conformance checker: vector runner, RFC 9421 / HPKE / A2A message inspector, optional capture proxy | Go | CLI | skeleton merged: vector runner, request/response/card inspection (F-07, 2026-09-12) |
| `sage-sdk-python`, `-typescript`, `-java` | Thin bindings over `rs-sage-core` | per language | packages | deferred; nothing to build until the C header and WASM exist |

Dependency direction:

```
sage-spec  <--  sage, rs-sage-core  <--  sage-gateway, sage-inspector
sage-contracts (ABI only)  <--  sage, sage-gateway
```

No repository imports in the reverse direction. `sage-spec` contains no code
that depends on either core; the cores prove conformance by running the
vectors in CI.

## 4. Execution order and deliverables

1. **F-01 `sage-spec` bootstrap.** Write the profiles from the fixed Go code:
   RFC 9421 component set and `;req` binding, secp256k1 Keccak convention,
   JCS, HPKE labels and export contexts, session sequence / replay / rekey,
   ChaCha20-Poly1305 as the mandatory AEAD. Generate vectors with
   `cmd/sage-vectors` and commit them to `sage-spec`. From this point every
   implementation must pass the vectors in CI.
2. **F-02 `sage-contracts` extraction.** Move `contracts/`, add an ABI
   publishing workflow and Go binding generation on tag. `make bindings-check`
   in `sage` is repointed at the released ABI.
3. **F-05 `sage-gateway` skeleton.** Promote the verification logic of
   `examples/mcp-integration` into a package; add the HTTP proxy and the
   client recipes.
4. **`sage-inspector`.** Vector runner first (needs step 1), message inspector
   second, capture proxy last.
5. **F-03 `rs-sage-core` alignment.** Fix the Critical and Major rows of §2 in
   order, delete the four-phase handshake and the `blockchain` module, add the
   vectors to CI, produce the C header (`cbindgen`) and the WASM build as CI
   artifacts. Only then decide whether SDK repositories are created.
6. **D-04 3.1 / D-03 2.7 unification** (approved). Keep v1 compatibility with
   aliases and wrappers where possible; document the breaking parts in the
   changelog.

Trade-off of this order: `rs-sage-core` stays non-interoperable until step 5
while its README still advertises RFC 9421 and HPKE compatibility. Add a
banner to the `rs-sage-core` README at the start of step 1 stating that it is
not compatible with the Go core v1.5 wire format and is being aligned to
`sage-spec`.

## 5. Open decisions

| Item | Situation | Recommendation |
|---|---|---|
| C-01 Sepolia redeploy | `ERC8004ReputationRegistry` (`0xE953…`) already has `validationRegistry()` set to `0x9729…`, so the initial-set gap is closed on chain. The remaining issue is that `ERC8004ValidationRegistry` at `0x9729…` runs pre-fix code with open admin setters. Testnet only. | Defer to just before mainnet or a public demo; mark C-01 "deferred" in `../BACKLOG.md`. |
| A-16 licence | Decided 2026-09-12: Go repositories stay LGPL-3.0; `rs-sage-core` MIT OR Apache-2.0 (LICENSE files fixed in PR #16); new repositories licensed at their first commit. | Closed; see `LICENSING.md` §5. |
| GitHub Actions on new repositories | The organisation's Actions policy allows workflows only for selected repositories: `rs-sage-core`, `sage-spec`, `sage-contracts`, `sage-gateway` and `sage-inspector` report `enabled: false`, and the repository-level toggle is refused with "disabled on this repository by the organization". All six repositories are public, so GitHub-hosted runner minutes are not billed (sage runs report 0 billable minutes); this is a policy setting, not a quota. | An organisation owner adds the five repositories under Settings > Actions > General > Policies before F-01 (vector CI) and F-03 start. |
| F08 tag signing | cosign keyless for release tags | pending |
| SDK repositories | nothing to bind until `rs-sage-core` exposes a C header and WASM | do not create yet |

## 6. Evidence

- `rs-sage-core/Cargo.toml`: crate name, version, `crate-type`, features,
  dependency versions listed in §2.
- `rs-sage-core/src/rfc9421/mod.rs`: four-entry algorithm enum.
- `rs-sage-core/src/session/secure_session.rs`: counter-derived 96-bit nonce,
  AES-GCM.
- `rs-sage-core/src/handshake/mod.rs`: four-phase handshake documentation.
- `rs-sage-core/src/ffi/*.rs`: four `pub extern "C"` functions.
- `sage/pkg/agent/crypto/keys/secp256k1_keccak.go`: Keccak-256 + RFC 6979 +
  low-S as the single secp256k1 implementation.
- Sepolia `validationRegistry()` read via a public RPC on 2026-09-11.
