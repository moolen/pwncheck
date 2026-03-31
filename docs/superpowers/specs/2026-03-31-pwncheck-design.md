# Pwncheck Design

## Goal

Build a Go application that monitors configured GHCR image repositories, records the digest and provenance for semver-like tags, and alerts when an existing tag's image or provenance changes. The baseline state is stored as GitHub release assets so scheduled GitHub Actions runs can compare current registry state against previously verified state.

## Scope

- Verify only semver-like tags, not mutable helper tags like `latest`
- Bootstrap baseline state automatically on first run
- Accept newly published semver tags without alerting, as long as they verify successfully
- Alert when an existing semver tag changes digest, changes provenance, loses provenance, or fails provenance policy checks
- Persist one JSON baseline file per configured repository as a release asset on a dedicated GitHub release

## Configuration

The application is configured by `config.yaml`.

The config contains:

- GitHub release settings for the baseline storage release
- A list of repositories to verify
- For each repository:
  - Logical name used for logs and the release asset filename
  - OCI package reference, for example `ghcr.io/external-secrets/external-secrets`
  - Semver matching policy
  - Pinned provenance identity policy

The pinned provenance policy must be repository specific and as strict as possible. The initial policy shape is:

- OIDC issuer pinned to GitHub Actions
- Source repository pinned to the expected owner and repository
- Workflow path pinned to the exact workflow file that produces the image
- Git ref pinned to `refs/heads/main`
- Optional support for reusable workflow caller identity if the project uses one

## Architecture

The application is split into focused packages:

- `cmd/pwncheck`: CLI entrypoint
- `internal/config`: config loading and validation
- `internal/registry`: GHCR tag enumeration and tag-to-digest resolution
- `internal/provenance`: provenance discovery, cryptographic verification, and policy evaluation
- `internal/state`: baseline JSON schema and serialization
- `internal/github`: GitHub release lookup, creation, asset download, asset upload, and asset replacement
- `internal/verify`: orchestration and drift detection

The main command is `verify`.

## Runtime Flow

1. Load `config.yaml` and authenticate using `GITHUB_TOKEN`.
2. Ensure the configured GitHub release exists. If it does not, create it.
3. For each configured repository, load its existing baseline JSON asset if present.
4. Query GHCR for tags and keep only semver-like tags.
5. For each semver tag:
   - Resolve the current image manifest digest.
   - Discover the attached provenance for that digest.
   - Verify provenance cryptographically.
   - Verify that the provenance subject matches the resolved image digest.
   - Verify that the certificate identity and workflow binding match the pinned policy.
6. Compare observed records against the loaded baseline.
7. If the baseline has no entry for a tag, add it without failing the run.
8. If the baseline has an entry and any tracked field differs, record a failure.
9. Upload the updated baseline asset if bootstrap occurred or new tags were added.
10. Exit non-zero if any existing tag drift or provenance validation failure was detected.

## Baseline Storage

Baseline state is stored in GitHub Releases rather than in the git repository.

Release model:

- A dedicated release tag such as `pwncheck-state`
- One JSON asset per configured repository

JSON model per repository:

- Schema version
- Repository metadata
- Last updated time
- Tag records keyed by semver tag

Each tag record stores:

- Tag
- Image manifest digest
- Provenance artifact digest
- Verified provenance summary
- Verification timestamp

The verified provenance summary stores enough identity detail to explain what was accepted, for example workflow path, repository, ref, issuer, and certificate subject.

## Drift Rules

The run must fail when any of the following happen for an already-tracked semver tag:

- The tag now resolves to a different image digest
- The provenance artifact digest changed
- Provenance is missing
- Provenance cryptographic verification fails
- Provenance subject no longer matches the image digest
- Provenance identity no longer matches the pinned GitHub Actions policy

The run must not fail when:

- No baseline exists yet and bootstrap succeeds
- A new semver tag appears and its image plus provenance verify successfully

## Provenance Verification

The verifier should use Sigstore-compatible verification and in-toto/SLSA provenance parsing.

Acceptance requirements:

- The attestation must be valid for the resolved digest being checked
- The signer certificate chain must validate
- The identity must match the configured GitHub Actions policy
- The attestation must bind to the expected repository and workflow

The code should keep the verification boundary clean so cryptographic verification and policy matching can be unit tested separately from GHCR and GitHub API behavior.

## GitHub Actions

The repository includes a scheduled workflow and a manual workflow trigger.

Workflow requirements:

- Build the Go application
- Run tests
- Execute `pwncheck verify`
- Use repository secrets or the default GitHub token for API access
- Grant the minimum permissions required to read packages and update the baseline release assets

The workflow failing is the alert mechanism.

## Testing

The implementation should include:

- Unit tests for config parsing and validation
- Unit tests for semver filtering
- Unit tests for baseline comparison rules
- Unit tests for provenance policy matching
- Unit tests for release asset persistence logic using mocked GitHub clients
- A CI smoke path that builds and runs the binary against a fixture configuration

## Open Implementation Choices

These should be decided during implementation:

- Which Go libraries to use for OCI registry access and provenance verification
- Exact JSON schema details for future compatibility
- Exact release tag and asset naming convention

The implementation should prefer direct Go libraries over shelling out to external CLIs unless a library gap makes that impractical.
