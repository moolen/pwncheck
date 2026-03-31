package provenance

import (
	"context"
	"fmt"
)

type Result struct {
	ProvenanceDigest string
	SubjectDigest    string
	Identity         VerifiedIdentity
}

type Verifier interface {
	Verify(ctx context.Context, digest string) (Result, error)
}

func VerifyDigest(ctx context.Context, verifier Verifier, digest string) (Result, error) {
	result, err := verifier.Verify(ctx, digest)
	if err != nil {
		return Result{}, fmt.Errorf("verify provenance for %s: %w", digest, err)
	}
	if result.SubjectDigest != digest {
		return Result{}, fmt.Errorf("subject digest mismatch: got %q want %q", result.SubjectDigest, digest)
	}

	return result, nil
}
