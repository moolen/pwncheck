package provenance

import (
	"context"
	"testing"
)

type fakeVerifier struct {
	result Result
	err    error
}

func (f fakeVerifier) Verify(_ context.Context, _ string) (Result, error) {
	return f.result, f.err
}

func TestVerifyRejectsSubjectDigestMismatch(t *testing.T) {
	verifier := fakeVerifier{
		result: Result{SubjectDigest: "sha256:other"},
	}

	_, err := VerifyDigest(context.Background(), verifier, "sha256:expected")
	if err == nil {
		t.Fatal("expected subject digest mismatch")
	}
}
