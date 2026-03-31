# Verify Workflow Schedule Design

## Goal

Adjust the `pwncheck` GitHub Actions schedule so the verification workflow runs every 6 hours at minute 0.

## Scope

- Change only the cron expression in `.github/workflows/verify.yaml`
- Keep the manual trigger and all workflow steps unchanged
- Reuse the existing workflow test and live GitHub Actions verification path

## Design

The workflow currently runs once daily. Replace the cron expression with:

```yaml
schedule:
  - cron: "0 */6 * * *"
```

This runs at:

- `00:00` UTC
- `06:00` UTC
- `12:00` UTC
- `18:00` UTC

No other workflow behavior changes.

## Verification

- Run `go test ./cmd/pwncheck -run TestVerifyWorkflowExists`
- Run `go test ./...`
- Push to `main`
- Trigger the workflow manually once and confirm it still passes
