# SAGE Supply-Chain, CI, Release and Version-Management Audit

- Repository: `SAGE-X-project/sage` (public, default branch `main`)
- Audited revision: `14283ce` (`main` == `origin/main` at audit time), 2026-09-11
- Method: read-only inspection of the working tree plus `gh api` / `gh run` / `git` read commands.
  Every claim cites `file:line` or the command whose output produced it. Items that could not be
  verified are marked **[not verified]**.
- Severity: `[치명]` (exploitable / integrity-breaking), `[중요]` (material weakness), `[권장]` (hygiene).
  Confidence: `[High]` direct evidence, `[Mid]` inference from evidence, `[Low]` estimate.

---

## 1. Summary

1. `[치명]` **`main` is effectively unprotected.** The only ruleset (`main`, id 8640770) has
   `conditions.ref_name.include: []`, so it matches no branch; `GET /branches/main/protection`
   returns 404 "Branch not protected" and `GET /rules/branches/main` returns `[]`. Four accounts with
   `write` plus team `sage-core-dev` (`push`) can commit directly, force-push, and push `v*` tags.
   A pushed tag runs `release.yml` with workflow-level `contents: write` and publishes release
   binaries with no review gate. `[High]`
2. `[치명]` **The Security workflow is disabled** (`state: disabled_inactivity`). Its last run is
   2026-07-20 (schedule). None of the 32 commits pushed to `main` in the last 90 days (including the
   2026-09-10/11 dependency and Docker security fixes) were scanned by CodeQL, gosec, Slither,
   gitleaks or dependency-review. CodeQL default setup is `not-configured`, so there is currently
   **no SAST or PR dependency review at all**. `[High]`
3. `[중요]` Secret scanning, push protection and validity checks are **disabled** on a public repo
   where they are free. `[High]`
4. `[중요]` 44 of 45 `uses:` references are tag-pinned (`@v5`, `@v2`, ...); only
   `aquasecurity/trivy-action` is SHA-pinned (`docker.yml:113`). Repo setting
   `sha_pinning_required: false`, `allowed_actions: all`. `[High]`
5. `[중요]` Releases are unsigned (no cosign/sigstore), carry no SBOM or SLSA provenance, and are
   built non-reproducibly (`BUILD_DATE` with seconds, no `-trimpath`, CGO left enabled, three
   different runner OSes). Tags are annotated but unsigned (`git verify-tag v1.5.2`: "no signature
   found"); commits are unsigned (`%G?` = `N`). `[High]`
6. `[중요]` Security tooling is wired so that it cannot fail CI: gosec, Slither, gitleaks and
   license checks all end in `|| true`, `--exit-code=0`, `-no-fail` or `continue-on-error: true`. `[High]`
7. `[중요]` Dependabot covers 4 of 10 manifests; `sdk/typescript`, `sdk/python`, `sdk/java`,
   `sdk/rust`, `contracts/solana`, `tools/codegraph` are unmonitored; `sdk/typescript` has no
   lockfile and `Cargo.lock` is git-ignored. `[High]`
8. `[권장]` Version sources disagree: `VERSION`/`pkg/version` 1.5.2, `lib/export.go` 1.3.1,
   `contracts/ethereum/package.json` 1.5.0, `sdk/typescript` 1.0.0, Java/Python/Rust/Solana 0.1.0. `[High]`

---

## 2. Findings table

| ID | Area | Finding | Evidence | Risk | Conf. |
|----|------|---------|----------|------|-------|
| F01 | Governance | Ruleset `main` matches no branch (`ref_name.include: []`); `main` and `dev` report `protected=false`; no required status checks, 0 required approvals, no signed-commit or linear-history rule | `gh api repos/SAGE-X-project/sage/rulesets/8640770` -> `"conditions":{"ref_name":{"exclude":[],"include":[]}}`, rules `deletion`, `non_fast_forward`, `pull_request{required_approving_review_count:0}`; `gh api .../branches/main/protection` -> 404; `gh api .../rules/branches/main` -> `[]` | [치명] | [High] |
| F02 | Governance | Anyone with write can push tags -> `release.yml` publishes binaries with `contents: write`; no `environment:` gate, no tests in the release path | `release.yml:3-9` (`on.push.tags: v*.*.*`, top-level `permissions: contents: write`); collaborators: `sujine2 write, onlyhyde admin, 0xwiederholen write, 0xmhha admin, Learning-N-Running write, scottXchoo write`; team `sage-core-dev` permission `push` | [치명] | [High] |
| F03 | CI | Security workflow disabled; CodeQL/gosec/Slither/gitleaks/dependency-review not executed on any 2026-09 push or PR | `gh api .../actions/workflows` -> `Security | .github/workflows/security.yml | disabled_inactivity`; `gh run list --workflow=security.yml` last run `2026-07-20 schedule`; `gh run list --limit 20` shows only Test/Integration/Docker/loadtest for 2026-09-10..11; `code-scanning/default-setup` -> `"state":"not-configured"` | [치명] | [High] |
| F04 | Governance | Secret scanning, non-provider patterns, push protection, validity checks all disabled (public repo) | `gh api repos/SAGE-X-project/sage` -> `security_and_analysis.secret_scanning.status: disabled`, `secret_scanning_push_protection: disabled`; `secret-scanning/alerts` -> "Secret scanning is disabled" | [중요] | [High] |
| F05 | Workflows | 44/45 `uses:` are mutable tag refs; only trivy-action is SHA-pinned; repo does not require SHA pinning | Section 3.1 table; `docker.yml:113`; `gh api .../actions/permissions` -> `"allowed_actions":"all","sha_pinning_required":false` | [중요] | [High] |
| F06 | Release | Binaries unsigned; no SBOM; no SLSA provenance; release notes are `git log` dump; `softprops/action-gh-release@v2` unpinned | `release.yml:78-86` (sha256 only), `:124-137`, `:140`; `gh release view v1.5.2` assets = 5 archives + `SHA256SUMS` only | [중요] | [High] |
| F07 | Release | Non-reproducible build: `BUILD_DATE` to the second, `GIT_BRANCH` baked in, no `-trimpath`, CGO not disabled, each OS built on a different runner image | `Makefile:19-33` (`GOFLAGS=-v`, `LDFLAGS=-w -s`, `BUILD_DATE?=$(shell date -u '+%Y-%m-%d %H:%M:%S UTC')`), `Makefile:233-247` (`build-platform`, no `-trimpath`/`CGO_ENABLED`), `release.yml:20-40` matrix `ubuntu/macos/windows-latest`; grep for `trimpath|mod=readonly|govulncheck` in `Makefile` and `.github/` -> no matches | [중요] | [High] |
| F08 | Release | Tags annotated but unsigned; commits unsigned; `web_commit_signoff_required: true` only affects web UI commits | `git for-each-ref refs/tags` -> all 13 tags `type=tag tagger=0xTopaz`; `git verify-tag v1.5.2` -> `error: no signature found`; `git log -15 --format='%G?'` -> all `N` | [중요] | [High] |
| F09 | Workflows | Security jobs cannot fail: gosec `-no-fail ... \|\| true` + `continue-on-error: true`; Slither `\|\| true`; gitleaks `--exit-code=0 \|\| true`; go-licenses/license-checker `continue-on-error: true` | `security.yml:86,88,90,91`, `:194`, `:256`, `:279`, `:283` | [중요] | [High] |
| F10 | Workflows | Unpinned tool installs at run time: `go install ...@latest` (gosec, go-licenses, migrate), `pip3 install slither-analyzer` (no version), `zricethezav/gitleaks:latest`, golangci-lint `version: latest`, `npx license-checker` (not a declared devDependency, fetched ad hoc) | `security.yml:83,189,256,276,286-287`; `test.yml:108`; `loadtest.yml:94,214,297` | [중요] | [High] |
| F11 | Workflows | `npm ci` runs lifecycle scripts (no `--ignore-scripts`) in 4 places; no `go mod verify`, `-mod=readonly` or `govulncheck` anywhere | `test.yml:73,133`; `integration-test.yml:54`; `security.yml:180`; grep result in F07 | [중요] | [High] |
| F12 | Container | Runtime base `alpine:latest` (mutable, no digest); builder `golang:1.26.3-alpine` tag-only; Dependabot `docker` ecosystem cannot pin/track digests until they exist | `Dockerfile:5`, `:33`; `.github/dependabot.yml:59-62` | [중요] | [High] |
| F13 | Container | Toolchain mismatch: `go.mod` says `go 1.25.2`, CI builds with 1.25.2, Docker image builds with Go 1.26.3 (`GOTOOLCHAIN=auto` allows it); release tarballs and GHCR image are produced by different compilers. `x/crypto` is held at 0.55.0 because 0.56+ needs Go 1.26 | `go.mod:3`; `test.yml:10,20`; `release.yml:12`; `Dockerfile:5`; `go.mod:19` `golang.org/x/crypto v0.55.0`; `CHANGELOG.md:23` still says `golang:1.25.6-alpine` | [중요] | [High] |
| F14 | Container | `RUN make build-lib \|\| true` hides c-archive/c-shared build failure inside the image build; builder installs `gcc`/`musl-dev`, so `make build` links CGO by default | `Dockerfile:8-12`, `:27`, `:30` | [권장] | [High] |
| F15 | Container | Trivy skips `/usr/local/bin` so the Go binaries in the image are never scanned for module CVEs; the comment claims gosec/CodeQL cover this but those are SAST, not vulnerability DB scans, and are disabled (F03) | `docker.yml:118-127` | [중요] | [High] |
| F16 | Dependencies | Dependabot covers `gomod:/`, `github-actions:/`, `npm:/contracts/ethereum`, `docker:/` only. Missing: `gomod:/tools/codegraph`, `npm:/sdk/typescript`, `pip:/sdk/python`, `maven:/sdk/java/sage-client`, `cargo:/sdk/rust/sage-client`, `cargo:/contracts/solana`. No `groups:`. `reviewers: sage-x-project/maintainers` -> team does not exist | `.github/dependabot.yml:7-73`; `tools/codegraph/go.mod:1`; `gh api orgs/SAGE-X-project/teams/maintainers` -> 404; `gh api orgs/SAGE-X-project/teams` -> only `sage-core-dev` | [중요] | [High] |
| F17 | Dependencies | Lockfiles: `sdk/typescript` has **no** lockfile; `Cargo.lock` is git-ignored (so `sdk/rust` and `contracts/solana` are unlocked); `sdk/python` uses open `>=` ranges with no lock; root `package-lock.json` is an empty stub with no `package.json` | `find` output (only `contracts/ethereum/package-lock.json`, `go.sum`, `tools/codegraph/go.sum`); `.gitignore:456` `Cargo.lock`; `sdk/python/requirements.txt:4-7`; root `package-lock.json` `"packages": {}` | [중요] | [High] |
| F18 | Dependencies | `go.mod` has no `replace`/`exclude`/`retract`; `go.sum` present; direct deps 19. `github.com/test-go/testify v1.1.4` is a fork of testify used alongside `stretchr/testify` | `grep -n '^(replace|exclude|retract)' go.mod` -> none; `go.mod:5-24` | [권장] | [High] |
| F19 | Workflows | `loadtest.yml` is syntactically invalid (`inputs:` nested under `env:`), so GitHub names the workflow by file path and **every push produces a failed run**; all three jobs are `if: false` and reference non-existent `tests/handshake/server/main.go` | `loadtest.yml:17-30`, `:35`, `:107`, `:166`, `:216`, `:260`, `:299`; `gh run view` -> "This run likely failed because of a workflow file issue"; `ls tests/handshake` -> No such file | [권장] | [High] |
| F20 | Workflows | `integration-test.yml` e2e job `if: false`, references `./test/e2e/...` (does not exist; real dir is `tests/`) | `integration-test.yml:119`, `:148`, `:163`; `ls test/e2e` -> No such file | [권장] | [High] |
| F21 | Workflows | `permissions:` absent in `test.yml`, `integration-test.yml`, `loadtest.yml`, and in `security.yml` jobs `dependency-review`, `gitleaks`, `license-check`. Repo default is `read` (so not write-all today), but this is a repo setting, not code, and Scorecard flags it | `grep -c 'permissions:'` -> test 0, integration 0, loadtest 0, docker 2, release 2, security 3; `gh api .../actions/permissions/workflow` -> `default_workflow_permissions: read` | [권장] | [High] |
| F22 | Workflows | `release.yml` sets `contents: write` at workflow level, so all five `build-release` matrix jobs and `docker-release` inherit write while only `create-release` needs it | `release.yml:8-9` | [권장] | [High] |
| F23 | Workflows | Gitleaks scans working tree only (`--no-git`), never history; `.gitleaksignore` suppresses 19 fingerprints | `security.yml:256`; `.gitleaksignore` | [권장] | [High] |
| F24 | Workflows | Excluded paths that no longer exist: `contracts/ethereum/bindings` (gosec, gofmt, golangci) | `security.yml:86-90`; `test.yml:117`; `.golangci.yml:23`; `ls contracts/ethereum/bindings` -> No such file | [권장] | [High] |
| F25 | Governance | Missing `SECURITY.md`, `CODEOWNERS`, `CODE_OF_CONDUCT.md`, `.editorconfig`, pre-commit config. Issue template links to `/security/policy` which has no backing file | `ls SECURITY.md .github/SECURITY.md CODEOWNERS .editorconfig .pre-commit-config.yaml` -> none; `.github/ISSUE_TEMPLATE/config.yml:9-11`; `CONTRIBUTING.md:18` has an inline CoC section only | [권장] | [High] |
| F26 | Governance | `CONTRIBUTING.md` claims `main` requires 2 approvals + status checks and `dev` 1 approval; reality is 0 approvals and no enforcement (F01) | `CONTRIBUTING.md:168-179` vs F01 evidence | [권장] | [High] |
| F27 | License | Root/Go/Java/Python/Rust are LGPL-3.0; `contracts/ethereum`, `contracts/solana` are MIT (declared in CONTRIBUTING); `sdk/typescript/package.json` says MIT with **no LICENSE file** although `files[]` lists `LICENSE`; README license section mentions only LGPL-3.0 | `LICENSE:1`; `sdk/java/sage-client/pom.xml:18`; `sdk/python/pyproject.toml:11`; `sdk/rust/sage-client/Cargo.toml:7`; `contracts/ethereum/LICENSE:1`; `contracts/solana/LICENSE:1`; `sdk/typescript/package.json:8-12,36`; `ls sdk/typescript/LICENSE` -> none; `README.md:417-423`; `CONTRIBUTING.md:778-784` | [권장] | [High] |
| F28 | Versioning | Version drift across 9 sources (table in 6.3). `update-version.sh` targets README pattern `What's New in v` which does not exist (`grep` -> none) and `lib/export.go` (still 1.3.1) | `VERSION` = `1.5.2`; `pkg/version/version.go:31`; `lib/export.go:33`; `contracts/ethereum/package.json:3`; `sdk/typescript/package.json:3`; `sdk/java/sage-client/pom.xml:9`; `sdk/python/pyproject.toml:7`; `sdk/rust/sage-client/Cargo.toml:3`; `contracts/solana/Cargo.toml:8`; `tools/scripts/update-version.sh:93,147` | [권장] | [High] |
| F29 | Versioning | CHANGELOG follows Keep a Changelog header but lacks sections for 1.1.1, 1.2.0, 1.3.0, 1.3.1, 1.4.0; `[1.1.0] - 2024-10-18` has wrong year; `[Unreleased]` describes `golang:1.25.6-alpine` while Dockerfile is 1.26.3; `docs/INDEX.md` "Last Updated 2025-10-11 (v1.0.0)"; CONTRIBUTING lists a root `package.json` that does not exist | `CHANGELOG.md:5-6,8,23,30,82,178,469,578`; `docs/INDEX.md:246`; `CONTRIBUTING.md:683-689` | [권장] | [High] |
| F30 | Versioning | Commits follow Conventional Commits (`feat/fix/ci/chore(deps)`) but no tooling enforces it and tags are cut manually (`git tag v...` per script); release notes are regenerated from `git log` | `git log -15 --format=%s`; `tools/scripts/update-version.sh:196-197`; `release.yml:131` | [권장] | [High] |
| F31 | Container | `deployments/docker/*.yml` use `:latest`/`:stable` images (prometheus, grafana, swagger-ui, alpine, solana, geth) | `deployments/docker/docker-compose.yml:62,106,124,146`; `deployments/docker/test-environment.yml:30,98` | [권장] | [High] |

No open Dependabot or code-scanning alerts at audit time (`gh api .../dependabot/alerts?state=open` -> 0; `.../code-scanning/alerts?state=open` -> 0), but the latter is stale because the scanning workflow is disabled (F03).

---

## 3. Workflow inventory

### `uses:` references (all workflows)

| Workflow | Line(s) | Action | Ref | SHA-pinned |
|----------|---------|--------|-----|------------|
| docker.yml | 26 | actions/checkout | v5 | no |
| docker.yml | 34 | docker/setup-qemu-action | v3 | no |
| docker.yml | 37 | docker/setup-buildx-action | v3 | no |
| docker.yml | 41, 107 | docker/login-action | v3 | no |
| docker.yml | 49 | docker/metadata-action | v5 | no |
| docker.yml | 63 | docker/build-push-action | v6 | no |
| docker.yml | 113 | aquasecurity/trivy-action | `ed142fd0673e97e23eac54620cfb913e5ce36c25 # v0.36.0` | **yes (model)** |
| docker.yml | 130 | github/codeql-action/upload-sarif | v4 | no |
| integration-test.yml | 34, 123 | actions/checkout | v5 | no |
| integration-test.yml | 37, 135 | actions/setup-go | v6 | no |
| integration-test.yml | 43 | actions/setup-node | v6 | no |
| integration-test.yml | 101, 158 | actions/upload-artifact | v5 | no |
| integration-test.yml | 126 | docker/setup-buildx-action | v3 | no |
| loadtest.yml | 70, 194, 282 | actions/checkout | v5 | no |
| loadtest.yml | 73, 197, 285 | actions/setup-go | v6 | no |
| loadtest.yml | 122, 228, 310 | grafana/setup-k6-action | v1 | no |
| loadtest.yml | 137, 243, 325 | actions/upload-artifact | v5 | no |
| release.yml | 44, 105, 159 | actions/checkout | v5 | no |
| release.yml | 49 | actions/setup-go | v6 | no |
| release.yml | 89 | actions/upload-artifact | v5 | no |
| release.yml | 110 | actions/download-artifact | v6 | no |
| release.yml | 140 | softprops/action-gh-release | v2 | no |
| release.yml | 166, 169, 172, 180, 190 | docker/setup-qemu, setup-buildx, login, metadata, build-push | v3/v3/v3/v5/v6 | no |
| security.yml | 29, 51, 69, 171, 250, 265 | actions/checkout | v5 | no |
| security.yml | 32, 38, 41 | github/codeql-action/{init,autobuild,analyze} | v4 | no |
| security.yml | 54 | actions/dependency-review-action | v4 | no |
| security.yml | 72, 268 | actions/setup-go | v6 | no |
| security.yml | 155, 239 | github/codeql-action/upload-sarif | v4 | no |
| security.yml | 174 | actions/setup-node | v6 | no |
| test.yml | 24, 62, 97, 149, 183 | actions/checkout | v5 | no |
| test.yml | 27, 100, 152, 186 | actions/setup-go | v6 | no |
| test.yml | 42 | codecov/codecov-action | v5 | no |
| test.yml | 50, 168, 210 | actions/upload-artifact | v5 | no |
| test.yml | 65, 125 | actions/setup-node | v6 | no |
| test.yml | 106 | golangci/golangci-lint-action | v8 (`version: latest`) | no |
| test.yml | 195 | actions/download-artifact | v6 | no |

### Permissions, triggers, secrets

| Workflow | Triggers | Workflow-level `permissions` | Job-level | `pull_request_target` | Secrets |
|----------|----------|------------------------------|-----------|----------------------|---------|
| docker.yml | push main/dev/tags, PR main (`:3-9`) | none | `build-and-push`: contents:read, packages:write (`:20-22`); `security-scan`: contents:read, packages:read, security-events:write (`:88-91`) | no | `GITHUB_TOKEN` (`:45,110`) |
| integration-test.yml | push/PR main,dev, dispatch (`:3-8`) | none | none | no | none |
| loadtest.yml | push (paths), PR main, cron, dispatch (`:3-15`) | none | none | no | none |
| release.yml | push tags `v*.*.*` (`:3-6`) | `contents: write` (`:8-9`) | `docker-release`: contents:read, packages:write (`:153-155`) | no | `GITHUB_TOKEN` (`:147,176`) |
| security.yml | push/PR main,dev, weekly cron (`:3-10`) | none | codeql/gosec/slither: actions:read, contents:read, security-events:write (`:17-20,62-65,164-167`); dependency-review/gitleaks/license-check: none | no | none |
| test.yml | push/PR main,dev (`:3-7`) | none | none | no | none (Codecov tokenless, `fail_ci_if_error: false` `:47`) |

No `pull_request_target` anywhere (`grep -rn pull_request_target .github/workflows/` -> none). No repository secrets are listed (`gh api .../actions/secrets` -> empty). Expression interpolation inside `run:` uses only `github.repository`, `github.ref_name`, `github.event.head_commit.timestamp` (`docker.yml:31,73,102`; `release.yml:163,200-201`); these are attacker-controllable only by users who can already push branches/tags, so script-injection risk is low but the pattern should still be replaced by `env:` indirection.

### Steps that hide failures

| File:line | Construct | Effect |
|-----------|-----------|--------|
| security.yml:86,88,90 | `gosec ... -no-fail ... \|\| true` | gosec findings never fail the job |
| security.yml:91 | `continue-on-error: true` on gosec step | same |
| security.yml:194 | `slither ... \|\| true` | Slither findings never fail; empty SARIF fabricated at `:207-235` |
| security.yml:256 | `gitleaks:latest ... --exit-code=0 \|\| true` | secrets never fail; image unpinned |
| security.yml:279,283 | `continue-on-error: true` on both license checks | GPL/AGPL detection cannot fail CI |
| test.yml:165 | `./build/bin/sage-verify help \|\| true` | binary name mismatch (`deployment-verify`, `Makefile:7`) masked |
| Dockerfile:30 | `RUN make build-lib \|\| true` | library build failure masked in image |
| Makefile:230 | `$(MAKE) build-platform ... \|\| true` | cross-build failures masked in `make release` |
| Makefile:116-120 | `\|\| echo "Warning: ..."` | library cross-builds masked |
| release.yml:131 | `git log ... \|\| true` | release-notes generation failure masked (acceptable) |

### Caching of untrusted content

- `actions/setup-go` `cache: true` keys on `go.sum`; `setup-node` `cache: npm` keys on
  `contracts/ethereum/package-lock.json`; Docker `cache-from/to: type=gha` (`docker.yml:69-70`,
  `release.yml:197-198`). GitHub scopes PR caches to the PR branch and forbids fork PRs from writing
  the base-branch cache, so cross-contamination is bounded. **Residual risk**: the `docker-release`
  job restores the `gha` cache written by `dev`/`main` builds into release images; a compromised
  `dev` push could poison layers reused by a release build. `[Mid]`

### Toolchain versions

- CI: Go `1.25.2` (`test.yml:10,20`, `integration-test.yml:11`, `release.yml:12`, `security.yml:74,270`);
  matrix has a single entry. Node `22` (`test.yml:67,127`; `integration-test.yml:45`; `security.yml:176`),
  matches `contracts/ethereum/package.json:86` `"node": ">=22.0.0"`.
- `go.mod:3` `go 1.25.2`; Dockerfile builder Go `1.26.3` (`Dockerfile:5`); `GOTOOLCHAIN: auto`
  everywhere (`docker.yml:14`, `test.yml:11`, `Makefile:21`) so a bumped `go` directive would silently
  download a new toolchain on the fly in CI.

---

## 4. Container and build inputs

| Item | Current value | Evidence | Assessment |
|------|---------------|----------|------------|
| Builder base | `golang:1.26.3-alpine` (tag only) | `Dockerfile:5` | pin by digest; Dependabot then tracks digest |
| Runtime base | `alpine:latest` | `Dockerfile:33` | mutable; pin `alpine:3.x@sha256:...` |
| `apk upgrade` | present | `Dockerfile:38` | good; keep after digest pin |
| Non-root user | `sage` uid 1000 | `Dockerfile:44-45,69` | good |
| Build flags | `GOFLAGS=-v`, `LDFLAGS=-w -s`, `-X` version vars | `Makefile:19-20,31-39` | no `-trimpath`, no `-buildvcs=false`, `CGO_ENABLED` unset for binaries |
| CGO | gcc/musl-dev installed in builder; main binaries built without `CGO_ENABLED=0` | `Dockerfile:8-12`; `Makefile:258,269,280` | binaries may dynamically link musl; static libs need CGO by design (`Makefile:129-210`) |
| Reproducibility | `BUILD_DATE` to seconds, `GIT_BRANCH` embedded | `Makefile:26-27,33-34` | not reproducible; use `SOURCE_DATE_EPOCH`/commit time and drop branch |
| `make release` | `clean build-all-platforms build-lib-all package checksums` | `Makefile:824` | packages `sage-$os-$arch.tar.gz` without version in name (`:803`), unlike `release.yml:72,74` which embeds the tag; two divergent packaging paths |
| Signing | none (no cosign/gpg/sigstore anywhere) | `grep -rn cosign\|sigstore\|slsa .github Makefile` -> no matches | fail |
| SBOM / provenance | none (no syft/cyclonedx/`attest-build-provenance`) | same grep | fail |
| Docker image scan | Trivy `vuln,config,secret`, `skip-dirs: /usr/local/bin` | `docker.yml:113-127` | Go binaries excluded |
| Image labels/attestation | metadata-action labels only; no `provenance: true`/`sbom: true` on build-push | `docker.yml:61-73`; `release.yml:189-201` | add |

---

## 5. Dependency hygiene

### Manifests, lockfiles, Dependabot coverage

| Directory | Ecosystem | Manifest | Lockfile committed | Dependabot |
|-----------|-----------|----------|--------------------|------------|
| `/` | gomod | `go.mod` (19 direct deps, no `replace`) | `go.sum` yes | yes (`dependabot.yml:7`) |
| `/tools/codegraph` | gomod | `go.mod` (`golang.org/x/tools v0.38.0`) | `go.sum` yes | **no** |
| `/` | github-actions | 6 workflows | n/a | yes (`:24`) |
| `/contracts/ethereum` | npm | `package.json` (hardhat `^3.1.10`, 10 `overrides`) | `package-lock.json` yes | yes (`:41`) |
| `/sdk/typescript` | npm | `package.json` (`@sage-x/sdk` 1.0.0) | **none** | **no** |
| `/sdk/python` | pip | `pyproject.toml` + `requirements.txt` (`>=` ranges) | **none** | **no** |
| `/sdk/java/sage-client` | maven | `pom.xml` (bouncycastle 1.84) | n/a (versions fixed in pom) | **no** |
| `/sdk/rust/sage-client` | cargo | `Cargo.toml` | `Cargo.lock` **git-ignored** (`.gitignore:456`) | **no** |
| `/contracts/solana` | cargo | workspace `Cargo.toml`, 2 programs (anchor-lang 0.29.0) | `Cargo.lock` git-ignored | **no** |
| `/` | docker | `Dockerfile` | n/a | yes (`:59`) but no digest to track |

Dependabot config details: weekly Monday 00:00 for all (`:9-12,26-29,43-46,60-63`); PR limits 10/5/10/5;
no `groups:`; `reviewers: sage-x-project/maintainers` (nonexistent team, 404); commit prefixes
`chore(deps)`, `chore(ci)`, `chore(docker)` (`:19-21,36-38,53-55,71-73`). Actual merged PRs show
`chore(deps)(deps-dev): ...` (`git log`), i.e. the `include: scope` option double-wraps the scope.

Security-updates: `dependabot_security_updates: enabled`, `automated-security-fixes: {enabled:true, paused:false}` (gh output). Dependency graph workflow active.

---

## 6. Repository governance

### Settings (gh output)

| Setting | Value |
|---------|-------|
| `default_branch` / visibility | `main` / public |
| Branch protection (`/branches/main/protection`) | 404 "Branch not protected" |
| Ruleset 8640770 `main` | enforcement `active`, **`include: []`**, rules: deletion, non_fast_forward, pull_request (0 approvals, dismiss stale, no code-owner review, merge methods squash+rebase), `bypass_actors: []` |
| Effective rules on `main` (`/rules/branches/main`) | `[]` |
| `dev` protected | false |
| Direct collaborators | 2 admin (`onlyhyde`, `0xmhha`), 4 write |
| Team | `sage-core-dev` -> `push` |
| Merge methods | squash, merge, rebase all allowed; `delete_branch_on_merge: false`; `allow_auto_merge: false` |
| `web_commit_signoff_required` | true |
| Actions | `allowed_actions: all`, `sha_pinning_required: false`, `default_workflow_permissions: read`, `can_approve_pull_request_reviews: false` |
| Secret scanning / push protection | disabled / disabled |
| Dependabot alerts / security updates | enabled / enabled |
| CodeQL default setup | not-configured (advanced workflow disabled, F03) |
| Recent `main` commits via PR | `14283ce`->#218, `e98b42b`/`67f7ba7`->#217, `a7ed062`->#215 (`gh api commits/{sha}/pulls`), merged with 0 required approvals |
| Activity | 32 commits to `main` in last 90 days; last push 2026-09-11 |

### Files present / absent

| File | Status |
|------|--------|
| `LICENSE` (LGPL-3.0), `NOTICE` | present (`LICENSE:1`, `NOTICE:1-15`) |
| `CONTRIBUTING.md` | present (release process `:598-760`, branch rules `:168-179`) |
| `SECURITY.md` | **absent** (root, `.github/`, `docs/`) |
| `CODEOWNERS` | **absent** |
| `CODE_OF_CONDUCT.md` | **absent** (inline section `CONTRIBUTING.md:18-32`) |
| Issue templates | 4 forms + `config.yml` (blank issues disabled) |
| PR template | `.github/PULL_REQUEST_TEMPLATE.md` |
| `.editorconfig`, pre-commit / husky / lefthook | **absent** |
| `.golangci.yml`, `.gosec.json`, `codecov.yml`, `.gitleaksignore` | present |

### Version sources

| Source | Value | Evidence |
|--------|-------|----------|
| `VERSION` | 1.5.2 | file |
| `pkg/version/version.go` | 1.5.2 | `:31` |
| `lib/export.go` (`SageVersion`) | **1.3.1** | `:33` |
| `contracts/ethereum/package.json` | **1.5.0** | `:3` |
| `sdk/typescript/package.json` | 1.0.0 | `:3` |
| `sdk/java/sage-client/pom.xml` | 0.1.0 | `:9` |
| `sdk/python/pyproject.toml` | 0.1.0 | `:7` |
| `sdk/rust/sage-client/Cargo.toml` | 0.1.0 | `:3` |
| `contracts/solana/Cargo.toml` (+2 programs) | 0.1.0 | `:8` |
| Latest git tag / GitHub release | v1.5.2 (2025-11-02), 42 commits ahead on `main` | `git rev-list v1.5.2..main --count` |
| CHANGELOG top released section | `[1.5.2] - 2025-11-02` | `CHANGELOG.md:30` |

Tags v1.0.0..v1.5.2 (13) are all annotated (`type=tag`) by `0xTopaz`, none signed. Release
workflow history: v1.5.1 run failed then v1.5.2 succeeded (`gh run list --workflow=release.yml`).

---

## 7. OpenSSF Scorecard-style checklist

| Check | Result | Evidence | Fix |
|-------|--------|----------|-----|
| Pinned-Dependencies | **FAIL** | F05, F10, F11, F12 | SHA-pin all `uses:`; pin `go install`/`pip`/docker tags; `npm ci --ignore-scripts`; digest-pin images |
| Token-Permissions | PARTIAL | F21, F22; repo default `read` | add `permissions: {contents: read}` at top of every workflow; move `contents: write` to `create-release` job only |
| Branch-Protection | **FAIL** | F01 | fix ruleset `include`, add required checks + 1 approval + linear history |
| Signed-Releases | **FAIL** | F06, F08 | cosign keyless + SLSA provenance; signed tags |
| SAST | **FAIL (currently)** | F03; CodeQL config exists but workflow disabled | re-enable workflow (`gh workflow enable security.yml`), or switch CodeQL to default setup; make gosec/Slither blocking |
| Dependency-Update-Tool | PASS (partial) | dependabot.yml present; F16 gaps | add 6 missing entries + groups |
| Vulnerabilities | PASS | 0 open Dependabot alerts | keep; add `govulncheck` for binaries |
| Dangerous-Workflow | PASS | no `pull_request_target`; no untrusted `${{ }}` in `run:` from PR context | move `github.ref_name` into `env:` |
| Maintained | PASS | 32 commits / 90 days | - |
| License | PASS (with F27 caveat) | LGPL-3.0 detected by GitHub | add `sdk/typescript/LICENSE`; document dual scheme in README |
| Security-Policy | **FAIL** | F25 | add `SECURITY.md` |
| CI-Tests | PASS (not enforced) | `test.yml` on PRs; no required checks | make `Go Tests`, `Smart Contract Tests`, `Lint`, `Build` required |
| Code-Review | **FAIL** | 0 required approvals; PRs #217/#218 merged without reviewer requirement | `required_approving_review_count: 1`, CODEOWNERS |
| Packaging | PASS (partial) | GHCR image + release archives | add SBOM + provenance + signatures |
| Fuzzing | PASS (partial) | 12 `func Fuzz*` in `pkg/agent/crypto/fuzz_test.go`, `pkg/agent/session/fuzz_test.go`; `cmd/random-test` | schedule `go test -fuzz` job or OSS-Fuzz |

---

## 8. Remediation plan

Cost is engineering time only; all tools below are free for public repos.

### P0 (this week; restores integrity guarantees)

**P0-1 Fix the ruleset so it actually applies (F01, F02, F26).** `PUT /repos/SAGE-X-project/sage/rulesets/8640770`:

```json
{
  "name": "main", "target": "branch", "enforcement": "active",
  "conditions": { "ref_name": { "include": ["~DEFAULT_BRANCH"], "exclude": [] } },
  "bypass_actors": [],
  "rules": [
    { "type": "deletion" },
    { "type": "non_fast_forward" },
    { "type": "required_linear_history" },
    { "type": "pull_request", "parameters": {
        "required_approving_review_count": 1,
        "dismiss_stale_reviews_on_push": true,
        "require_code_owner_review": true,
        "require_last_push_approval": true,
        "required_review_thread_resolution": true,
        "allowed_merge_methods": ["squash", "rebase"] } },
    { "type": "required_status_checks", "parameters": {
        "strict_required_status_checks_policy": true,
        "required_status_checks": [
          { "context": "Go Tests" }, { "context": "Smart Contract Tests" },
          { "context": "Lint" }, { "context": "Build (ubuntu-latest)" },
          { "context": "CodeQL Analysis (go)" }, { "context": "Go Security Scan" } ] } }
  ]
}
```

Add a second ruleset for tags: `target: "tag"`, `include: ["refs/tags/v*"]`, rules `creation`
restricted to admins via `bypass_actors` (RepositoryRole admin, `bypass_mode: always`) plus
`update`/`deletion`. Cost: 1 h. Trade-off: with only 2 admins, a 1-approval rule can stall
solo work; use `bypass_actors` for the admin role rather than lowering the count. Update
`CONTRIBUTING.md:168-179` to match.

**P0-2 Re-enable security scanning (F03).** `gh workflow enable security.yml`; add
`workflow_dispatch:` to `security.yml:3` so inactivity cannot silently disable it again, and add a
weekly `keepalive` (or use `actions/keepalive`-style empty commit) or move CodeQL to default setup
(`PATCH /code-scanning/default-setup {"state":"configured","languages":["go","javascript"]}`).
Cost: 0.5 h.

**P0-3 Enable secret scanning + push protection (F04).**
`gh api -X PATCH repos/SAGE-X-project/sage -f 'security_and_analysis[secret_scanning][status]=enabled' -f 'security_and_analysis[secret_scanning_push_protection][status]=enabled'`. Cost: 5 min.

**P0-4 Add `SECURITY.md` (F25)** with private reporting via GitHub Security Advisories, supported
versions (1.5.x), 90-day disclosure window. Enable "Private vulnerability reporting" in repo
settings. Cost: 1 h.

**P0-5 Explicit least-privilege `permissions:` (F21, F22).** Add to the top of `test.yml`,
`integration-test.yml`, `loadtest.yml`, `security.yml`, `docker.yml`:

```yaml
permissions:
  contents: read
```

and in `release.yml` replace lines 8-9 with `permissions: {}` at workflow level, then per job:

```yaml
  build-release:
    permissions: { contents: read }
  create-release:
    permissions: { contents: write, id-token: write, attestations: write }
  docker-release:
    permissions: { contents: read, packages: write, id-token: write, attestations: write }
```

`security.yml` jobs `dependency-review`, `gitleaks`, `license-check`: `permissions: { contents: read }`
(`dependency-review` additionally `pull-requests: write` if `comment-summary-in-pr` is used). Cost: 1 h.

### P1 (this month; hardens inputs and outputs)

**P1-1 SHA-pin every action (F05).** Follow the `docker.yml:113` model
(`owner/repo@<40-hex> # vX.Y.Z`). Do not copy SHAs from memory; resolve each with
`gh api repos/<owner>/<repo>/git/ref/tags/<tag> --jq .object.sha` (dereference annotated tags via
`git/tags/<sha>`), or run `pinact run` / `ratchet pin .github/workflows/*.yml`. Dependabot
`github-actions` already keeps SHA pins current and preserves the version comment. Then set
`gh api -X PUT repos/SAGE-X-project/sage/actions/permissions -f enabled=true -f allowed_actions=selected`
with `github_owned_allowed=true`, `verified_allowed=true`, and `sha_pinning_required=true`.
Cost: 2 h. Trade-off: Dependabot PR volume rises (one PR per action bump); mitigate with a
`groups: { actions: { patterns: ["*"] } }` block.

**P1-2 Pin run-time installs and make scanners blocking (F09, F10, F11).**

- `security.yml:83` -> `go install github.com/securego/gosec/v2/cmd/gosec@v2.<pinned>`; remove
  `-no-fail`, `|| true` and `continue-on-error` (`:86-91`); keep `.gosec.json` excludes as the
  suppression mechanism.
- `security.yml:189` -> `pip3 install slither-analyzer==0.<pinned>` and drop `|| true` at `:194`;
  configure `.slither.config.json` `fail-on` to `high`.
- `security.yml:256` -> `uses: gitleaks/gitleaks-action@<sha> # v2.x` with `GITLEAKS_ENABLE_SUMMARY`,
  drop `--no-git`/`--exit-code=0`; this scans history and fails on hits.
- `security.yml:276` -> `go install github.com/google/go-licenses/v2@v2.<pinned>`; move
  `license-checker` into `contracts/ethereum/package.json` devDependencies; remove
  `continue-on-error` at `:279,283`.
- `test.yml:108` `version: latest` -> `version: v2.<pinned>`.
- Every `npm ci` (`test.yml:73,133`, `integration-test.yml:54`, `security.yml:180`) ->
  `npm ci --ignore-scripts` (Hardhat does not need lifecycle scripts; verify `npx hardhat compile`
  still passes).
- Every `go mod download` -> `go mod download && go mod verify`; add `GOFLAGS: -mod=readonly` to
  workflow `env:`; add a `govulncheck` step (`golang/govulncheck-action@<sha>`) to `security.yml`
  and `docker.yml` (binary mode against `build/bin/*`) to close the Trivy `skip-dirs` gap (F15).
Cost: 3 h. Trade-off: first blocking run will surface existing gosec/Slither findings; budget a
triage PR (`.gosec.json`, `.slither.config.json` `detectors_to_exclude`) before flipping to blocking.

**P1-3 Signed, attested, reproducible releases (F06, F07, F08).** Replace the hand-rolled
`build-release`/`create-release` jobs (`release.yml:17-147`) with GoReleaser + cosign keyless + SLSA:

```yaml
jobs:
  goreleaser:
    runs-on: ubuntu-latest
    permissions: { contents: write, id-token: write, attestations: write, packages: write }
    steps:
      - uses: actions/checkout@<sha> # v5
        with: { fetch-depth: 0, persist-credentials: false }
      - uses: actions/setup-go@<sha> # v6
        with: { go-version-file: go.mod }
      - uses: sigstore/cosign-installer@<sha> # v3
      - uses: anchore/sbom-action/download-syft@<sha> # v0
      - uses: goreleaser/goreleaser-action@<sha> # v6
        with: { version: "~> v2", args: release --clean }
        env: { GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}" }
      - uses: actions/attest-build-provenance@<sha> # v2
        with: { subject-path: "dist/*.tar.gz,dist/*.zip,dist/checksums.txt" }
```

`.goreleaser.yaml` essentials: `builds[].env: [CGO_ENABLED=0]`, `flags: [-trimpath, -buildvcs=false]`,
`mod_timestamp: '{{ .CommitTimestamp }}'`, `ldflags: -s -w -X github.com/sage-x-project/sage/pkg/version.Version={{.Version}} -X ...GitCommit={{.Commit}} -X ...BuildDate={{.CommitDate}}` (drop `GitBranch`),
`sboms: [{artifacts: archive}]`, `signs: [{cmd: cosign, certificate: '${artifact}.pem', args: [sign-blob, --yes, --output-certificate, '${certificate}', --output-signature, '${signature}', '${artifact}'], artifacts: checksum}]`,
`docker_signs` for GHCR, `changelog.use: github` with conventional-commit groups. Cross-compiling
all five targets from one Linux runner also removes the per-OS runner divergence. For the C
libraries (`Makefile:129-210`, need CGO) keep `make build-lib-all` as a separate optional job.
Cost: 1 day. Trade-off: GoReleaser replaces `Makefile` `package`/`checksums`/`release` targets
(`:797-835`); keep them only for local builds or delete to avoid two packaging paths.

Also: sign tags (`git tag -s`), add rule `required_signatures` to the ruleset once maintainers
have GPG/SSH signing keys registered. Cost: 1 h.

**P1-4 Container pinning (F12, F13, F14).**

```dockerfile
FROM golang:1.25.2-alpine@sha256:<digest> AS builder   # match go.mod; resolve with `docker buildx imagetools inspect`
...
RUN CGO_ENABLED=0 make build-binaries GOFLAGS="-trimpath -buildvcs=false"
RUN make build-lib            # fail loudly, or drop from the image entirely
FROM alpine:3.22@sha256:<digest>
```

Decide one Go toolchain: either raise `go.mod` to `go 1.26` everywhere (unblocks `x/crypto` 0.56+)
or hold Docker at 1.25.x; set `GOTOOLCHAIN=local` in CI so drift fails instead of auto-downloading.
Enable `provenance: true` and `sbom: true` on `docker/build-push-action` (`docker.yml:63`,
`release.yml:190`). Cost: 2 h.

**P1-5 Dependabot coverage and grouping (F16, F17).** Append to `.github/dependabot.yml`:

```yaml
  - package-ecosystem: "gomod"
    directory: "/tools/codegraph"
    schedule: { interval: "weekly" }
  - package-ecosystem: "npm"
    directory: "/sdk/typescript"
    schedule: { interval: "weekly" }
  - package-ecosystem: "pip"
    directory: "/sdk/python"
    schedule: { interval: "weekly" }
  - package-ecosystem: "maven"
    directory: "/sdk/java/sage-client"
    schedule: { interval: "weekly" }
  - package-ecosystem: "cargo"
    directory: "/sdk/rust/sage-client"
    schedule: { interval: "weekly" }
  - package-ecosystem: "cargo"
    directory: "/contracts/solana"
    schedule: { interval: "weekly" }
```

Add `groups:` to the existing four entries (e.g. `go-minor-patch: {update-types: [minor, patch]}`,
`hardhat: {patterns: ["@nomicfoundation/*", "hardhat*"]}`, `actions: {patterns: ["*"]}`), fix
`reviewers` to `sage-x-project/sage-core-dev` or remove it. Commit lockfiles: `sdk/typescript/package-lock.json`
(`npm i --package-lock-only`), remove `Cargo.lock` from `.gitignore:456` and commit locks for
`sdk/rust` and `contracts/solana`, add `uv.lock` or `requirements.lock` for `sdk/python`. Delete the
stray root `package-lock.json`. Cost: 2 h. Trade-off: ~6 more weekly PR streams; groups keep it to
roughly one PR per ecosystem per week.

### P2 (next quarter; hygiene and automation)

- **P2-1** Delete or repair `loadtest.yml` (F19): move `inputs:` under `on.workflow_dispatch`, remove
  the `if: false` jobs or point them at a real server target; every push currently produces a red run.
  Remove the dead e2e job (F20). Cost: 1 h.
- **P2-2** `CODEOWNERS` (`* @SAGE-X-project/sage-core-dev`, `/contracts/ @onlyhyde`, `/.github/ @0xmhha`),
  `CODE_OF_CONDUCT.md`, `.editorconfig`, `.pre-commit-config.yaml` (gofmt, golangci-lint,
  gitleaks, conventional-commit msg hook). Cost: 2 h.
- **P2-3** Release automation: adopt **release-please** (`googleapis/release-please-action`) in
  `manifest` mode with `.release-please-manifest.json` listing each versioned path, so tags,
  `CHANGELOG.md`, `VERSION`, `pkg/version/version.go`, `lib/export.go` and `package.json` files are
  bumped by one bot PR from Conventional Commits (`extra-files` per package). Retire
  `tools/scripts/update-version.sh` (its README step is a no-op, F28). Enforce commit format with
  `amannn/action-semantic-pull-request` on PR titles (squash-merge uses the title). Cost: 1 day.
  Trade-off: release-please expects squash merges with conventional titles; the ruleset already
  limits merge methods to squash/rebase, so disable rebase to keep history parseable.
- **P2-4** License consistency (F27): add `sdk/typescript/LICENSE` (choose MIT or LGPL-3.0 and align
  `package.json:36`), add a "Licensing" subsection to `README.md:417` stating Go/SDK = LGPL-3.0,
  contracts = MIT, and list the TypeScript decision. Cost: 1 h.
- **P2-5** Backfill missing CHANGELOG sections 1.1.1-1.4.0 from tag messages, fix `2024-10-18`,
  update `docs/INDEX.md:246`. Cost: 2 h.
- **P2-6** Pin `deployments/docker/*.yml` images by tag+digest (F31). Cost: 1 h.
- **P2-7** Run `scorecard` in CI (`ossf/scorecard-action`, weekly, `security-events: write`) and add
  the badge; it will regress-test everything above. Cost: 0.5 h.

---

## 9. Proposed version-management policy (multi-repository)

The repository currently ships one Go module, one C ABI, two contract suites, four SDKs and a
Docker image from a single `VERSION` file that only the Go side honours (section 6.3). A single
number cannot express that `AgentCardRegistry` v1.5.0 on Sepolia is compatible with Go 1.5.2 and
TypeScript 1.0.0. The proposal below is per-artifact SemVer with an explicit compatibility matrix.

### Rules

1. **One SemVer line per publishable artifact; independent cadence.** Artifacts: `sage` Go module,
   `libsage` C ABI, `sage-contracts` (EVM), `sage-solana`, `@sage-x/sdk` (TS), `sage-client` (Py),
   `com.sage:sage-client` (Java), `sage-client` (Rust), `ghcr.io/sage-x-project/sage` image.
2. **Go module tags are the repo's `vX.Y.Z` tags** (Go tooling requires plain `vX.Y.Z` at module
   root). Sub-artifact tags are namespaced: `contracts/ethereum/v1.6.0`, `sdk/typescript/v1.1.0`,
   `sdk/python/v0.2.0`, `sdk/java/v0.2.0`, `sdk/rust/v0.2.0`, `contracts/solana/v0.2.0`. If a
   sub-artifact moves to its own repository, it keeps the same numeric line and drops the prefix.
   `tools/codegraph` is a nested module; tag `tools/codegraph/vX.Y.Z` only if published.
3. **Go module major bumps** follow Go convention: `v2+` requires `/v2` module path
   (`github.com/sage-x-project/sage/v2`) and the release-please `major` type triggered by
   `BREAKING CHANGE:` footers or `feat!:`.
4. **Contract versioning is two-dimensional**: (a) package SemVer (`contracts/ethereum/package.json`)
   for the Hardhat project, and (b) an on-chain `ABI_VERSION`/`VERSION()` constant per contract plus
   an `abi/` directory committed per release (`contracts/ethereum/abi/<Contract>.v<major>.json`).
   ABI major changes when a function/event signature is removed or changed; minor when added.
   Deployed addresses are recorded per network in `contracts/ethereum/deployments/<network>-<version>.json`
   (today only `hardhat-latest.json` exists). Go bindings regenerated from a given ABI carry that ABI
   version in the package doc comment.
5. **SDK versions track the protocol, not the Go module.** Define `PROTOCOL_VERSION` (handshake +
   RFC 9421 profile + HPKE suite) as a small integer or `major.minor` in `docs/PROTOCOL.md`; each SDK
   declares which protocol versions it speaks. SDK SemVer major bumps only on SDK API breaks.
6. **Docker image tags** = Go module version (`1.5.2`, `1.5`, `1`, `latest`, `sha-<7>`) plus digest
   published in release notes; `latest` only from the default branch (already so, `docker.yml:59`).
7. **Compatibility matrix** is a versioned file `docs/COMPATIBILITY.md` generated by release-please
   `extra-files` and validated in CI (a Go test that parses it and asserts the running module version
   is listed). Row = Go module release; columns = min/max supported versions of each other artifact.
8. **Support window**: latest minor receives fixes; previous minor receives security fixes for 6
   months; contract ABIs are supported for as long as the deployment is live (they cannot be
   patched, only superseded). Record this in `SECURITY.md`.
9. **Single source of truth per artifact**: Go = `pkg/version/version.go` (ldflags override), TS =
   `package.json`, Py = `pyproject.toml`, Java = `pom.xml`, Rust = `Cargo.toml`, contracts =
   `package.json` + on-chain constant. Delete `VERSION` and the `lib/export.go` literal, or derive
   both from `pkg/version` at build time. `CHANGELOG.md` is split per artifact
   (`CHANGELOG.md` for Go, `contracts/ethereum/CHANGELOG.md`, `sdk/<lang>/CHANGELOG.md`) and each is
   Keep-a-Changelog formatted with release-please managing headings.

### Starting matrix (values observed today; compatibility columns are **[not verified]** and must be filled by the maintainers)

| Go module | libsage ABI | sage-contracts | Solana | TS SDK | Py SDK | Java SDK | Rust SDK | Image |
|-----------|-------------|----------------|--------|--------|--------|----------|----------|-------|
| v1.5.2 | 1.3.1 (stale) | 1.5.0 | 0.1.0 | 1.0.0 | 0.1.0 | 0.1.0 | 0.1.0 | 1.5.2 |

### Multi-repo split guidance

If SDKs or contracts move to separate repositories: keep the tag namespace rule (rule 2) so
history migrates cleanly; publish the compatibility matrix from the core repo and have each SDK
repo's CI fetch and assert against it; use `release-please` per repo with the same commit
convention; sign all tags and attest all packages (cosign for npm/PyPI/Maven Central via sigstore,
GitHub Attestations for archives) so consumers verify one provenance format across repos.

---

## Appendix A. Not verified

- Whether the ruleset's empty `include` was intentional (e.g. created via UI without selecting a
  target) - only the effect is verified.
- Whether GitHub disabled `security.yml` because of the 60-day inactivity rule or manually; only the
  `disabled_inactivity` state string is verified.
- Whether `npm ci --ignore-scripts` breaks any Hardhat plugin post-install steps - untested.
- Compatibility columns in section 9 matrix.
- Exact SHAs for action pins are intentionally omitted; resolve at pin time.

## Appendix B. Commands used

`gh api repos/SAGE-X-project/sage`, `.../branches/main`, `.../branches/main/protection`,
`.../rulesets`, `.../rulesets/8640770`, `.../rules/branches/main`, `.../collaborators?affiliation=all`,
`.../teams`, `orgs/SAGE-X-project/teams`, `.../actions/permissions`, `.../actions/permissions/workflow`,
`.../actions/workflows`, `.../actions/secrets`, `.../automated-security-fixes`,
`.../code-scanning/default-setup`, `.../dependabot/alerts?state=open`, `.../code-scanning/alerts?state=open`,
`.../secret-scanning/alerts`, `.../commits/<sha>/pulls`; `gh run list [--workflow=...]`, `gh run view`,
`gh release list`, `gh release view v1.5.2`; `git for-each-ref refs/tags`, `git verify-tag v1.5.2`,
`git log --format='%h %G? %s'`, `git merge-base --is-ancestor v1.5.2 main`, `git rev-list v1.5.2..main --count`;
`cat -n` / `grep -n` / `find` / `ls` over the paths cited above.
