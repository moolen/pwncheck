package state

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestRepositoryStateJSONRoundTrip(t *testing.T) {
	now := time.Unix(123, 0).UTC()
	input := RepositoryState{
		SchemaVersion: 1,
		Repository:    "external-secrets",
		UpdatedAt:     now,
		Tags: map[string]TagRecord{
			"v1.2.3": {
				Tag:              "v1.2.3",
				ManifestDigest:   "sha256:abc",
				ProvenanceDigest: "sha256:def",
				VerificationTime: now,
				VerifiedIdentity: VerifiedIdentity{
					Issuer:       "https://token.actions.githubusercontent.com",
					Repository:   "external-secrets/external-secrets",
					WorkflowPath: ".github/workflows/release.yaml",
					Ref:          "refs/heads/main",
				},
			},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var output RepositoryState
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if diff := cmp.Diff(input, output); diff != "" {
		t.Fatalf("state mismatch (-want +got):\n%s", diff)
	}
}

func TestDetectDriftReportsChangedDigest(t *testing.T) {
	baseline := TagRecord{Tag: "v1.2.3", ManifestDigest: "sha256:old", ProvenanceDigest: "sha256:prov"}
	observed := TagRecord{Tag: "v1.2.3", ManifestDigest: "sha256:new", ProvenanceDigest: "sha256:prov"}

	result := CompareTag(baseline, observed)
	if !result.Drift {
		t.Fatal("expected drift to be detected")
	}
}
