package provenance

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Result struct {
	ProvenanceDigest string
	SubjectDigest    string
	Identity         VerifiedIdentity
}

type Verifier interface {
	Verify(ctx context.Context, imageRef string, policy Policy) (Result, error)
}

type CosignVerifier struct {
	Binary string
}

func NewCosignVerifier() *CosignVerifier {
	return &CosignVerifier{Binary: "cosign"}
}

func VerifyImage(ctx context.Context, verifier Verifier, imageRef string, digest string, policy Policy) (Result, error) {
	result, err := verifier.Verify(ctx, imageRef, policy)
	if err != nil {
		return Result{}, fmt.Errorf("verify provenance for %s: %w", imageRef, err)
	}
	if result.SubjectDigest != digest {
		return Result{}, fmt.Errorf("subject digest mismatch: got %q want %q", result.SubjectDigest, digest)
	}

	return result, nil
}

func (v *CosignVerifier) Verify(ctx context.Context, imageRef string, policy Policy) (Result, error) {
	provenanceDigest, subjectDigest, err := findProvenanceDigest(ctx, imageRef)
	if err != nil {
		return Result{}, err
	}

	args := []string{
		"verify-attestation",
		"--type", "slsaprovenance",
		"--certificate-identity", policy.CertificateIdentity(),
		"--certificate-oidc-issuer", policy.Issuer,
		"--certificate-github-workflow-repository", policy.Repository,
		"--certificate-github-workflow-ref", policy.Ref,
		imageRef,
	}

	cmd := exec.CommandContext(ctx, v.Binary, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("%s %s: %w\n%s", v.Binary, strings.Join(args, " "), err, bytes.TrimSpace(output))
	}

	identity := parseIdentity(output)
	if identity.Subject == "" {
		identity.Subject = policy.CertificateIdentity()
	}
	if identity.Issuer == "" {
		identity.Issuer = policy.Issuer
	}
	if identity.Repository == "" {
		identity.Repository = policy.Repository
	}
	if identity.Ref == "" {
		identity.Ref = policy.Ref
	}
	if identity.WorkflowPath == "" {
		identity.WorkflowPath = policy.WorkflowPath
	}

	return Result{
		ProvenanceDigest: provenanceDigest,
		SubjectDigest:    subjectDigest,
		Identity:         identity,
	}, nil
}

type dsseEnvelope struct {
	Payload string `json:"payload"`
}

type inTotoStatement struct {
	PredicateType string `json:"predicateType"`
	Subject       []struct {
		Name   string            `json:"name"`
		Digest map[string]string `json:"digest"`
	} `json:"subject"`
}

func findProvenanceDigest(ctx context.Context, imageRef string) (string, string, error) {
	repository, digest, err := parseImageReference(imageRef)
	if err != nil {
		return "", "", err
	}

	attestationTag, err := name.NewTag(fmt.Sprintf("%s:%s", repository, attachmentTag(digest)))
	if err != nil {
		return "", "", fmt.Errorf("parse attestation tag: %w", err)
	}

	image, err := remote.Image(attestationTag, remote.WithContext(ctx))
	if err != nil {
		return "", "", fmt.Errorf("load attestation manifest: %w", err)
	}

	layers, err := image.Layers()
	if err != nil {
		return "", "", fmt.Errorf("list attestation layers: %w", err)
	}

	var matches []struct {
		digest  string
		subject string
	}
	for _, layer := range layers {
		statement, err := decodeStatement(layer)
		if err != nil {
			return "", "", err
		}
		if !isProvenancePredicate(statement.PredicateType) {
			continue
		}
		layerDigest, err := layer.Digest()
		if err != nil {
			return "", "", fmt.Errorf("get attestation layer digest: %w", err)
		}
		subjectDigest, err := statementSubjectDigest(statement)
		if err != nil {
			return "", "", err
		}
		matches = append(matches, struct {
			digest  string
			subject string
		}{
			digest:  layerDigest.String(),
			subject: subjectDigest,
		})
	}

	switch len(matches) {
	case 0:
		return "", "", errors.New("no slsaprovenance attestation found")
	case 1:
		return matches[0].digest, matches[0].subject, nil
	default:
		return "", "", fmt.Errorf("found %d slsaprovenance attestations", len(matches))
	}
}

func parseImageReference(imageRef string) (string, string, error) {
	ref, err := name.ParseReference(imageRef)
	if err != nil {
		return "", "", fmt.Errorf("parse image reference %q: %w", imageRef, err)
	}

	digestRef, ok := ref.(name.Digest)
	if !ok {
		return "", "", fmt.Errorf("image reference %q must include a digest", imageRef)
	}

	return digestRef.Context().Name(), digestRef.DigestStr(), nil
}

func attachmentTag(digest string) string {
	algorithm, hex, found := strings.Cut(digest, ":")
	if !found {
		return digest + ".att"
	}
	return algorithm + "-" + hex + ".att"
}

func decodeStatement(layer v1.Layer) (inTotoStatement, error) {
	reader, err := layer.Compressed()
	if err != nil {
		return inTotoStatement{}, fmt.Errorf("open attestation layer: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return inTotoStatement{}, fmt.Errorf("read attestation layer: %w", err)
	}

	var envelope dsseEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return inTotoStatement{}, fmt.Errorf("decode dsse envelope: %w", err)
	}

	payload, err := base64.StdEncoding.DecodeString(envelope.Payload)
	if err != nil {
		return inTotoStatement{}, fmt.Errorf("decode dsse payload: %w", err)
	}

	var statement inTotoStatement
	if err := json.Unmarshal(payload, &statement); err != nil {
		return inTotoStatement{}, fmt.Errorf("decode in-toto statement: %w", err)
	}

	return statement, nil
}

func isProvenancePredicate(predicateType string) bool {
	switch predicateType {
	case "https://slsa.dev/provenance/v0.2", "https://slsa.dev/provenance/v1":
		return true
	default:
		return false
	}
}

func statementSubjectDigest(statement inTotoStatement) (string, error) {
	if len(statement.Subject) == 0 {
		return "", errors.New("attestation statement has no subjects")
	}

	for algorithm, digest := range statement.Subject[0].Digest {
		return algorithm + ":" + digest, nil
	}

	return "", errors.New("attestation subject digest is empty")
}

func parseIdentity(output []byte) VerifiedIdentity {
	var identity VerifiedIdentity

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "Certificate subject: "):
			identity.Subject = strings.TrimPrefix(line, "Certificate subject: ")
		case strings.HasPrefix(line, "Certificate issuer URL: "):
			identity.Issuer = strings.TrimPrefix(line, "Certificate issuer URL: ")
		case strings.HasPrefix(line, "GitHub Workflow Repository: "):
			identity.Repository = strings.TrimPrefix(line, "GitHub Workflow Repository: ")
		case strings.HasPrefix(line, "GitHub Workflow Ref: "):
			identity.Ref = strings.TrimPrefix(line, "GitHub Workflow Ref: ")
		}
	}

	if identity.Subject != "" {
		trimmed := strings.TrimPrefix(identity.Subject, "https://github.com/")
		repositoryPath, ref, found := strings.Cut(trimmed, "@")
		if found {
			identity.Ref = ref
			parts := strings.Split(repositoryPath, "/")
			if len(parts) >= 3 {
				identity.Repository = parts[0] + "/" + parts[1]
				identity.WorkflowPath = strings.Join(parts[2:], "/")
			}
		}
	}

	return identity
}
