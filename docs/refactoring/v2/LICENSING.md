# Licence review (2026-09-12)

## 1. Summary

`sage` is licensed under LGPL-3.0 with the Solidity contracts under MIT. The
review found two problems that must be fixed before any of the new
repositories publishes a release, and one policy decision (A-16) that is still
open:

1. `rs-sage-core` declares `MIT OR Apache-2.0` in `Cargo.toml` and its README,
   but the repository's `LICENSE` file is the GPL-2.0 text. The crate cannot
   be published to crates.io in this state without misrepresenting its terms.
   Severity: Critical. Confidence: High.
2. Inside `sage`, `sdk/typescript/package.json` says `MIT` while the Python,
   Java and Rust SDKs say `LGPL-3.0` and the README says the whole project is
   LGPL-3.0 except `contracts/ethereum/`. Severity: Major. Confidence: High.
3. `sage-spec`, `sage-contracts`, `sage-gateway` and `sage-inspector` have no
   licence file. A repository with no licence grants no rights to anyone.
   Severity: Major until the first commit, Critical at first release.
   Confidence: High.

Recommendation: keep LGPL-3.0 for the Go core and apply a licence per
repository role (§4) instead of relicensing everything now. Relicensing the
Go core to Apache-2.0 (the option in `../STRATEGY.md` §7) remains possible but
requires consent from the three human contributors in the history and is
therefore tracked as A-16, not done as part of the split.

## 2. Current state

| Location | Declared licence | Where declared | Consistent? |
|---|---|---|---|
| `sage` | LGPL-3.0 | `LICENSE`, `README.md` §License, `NOTICE` | yes |
| `sage/contracts/ethereum` | MIT | `contracts/ethereum/LICENSE`, README | yes (explicit carve-out) |
| `sage/sdk/python` | LGPL-3.0 | `pyproject.toml` | yes |
| `sage/sdk/java/sage-client` | LGPL-3.0 | `pom.xml` | yes |
| `sage/sdk/rust/sage-client` | LGPL-3.0 | `Cargo.toml` | yes |
| `sage/sdk/typescript` | MIT | `package.json` | **no** (README carve-out covers contracts only) |
| `rs-sage-core` | MIT OR Apache-2.0 | `Cargo.toml`, README (links `LICENSE-APACHE`, `LICENSE-MIT`) | **no**: `LICENSE` is GPL-2.0; the two linked files do not exist |
| `sage-spec` | none | | **no** |
| `sage-contracts` | none | | **no** |
| `sage-gateway` | none | | **no** |
| `sage-inspector` | none | | **no** |

Third-party constraint that matters for the Go core: `sage` links
`github.com/ethereum/go-ethereum` library packages, which are LGPL-3.0 (the
GPL-3.0 entry in `NOTICE` comes from the repository-level `COPYING`, which
covers the `cmd/` binaries, not the library). LGPL-3.0 is compatible with a
Go core under LGPL-3.0 or Apache-2.0; either way downstream users of the Go
core inherit the LGPL relinking obligation for go-ethereum because Go links
statically.

Contributor set for a relicensing decision (from `git shortlog`): three human
authors in `sage` (the two maintainers and one earlier contributor) plus
Dependabot; two human authors in `rs-sage-core`. `CONTRIBUTING.md` asks for a
`Signed-off-by` line (DCO) but there is no CLA, so every past author must
agree explicitly before a licence change.

## 3. Fixes required before any release

| # | Repository | Fix | Severity |
|---|---|---|---|
| 1 | `rs-sage-core` | Replace the GPL-2.0 `LICENSE` with `LICENSE-APACHE` and `LICENSE-MIT` (the files the README already links), or change `Cargo.toml` and README to the licence actually intended. The dual licence is the intended one (STRATEGY §7, SDK embedding). | Critical |
| 2 | `sage` | Set `sdk/typescript/package.json` `license` to `LGPL-3.0-only` while the SDKs live in this repository. When the SDKs move to their own repositories they take the licence of `rs-sage-core`. | Major |
| 3 | `sage` | Add a sentence to `README.md` §License stating that SDKs under `sdk/` follow the repository licence until they are split out. | Recommended |
| 4 | `sage` | Correct the go-ethereum row of `NOTICE` to LGPL-3.0 (library packages) so the notice does not overstate the copyleft obligation. | Recommended |
| 5 | new repositories | Add `LICENSE` (and `NOTICE` where Apache-2.0) in the first commit, per §4. | Major |

Status on 2026-09-12: fix 1 is in `rs-sage-core` PR #16; fixes 2, 3 and 4 are
in `sage` PR #301; fix 5 landed as the initial commit of each new repository
(`sage-spec` Apache-2.0 with NOTICE, `sage-contracts` MIT, `sage-gateway` and
`sage-inspector` LGPL-3.0), and GitHub detects the expected licence on all
four.

## 4. Licence per repository

| Repository | Licence | Reason | Trade-off |
|---|---|---|---|
| `sage` | LGPL-3.0 (keep) | Already applied; relicensing needs contributor consent (A-16). | Embedding in proprietary Go binaries requires the relinking provision of LGPL-3.0 §4(d), which is awkward with static linking. This is the adoption cost STRATEGY §7 warned about. |
| `rs-sage-core` | MIT OR Apache-2.0 (fix the `LICENSE` file) | It is linked statically into WASM bundles and native addons; LGPL would push the relinking obligation onto every SDK user. Matches Rust ecosystem convention. | Permissive: downstream may ship modified cores without publishing changes. Mitigated by vectors and the spec, not by the licence. |
| `sage-spec` | Apache-2.0 | Implementers must be able to copy text, vectors and schemas into their own code and tests without copyleft; Apache-2.0 also carries a patent grant, which matters for a protocol. | Spec prose under a software licence is slightly unusual; CC-BY-4.0 for prose plus Apache-2.0 for vectors is the alternative, at the cost of two licence files. |
| `sage-contracts` | MIT | Carries over `contracts/ethereum/LICENSE`; matches on-chain source verification conventions and OpenZeppelin. | Permissive; no patent grant. |
| `sage-gateway` | LGPL-3.0 | It imports `sage`; using the same licence avoids a mixed-licence Go binary and any question about the LGPL boundary. | Same embedding cost as `sage`; the gateway is a standalone binary, so it rarely matters. |
| `sage-inspector` | LGPL-3.0 | Same reasoning as `sage-gateway`; it also imports `sage` for verification. | Same as above. If the inspector is later reduced to a vector runner with no `sage` import, Apache-2.0 becomes possible. |
| `sage-sdk-*` (deferred) | MIT OR Apache-2.0 | Thin bindings over `rs-sage-core`; must not be more restrictive than the core they wrap. | None beyond the permissive trade-off above. |

## 5. A-16 decision options

| Option | What changes | Cost | Benefit |
|---|---|---|---|
| A. Keep LGPL-3.0 on Go repositories (recommended now) | Only the fixes in §3 | Low; no consent process | Split proceeds immediately; Go-side embedding cost remains |
| B. Relicense `sage`, `sage-gateway`, `sage-inspector` to Apache-2.0 | `LICENSE`, `NOTICE`, README, SDK metadata, badge; written consent from the three `sage` authors | Consent process; must happen before `sage-gateway` and `sage-inspector` accept outside contributions or the set grows | Uniform permissive licensing across all repositories; removes the relinking obligation for Go consumers (go-ethereum's own LGPL obligation still applies) |

Option B can be taken later without redoing the split, because each new
repository starts with a single licence file and no outside contributors.
