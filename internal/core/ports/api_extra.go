package ports

import (
	"context"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/ssl"
)

// SSLAPI manages SSL certificates.
type SSLAPI interface {
	ListCertificates(ctx context.Context) ([]ssl.Certificate, error)
	PurchaseCertificate(ctx context.Context, p ssl.Purchase) (ssl.Certificate, error)
	ActivateCertificate(ctx context.Context, a ssl.Activation) error
	CertificateInfo(ctx context.Context, id string) (ssl.Certificate, error)
	RenewCertificate(ctx context.Context, id string, p ssl.Purchase) (ssl.Certificate, error)
	ReissueCertificate(ctx context.Context, a ssl.Activation) error
	ApproverEmails(ctx context.Context, domain string, certType string) ([]string, error)
	ResendApproverEmail(ctx context.Context, id string) error
	RevokeCertificate(ctx context.Context, id string, certType string) error
}

// PrivacyAPI manages domain-privacy (whoisguard) subscriptions.
type PrivacyAPI interface {
	ListPrivacy(ctx context.Context) ([]account.PrivacySubscription, error)
	EnablePrivacy(ctx context.Context, id, forwardTo string) error
	DisablePrivacy(ctx context.Context, id string) error
	RenewPrivacy(ctx context.Context, id string, years int) error
	ChangePrivacyEmail(ctx context.Context, id string) error
	AssignPrivacy(ctx context.Context, id, domain string) error
	UnassignPrivacy(ctx context.Context, id string) error
	DiscardPrivacy(ctx context.Context, id string) error
}

// AccountAPI exposes balances, pricing, and the address book.
type AccountAPI interface {
	Balances(ctx context.Context) (account.Balance, error)
	Pricing(ctx context.Context, productType, category, product string) ([]account.Price, error)
	ListAddresses(ctx context.Context) ([]account.Address, error)
	GetAddress(ctx context.Context, id string) (account.Address, error)
	CreateAddress(ctx context.Context, a account.Address) error
	UpdateAddress(ctx context.Context, a account.Address) error
	DeleteAddress(ctx context.Context, id string) error
	SetDefaultAddress(ctx context.Context, id string) error
}
