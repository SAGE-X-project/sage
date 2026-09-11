# Dependency PR verification log (2026-09-11)

Branch: `integration/dependabot-2026-09` (local). Each open dependabot PR was squash-applied in the order below, then the ecosystem checks ran: Go = `go mod tidy` (no diff), build, vet, gofmt, golangci-lint, `go test ./...`; npm = `npm ci`, `hardhat compile`, solhint, `npm test` (202 tests); Docker = `docker build`; Java = `mvn clean test`. After every successful application the code graph was regenerated and diffed against the previous run.

| PR | Title | Result | Symbols / edges (before -> after) | External import changes |
|---|---|---|---|---|
| #214 | chore(deps)(deps): bump github.com/ethereum/go-ethereum from 1.17.0 to 1.17.3 | OK | 1730->1730 / 1423->1423 | none |
| #213 | chore(deps)(deps-dev): bump axios from 1.13.5 to 1.16.1 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #212 | chore(deps)(deps): bump golang.org/x/crypto from 0.48.0 to 0.51.0 | OK | 1730->1730 / 1423->1423 | none |
| #211 | chore(deps)(deps): bump github.com/gagliardetto/solana-go from 1.14.0 to 1.20.0 | OK (re-applied) | 1731->1731 / 1426->1426 | none |
| #210 | chore(docker)(deps): bump golang from 1.25.6-alpine to 1.26.3-alpine | OK | 1730->1730 / 1423->1423 | none |
| #209 | chore(deps)(deps-dev): bump fast-uri from 3.1.0 to 3.1.2 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #206 | chore(deps)(deps): bump go.opentelemetry.io/otel from 1.39.0 to 1.41.0 | OK (re-applied) | 1730->1730 / 1423->1423 | none |
| #203 | chore(deps)(deps): bump github.com/jackc/pgx/v5 from 5.8.0 to 5.9.2 | OK | 1730->1730 / 1423->1423 | none |
| #202 | chore(deps): bump org.bouncycastle:bcprov-jdk18on from 1.79 to 1.84 in /sdk/java/sage-client | OK | 1730->1730 / 1423->1423 | none |
| #201 | chore(deps): bump org.bouncycastle:bcpkix-jdk18on from 1.79 to 1.84 in /sdk/java/sage-client | OK | 1730->1730 / 1423->1423 | none |
| #200 | chore(deps)(deps-dev): bump follow-redirects from 1.15.11 to 1.16.0 in /contracts/ethereum | SUPERSEDED (already 1.16.0 via #213 axios) | - | - |
| #195 | chore(deps)(deps): bump lodash from 4.17.23 to 4.18.1 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #194 | chore(deps)(deps): bump picomatch from 2.3.1 to 2.3.2 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #192 | chore(deps)(deps): bump github.com/mr-tron/base58 from 1.2.0 to 1.3.0 | OK | 1730->1730 / 1423->1423 | none |
| #190 | chore(deps)(deps-dev): bump handlebars from 4.7.8 to 4.7.9 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #187 | chore(deps)(deps): bump undici from 6.23.0 to 6.24.1 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #186 | chore(deps)(deps): bump golang.org/x/sync from 0.19.0 to 0.20.0 | SUPERSEDED (already v0.20.0 via #214/#212) | - | - |
| #183 | chore(deps)(deps-dev): bump hardhat from 3.1.8 to 3.1.10 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #182 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-typechain from 3.0.2 to 3.0.3 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #181 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-mocha from 3.0.9 to 3.0.11 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #180 | chore(deps)(deps): bump github.com/decred/dcrd/dcrec/secp256k1/v4 from 4.4.0 to 4.4.1 | OK (re-applied) | 1730->1731 / 1423->1426 | none |
| #179 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-keystore from 3.0.4 to 3.0.5 in /contracts/ethereum | OK (re-applied) | 1731->1731 / 1426->1426 | none |
| #178 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-ignition-ethers from 3.0.7 to 3.0.8 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #177 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-ethers-chai-matchers from 3.0.2 to 3.0.3 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #176 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-verify from 3.0.10 to 3.0.11 in /contracts/ethereum | OK (re-applied) | 1731->1731 / 1426->1426 | none |
| #175 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-ignition from 3.0.7 to 3.0.8 in /contracts/ethereum | OK (re-applied) | 1731->1731 / 1426->1426 | none |
| #174 | chore(deps)(deps): bump minimatch from 10.2.1 to 10.2.4 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #171 | chore(deps)(deps-dev): bump chai from 5.3.3 to 6.2.2 in /contracts/ethereum | OK | 1730->1730 / 1423->1423 | none |
| #170 | chore(deps)(deps): bump filippo.io/edwards25519 from 1.1.1 to 1.2.0 | OK | 1730->1730 / 1423->1423 | none |

Notes:

- #211 (solana-go 1.20.0): `rpc.Client.GetRecentBlockhash` was removed upstream; `pkg/agent/did/solana/client.go` now calls `GetLatestBlockhash`.
- #180 (decred secp256k1 4.4.1): `S256()` is deprecated (SA1019); added `keys.Secp256k1Curve()` with a single suppression and used it from `formats/pem.go` and `formats/jwk.go`.
- #186 (x/sync 0.20.0) and #200 (follow-redirects 1.16.0): already pulled in by #214/#212 and #213 respectively; squash-applying them produced no diff. Dependabot closes such PRs once main carries the version.
- Conflicting PRs (#211, #206, #180, #179, #176, #175) were re-applied by running the equivalent `go get` / `npm install` pinned to the PR's target version, because their branches were based on an older main.
- Symbol/edge counts do not change for pure dependency bumps; the +1 symbol / +3 edges at #180 are the new `Secp256k1Curve` helper and its three callers.

## Batch 2 (2026-09-11, integration/dependabot-2026-09-11)

Same procedure as batch 1. Dependabot opened 20 PRs after the security workflow and action pinning changes; 18 were applied (6 re-applied after lockfile/go.sum conflicts, pinned to the PR version) and 2 were closed because the target release requires Go 1.26 (Dependabot ignore rules added).

| PR | Title | Result | Symbols / edges | External import changes |
|---|---|---|---|---|
| #225 | chore(docker)(deps): bump golang from 1.26.3-alpine to 1.27.1-alpine | OK | 1752->1752 / 1457->1457 | none |
| #226 | chore(deps)(deps-dev): bump prettier from 3.8.1 to 3.9.6 in /contracts/ethereum | OK | 1752->1752 / 1457->1457 | none |
| #227 | chore(deps)(deps): bump github.com/stretchr/testify from 1.11.1 to 1.12.1 | OK | 1731->1752 / 1426->1457 | none |
| #228 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-ignition-ethers from 3.0.8 to 3.1.6 in /contracts/ethereum | OK | 1752->1752 / 1457->1457 | none |
| #229 | chore(deps)(deps): bump github.com/gagliardetto/solana-go from 1.20.0 to 1.23.0 | OK | 1752->1752 / 1457->1457 | none |
| #230 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-chai-matchers from 2.1.0 to 3.0.0 in /contracts/ethereum | OK | 1752->1752 / 1457->1457 | none |
| #231 | chore(deps)(deps): bump golang.org/x/crypto from 0.55.0 to 0.56.0 | CLOSED (requires Go 1.26; Dependabot ignore added) | - | - |
| #232 | chore(deps)(deps): bump github.com/prometheus/client_golang from 1.23.2 to 1.24.1 | OK (re-applied) | 1752->1752 / 1457->1457 | none |
| #233 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-verify from 3.0.11 to 3.1.0 in /contracts/ethereum | OK | 1752->1752 / 1457->1457 | none |
| #234 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-typechain from 3.0.3 to 3.1.1 in /contracts/ethereum | OK (re-applied) | 1752->1752 / 1457->1457 | none |
| #235 | chore(deps)(deps): bump github.com/ethereum/go-ethereum from 1.17.3 to 1.17.5 | OK (re-applied) | 1752->1752 / 1457->1457 | none |
| #236 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-ignition from 3.0.8 to 3.1.8 in /contracts/ethereum | OK (re-applied) | 1752->1752 / 1457->1457 | none |
| #237 | chore(deps)(deps): bump github.com/cloudflare/circl from 1.6.3 to 1.6.5 | OK | 1752->1752 / 1457->1457 | none |
| #238 | chore(deps)(deps): bump golang.org/x/sync from 0.22.0 to 0.23.0 | CLOSED (requires Go 1.26; Dependabot ignore added) | - | - |
| #239 | chore(deps)(deps-dev): bump ethers from 6.16.0 to 6.17.0 in /contracts/ethereum | OK (re-applied) | 1752->1752 / 1457->1457 | none |
| #240 | chore(deps)(deps): bump github.com/jackc/pgx/v5 from 5.9.2 to 5.11.0 | OK | 1752->1752 / 1457->1457 | none |
| #241 | chore(deps)(deps-dev): bump @nomicfoundation/hardhat-keystore from 3.0.5 to 3.0.13 in /contracts/ethereum | OK (re-applied) | 1752->1752 / 1457->1457 | none |
| #242 | chore(deps)(deps-dev): bump hardhat from 3.1.10 to 3.16.0 in /contracts/ethereum | OK | 1752->1752 / 1457->1457 | none |
| #243 | chore(deps)(deps): bump @openzeppelin/contracts from 5.4.0 to 5.6.1 in /contracts/ethereum | OK | 1752->1752 / 1457->1457 | none |
| #244 | chore(ci)(deps): bump the actions group with 14 updates | OK | 1752->1752 / 1457->1457 | none |

Notes:

- #244 (actions group) moves every pinned action to a new major (checkout v7, setup-go v7, upload-artifact v7, download-artifact v8, codecov v7, golangci-lint-action v9, docker/* v4/v6/v7, action-gh-release v3); the consolidating PR's own CI run is the verification.
- #242 (hardhat 3.16.0) and #230 (hardhat-chai-matchers 3.0.0) resolve the peer-range mismatch that previously kept hardhat 2 plugins in the tree; all Hardhat plugin bumps that conflicted (#234, #236, #241) were re-applied at the exact PR versions.
- #231 (x/crypto 0.56.0) and #238 (x/sync 0.23.0) require Go 1.26 and were closed; `.github/dependabot.yml` now ignores those ranges until the toolchain moves.
