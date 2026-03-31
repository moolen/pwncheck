package state

import "time"

const CurrentSchemaVersion = 1

type RepositoryState struct {
	SchemaVersion int                  `json:"schemaVersion"`
	Repository    string               `json:"repository"`
	UpdatedAt     time.Time            `json:"updatedAt"`
	Tags          map[string]TagRecord `json:"tags"`
}

type TagRecord struct {
	Tag              string           `json:"tag"`
	ManifestDigest   string           `json:"manifestDigest"`
	ProvenanceDigest string           `json:"provenanceDigest"`
	VerificationTime time.Time        `json:"verificationTime"`
	VerifiedIdentity VerifiedIdentity `json:"verifiedIdentity"`
}

type VerifiedIdentity struct {
	Issuer       string `json:"issuer"`
	Repository   string `json:"repository"`
	WorkflowPath string `json:"workflowPath"`
	Ref          string `json:"ref"`
	Subject      string `json:"subject,omitempty"`
}

type CompareResult struct {
	Drift   bool
	Reasons []string
}

func CompareTag(baseline, observed TagRecord) CompareResult {
	result := CompareResult{}

	if baseline.ManifestDigest != observed.ManifestDigest {
		result.Drift = true
		result.Reasons = append(result.Reasons, "manifest digest changed")
	}

	if baseline.ProvenanceDigest != observed.ProvenanceDigest {
		result.Drift = true
		result.Reasons = append(result.Reasons, "provenance digest changed")
	}

	return result
}
