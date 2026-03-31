package provenance

import (
	"context"
	"testing"
)

type fakeVerifier struct {
	result Result
	err    error
}

func (f fakeVerifier) Verify(_ context.Context, _ string, _ Policy) (Result, error) {
	return f.result, f.err
}

func TestVerifyRejectsSubjectDigestMismatch(t *testing.T) {
	verifier := fakeVerifier{
		result: Result{SubjectDigest: "sha256:other"},
	}

	_, err := VerifyImage(context.Background(), verifier, "ghcr.io/example/app@sha256:expected", "sha256:expected", Policy{})
	if err == nil {
		t.Fatal("expected subject digest mismatch")
	}
}
