package provenance

import "testing"

func TestPolicyMatchesExpectedIdentity(t *testing.T) {
	policy := Policy{
		Issuer:       "https://token.actions.githubusercontent.com",
		Repository:   "external-secrets/external-secrets",
		WorkflowPath: ".github/workflows/release.yaml",
		Ref:          "refs/heads/main",
	}

	identity := VerifiedIdentity{
		Issuer:       "https://token.actions.githubusercontent.com",
		Repository:   "external-secrets/external-secrets",
		WorkflowPath: ".github/workflows/release.yaml",
		Ref:          "refs/heads/main",
	}

	if err := policy.Match(identity); err != nil {
		t.Fatalf("match policy: %v", err)
	}
}

func TestPolicyRejectsWorkflowMismatch(t *testing.T) {
	policy := Policy{WorkflowPath: ".github/workflows/release.yaml"}
	identity := VerifiedIdentity{WorkflowPath: ".github/workflows/other.yaml"}

	if err := policy.Match(identity); err == nil {
		t.Fatal("expected policy mismatch")
	}
}
