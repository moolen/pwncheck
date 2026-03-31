# Verify Workflow Schedule Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change the verification workflow schedule to run every 6 hours at minute 0 without changing any other workflow behavior.

**Architecture:** This is a workflow-only change. Update the single cron expression in `.github/workflows/verify.yaml`, keep the existing manual trigger and jobs intact, then verify with the existing workflow test and a live GitHub Actions run.

**Tech Stack:** GitHub Actions workflow YAML, Go test suite, GitHub CLI.

---

### Task 1: Update the workflow schedule

**Files:**
- Modify: `.github/workflows/verify.yaml`
- Test: `cmd/pwncheck/main_test.go`

- [ ] **Step 1: Use the existing workflow test as the verification guard**

Run:

```bash
go test ./cmd/pwncheck -run TestVerifyWorkflowExists
```

Expected: PASS before the change so we know the workflow file is readable.

- [ ] **Step 2: Change the cron schedule**

Update:

```yaml
- cron: "17 3 * * *"
```

To:

```yaml
- cron: "0 */6 * * *"
```

- [ ] **Step 3: Re-run local verification**

Run:

```bash
go test ./cmd/pwncheck -run TestVerifyWorkflowExists
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/verify.yaml
git commit -m "ci: run verification every 6 hours"
```

- [ ] **Step 5: Push and re-run the workflow**

Run:

```bash
git push --force origin feat/pwncheck-implementation:main
gh workflow run verify.yaml --repo moolen/pwncheck --ref main
gh run watch --repo moolen/pwncheck --exit-status
```

Expected: the workflow passes on the updated schedule definition.
