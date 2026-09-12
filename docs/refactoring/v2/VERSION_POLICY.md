# Cross-repository version policy (F-06)

Status: policy, 2026-09-12. Automation items (signing, provenance,
changelog generation) are tracked separately in `BACKLOG.md` section A.

## 1. What is versioned where

| Repository | Artefact | Scheme | Today |
|---|---|---|---|
| `sage-spec` | the protocol: `spec/*.md` and `vectors/*.json` | SemVer of the protocol; `1.0.0-draft.N` until frozen | `1.0.0-draft.1` (untagged, main `a34dd49`) |
| `sage` (Go core) | Go module `github.com/sage-x-project/sage` | Go module tags `vMAJOR.MINOR.PATCH`, no `/v2` path move (DECISIONS.md 2) | `v1.5.2`; main carries the unreleased 1.6.0 changes |
| `sage-contracts` | Solidity sources, `abi/*.json` | SemVer of the ABI set; a tag is the unit `sage` pins | untagged (main `d9f313b`); first tag `v1.5.0` planned |
| `rs-sage-core` | crate `sage_crypto_core`, `include/sage_crypto.h`, WASM package | Cargo SemVer; `0.x` until live interoperability with the Go core is proven (F-03b) | `0.3.0` (main `284f754`) |
| `sage-gateway`, `sage-inspector` | Go binaries | Go module tags; each release names the `sage` version it was built against | untagged; pinned to `sage` pseudo-versions |
| `sage-sdk-*` (F-04, not started) | language packages | package-manager SemVer; each declares the `rs-sage-core` version it wraps | none |

## 2. Pins between repositories

Every dependency on another SAGE repository is an explicit, committed pin.
Nothing tracks a moving branch except the spec vector checkouts listed as
"to fix" below.

| Consumer | Provider | Pin | Updated by |
|---|---|---|---|
| `sage` | `sage-contracts` | `.contracts-version` (commit, later a tag); `make contracts-checkout`, `make bindings-check` | PR that bumps the pin and regenerates bindings |
| `sage` CI, `rs-sage-core` CI, `sage-inspector` CI | `sage-spec` | `actions/checkout` of `SAGE-X-project/sage-spec` at `ref: main` (to fix: pin `ref:` to the spec tag the implementation claims) | PR that bumps the ref |
| `sage-gateway`, `sage-inspector` | `sage` | `go.mod` pseudo-version or tag | `go get github.com/sage-x-project/sage@vX.Y.Z` in a PR |
| `sage-sdk-*` | `rs-sage-core` | crate version / header + `.wasm` from a release | release of the SDK |
| implementations | `sage-spec` | a `spec` declaration: `sage` README + `pkg/vectors` (suite list), `rs-sage-core` README, gateway/inspector README; must name the spec version whose vectors CI runs | PR that bumps the ref above |

## 3. Compatibility matrix

Updated in the PR that changes any pin. One row per released or
release-candidate combination; "main" rows are removed once tagged.

| sage-spec | sage (Go) | rs-sage-core | sage-contracts | sage-gateway | sage-inspector | Verified by |
|---|---|---|---|---|---|---|
| 1.0.0-draft.1 (main `a34dd49`) | main after #310 (`v1.5.2` + unreleased 1.6.0) | 0.3.0 (main `284f754`) | main `d9f313b` (pin `2ff992e`) | main `4e72668` (sage `b04477b`) | main `05b890d` (sage `5ab9c7e`) | 26 vectors in each CI; no live Go/Rust exchange yet (F-03b) |

## 4. Rules

1. **Spec first.** A protocol change lands in `sage-spec` (text and vectors) before any implementation; implementations then bump their spec ref in the same PR that makes them pass. A spec major bump is the only event that forces a coordinated release of every repository.
2. **Contracts tag, then repin.** `sage` never pins a contracts commit that is not on `sage-contracts` main; once `v1.5.0` exists, `.contracts-version` holds tags only and `make bindings-check` fails on drift.
3. **Go core: minor releases, deprecations for two minors.** Exported symbols are removed two minor releases after the release that marked them `Deprecated:` (the 1.6.0 changelog lists the current set). Breaking changes that cannot be shimmed go into a major release with a `/v2` module path, which is not planned.
4. **Rust core: `0.x` until F-03b.** `rs-sage-core` reaches `1.0.0` when a CI job exchanges an HPKE handshake and RFC 9421 messages with the Go core. Until then minor bumps may change the API; the C header (`include/sage_crypto.h`) is regenerated and drift-checked in CI, and a changed C signature is a minor bump.
5. **Gateway and inspector track the Go core.** They release after each `sage` tag they need, pin it in `go.mod`, and their release notes name it. They never use `replace` directives.
6. **Release order for a coordinated change:** `sage-spec` → `sage-contracts` → `sage` → `rs-sage-core` → `sage-gateway` / `sage-inspector` → SDKs. Each step's CI must be green on the pinned upstream versions before the next step tags.
7. **Tags are made by maintainers, from main, after CI is green;** CI builds release artefacts from tags only. Signing (Sigstore keyless), SLSA provenance and SBOMs are not in place yet; until they are, release notes state that artefacts are unsigned.

## 5. Open actions

| Action | Where | Status |
|---|---|---|
| Tag `sage-contracts` `v1.5.0` and switch `.contracts-version` to the tag | sage-contracts, sage | needs a maintainer to create the tag |
| Pin the sage-spec checkout `ref:` in the three CIs to a spec tag; tag `sage-spec` `v1.0.0-draft.1` | sage-spec, sage, rs-sage-core, sage-inspector | after the first spec tag exists |
| Add the `spec` declaration to the four READMEs | sage, rs-sage-core, sage-gateway, sage-inspector | open |
| Move the matrix in §3 to the `sage-spec` README once it has a second row | sage-spec | later |
| Tag `sage` `v1.6.0` | sage | Phase 3 is complete (2026-09-12); tag after the G-01 to G-05 wire-format fixes so that v1.6.0 already speaks the 1.0.0 protocol |
| Do not tag `sage-spec` `v1.0.0` until BACKLOG G-01 to G-05 are in the text and the vectors | sage-spec | decided 2026-09-13 |
| Deprecated code (including `handshake`) is removed in `v1.8.0`, not earlier | sage | decided 2026-09-13; BACKLOG D-06 aligned |
