package provenance

import (
	"fmt"
	"path"
)

type Policy struct {
	Issuer       string
	Repository   string
	WorkflowPath string
	Ref          string
}

type VerifiedIdentity struct {
	Issuer       string
	Repository   string
	WorkflowPath string
	Ref          string
	Subject      string
}

func (p Policy) CertificateIdentity() string {
	return "https://github.com/" + path.Join(p.Repository, p.WorkflowPath) + "@" + p.Ref
}

func (p Policy) Match(identity VerifiedIdentity) error {
	switch {
	case p.Issuer != "" && p.Issuer != identity.Issuer:
		return fmt.Errorf("issuer mismatch: got %q want %q", identity.Issuer, p.Issuer)
	case p.Repository != "" && p.Repository != identity.Repository:
		return fmt.Errorf("repository mismatch: got %q want %q", identity.Repository, p.Repository)
	case p.WorkflowPath != "" && p.WorkflowPath != identity.WorkflowPath:
		return fmt.Errorf("workflow mismatch: got %q want %q", identity.WorkflowPath, p.WorkflowPath)
	case p.Ref != "" && p.Ref != identity.Ref:
		return fmt.Errorf("ref mismatch: got %q want %q", identity.Ref, p.Ref)
	default:
		return nil
	}
}
