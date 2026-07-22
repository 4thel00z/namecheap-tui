// Package ssl holds SSL-certificate types.
package ssl

import (
	"fmt"
	"time"
)

// Certificate is one SSL certificate on the account.
type Certificate struct {
	ID                string
	Host              string
	Type              string
	Status            string
	Years             int
	Purchased         time.Time
	Expires           time.Time
	ActivationExpires time.Time
	Expired           bool
}

// Purchase is a certificate-purchase request.
type Purchase struct {
	Type  string // e.g. "PositiveSSL"
	Years int
}

// Validate checks the purchase request.
func (p Purchase) Validate() error {
	if p.Type == "" {
		return fmt.Errorf("certificate type is required")
	}
	if p.Years < 1 || p.Years > 5 {
		return fmt.Errorf("years %d out of range 1..5", p.Years)
	}
	return nil
}

// DVMethod is a domain-control validation method.
type DVMethod string

const (
	DVEmail DVMethod = "email"
	DVHTTP  DVMethod = "http"
	DVDNS   DVMethod = "dns"
)

// Activation is a certificate-activation request.
type Activation struct {
	CertificateID string
	CSR           string
	WebServerType string // e.g. "nginx", "apacheopenssl"
	DVMethod      DVMethod
	ApproverEmail string // required for DVEmail
}

// Validate checks the activation request.
func (a Activation) Validate() error {
	if a.CertificateID == "" {
		return fmt.Errorf("certificate id is required")
	}
	if a.CSR == "" {
		return fmt.Errorf("csr is required")
	}
	switch a.DVMethod {
	case DVEmail:
		if a.ApproverEmail == "" {
			return fmt.Errorf("approver email is required for email validation")
		}
	case DVHTTP, DVDNS:
	default:
		return fmt.Errorf("unknown dv method %q (want email, http, or dns)", a.DVMethod)
	}
	return nil
}
