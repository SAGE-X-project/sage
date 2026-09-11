# Security Policy

## Supported versions

| Version | Supported |
|---|---|
| 1.5.x (latest minor) | Yes |
| < 1.5 | No |

## Reporting a vulnerability

Please do not open a public issue for security problems.

Use GitHub's private vulnerability reporting for this repository:
https://github.com/SAGE-X-project/sage/security/advisories/new

Include the affected component (Go package, CLI, contract, workflow), a reproduction, and the impact you see. You will receive an acknowledgement within 5 business days. We aim to publish a fix and an advisory within 90 days of the report; if a fix takes longer we will tell you why and agree a disclosure date with you.

## Scope

- Go library and CLIs in this repository (`pkg/`, `cmd/`)
- Smart contracts in `contracts/`
- CI workflows and release artifacts

The multi-language SDKs under `sdk/` are experimental and not covered by this policy until they are released separately.

## Handling

Reports are triaged by the maintainers, fixed on a private branch, released as a patch version, and published as a GitHub Security Advisory with a CVE where applicable. Credit is given to reporters unless they ask otherwise.

## Verifying releases

Release binaries are built by GoReleaser in GitHub Actions from the tagged
commit with `CGO_ENABLED=0` and `-trimpath`, so a rebuild of the same tag
yields identical binaries. Each release carries:

- `SHA256SUMS` listing every archive, signed keyless with Sigstore
  (`SHA256SUMS.sig`, `SHA256SUMS.pem`; identity = this repository's
  `release.yml` workflow at the release tag);
- SLSA build provenance for every archive and for the container image
  (GitHub artifact attestations);
- a CycloneDX SBOM per archive (`<archive>.cdx.json`).

```bash
# Signature of the checksum file
cosign verify-blob SHA256SUMS \
  --signature SHA256SUMS.sig --certificate SHA256SUMS.pem \
  --certificate-identity-regexp '^https://github.com/SAGE-X-project/sage/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com

# Archive checksums
sha256sum --ignore-missing -c SHA256SUMS

# Build provenance
gh attestation verify sage-vX.Y.Z-linux-amd64.tar.gz --repo SAGE-X-project/sage

# Container image signature and provenance
cosign verify ghcr.io/sage-x-project/sage:vX.Y.Z \
  --certificate-identity-regexp '^https://github.com/SAGE-X-project/sage/\.github/workflows/release\.yml@refs/tags/v' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com
gh attestation verify oci://ghcr.io/sage-x-project/sage:vX.Y.Z --repo SAGE-X-project/sage
```
