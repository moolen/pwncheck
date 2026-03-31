# pwncheck

`pwncheck` verifies semver-like GHCR image tags against a stored baseline of image manifest digests and signed provenance.

For each configured repository, the verifier:

- lists semver-like tags from GHCR
- resolves each tag to the current manifest digest
- finds the attached `slsaprovenance` attestation
- verifies the attestation with `cosign`
- enforces an exact GitHub Actions identity pin
- compares the result with a baseline JSON stored as a GitHub release asset

Bootstrap is automatic. A missing baseline creates the initial state. Newly published semver tags are verified and added to the baseline without failing the run. Existing tags fail the run if the image digest changes, the provenance digest changes, the provenance goes missing, or the provenance no longer verifies against the pinned workflow identity.

## Configuration

The application reads `config.yaml`.

Example:

```yaml
release:
  owner: moolen
  repo: pwncheck
  tag: pwncheck-state
  name: Pwncheck State

repositories:
  - name: external-secrets
    package: ghcr.io/external-secrets/external-secrets
    policy:
      issuer: https://token.actions.githubusercontent.com
      repository: external-secrets/external-secrets
      workflowPath: .github/workflows/release.yml
      ref: refs/heads/main
```

## Running

`pwncheck` requires `GITHUB_TOKEN` for GitHub Releases access.

```bash
go test ./...
GITHUB_TOKEN=... go run ./cmd/pwncheck verify --config config.yaml
```

## GitHub Actions

The repository includes `.github/workflows/verify.yaml` to run on a schedule and on manual dispatch. The workflow:

- runs the Go test suite
- verifies configured repositories
- updates the `pwncheck-state` release assets when bootstrap or new tags are observed

One JSON asset is stored per configured repository.
