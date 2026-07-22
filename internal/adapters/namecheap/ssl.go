package namecheap

import (
	"context"
	"net/url"
	"strconv"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/ssl"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.SSLAPI = (*Client)(nil)

type xmlSSL struct {
	ID                string `xml:"CertificateID,attr"`
	Host              string `xml:"HostName,attr"`
	Type              string `xml:"SSLType,attr"`
	Status            string `xml:"Status,attr"`
	Years             int    `xml:"Years,attr"`
	PurchaseDate      string `xml:"PurchaseDate,attr"`
	ExpireDate        string `xml:"ExpireDate,attr"`
	ActivationExpires string `xml:"ActivationExpireDate,attr"`
	Expired           bool   `xml:"IsExpiredYN,attr"`
}

func (x xmlSSL) cert() ssl.Certificate {
	return ssl.Certificate{
		ID: x.ID, Host: x.Host, Type: x.Type, Status: x.Status, Years: x.Years,
		Purchased: parseDate(x.PurchaseDate), Expires: parseDate(x.ExpireDate),
		ActivationExpires: parseDate(x.ActivationExpires), Expired: x.Expired,
	}
}

// ListCertificates fetches namecheap.ssl.getList (all pages merged).
func (c *Client) ListCertificates(ctx context.Context) ([]ssl.Certificate, error) {
	var out []ssl.Certificate
	for page := 1; ; page++ {
		params := url.Values{}
		params.Set("Page", strconv.Itoa(page))
		params.Set("PageSize", "100")
		var resp struct {
			SSL    []xmlSSL `xml:"CommandResponse>SSLListResult>SSL"`
			Paging struct {
				TotalItems  int `xml:"TotalItems"`
				CurrentPage int `xml:"CurrentPage"`
				PageSize    int `xml:"PageSize"`
			} `xml:"CommandResponse>Paging"`
		}
		if err := c.call(ctx, "namecheap.ssl.getList", params, &resp); err != nil {
			return nil, err
		}
		for _, s := range resp.SSL {
			out = append(out, s.cert())
		}
		fetched := resp.Paging.CurrentPage * resp.Paging.PageSize
		if resp.Paging.PageSize == 0 || fetched >= resp.Paging.TotalItems {
			return out, nil
		}
	}
}

// PurchaseCertificate buys a certificate via namecheap.ssl.create.
func (c *Client) PurchaseCertificate(ctx context.Context, p ssl.Purchase) (ssl.Certificate, error) {
	if err := p.Validate(); err != nil {
		return ssl.Certificate{}, err
	}
	params := url.Values{}
	params.Set("Type", p.Type)
	params.Set("Years", strconv.Itoa(p.Years))
	var resp struct {
		Cert xmlSSL `xml:"CommandResponse>SSLCreateResult>SSLCertificate"`
	}
	if err := c.call(ctx, "namecheap.ssl.create", params, &resp); err != nil {
		return ssl.Certificate{}, err
	}
	return resp.Cert.cert(), nil
}

func activationParams(a ssl.Activation) url.Values {
	params := url.Values{}
	params.Set("CertificateID", a.CertificateID)
	params.Set("CSR", a.CSR)
	if a.WebServerType != "" {
		params.Set("WebServerType", a.WebServerType)
	}
	switch a.DVMethod {
	case ssl.DVEmail:
		params.Set("ApproverEmail", a.ApproverEmail)
	case ssl.DVHTTP:
		params.Set("HTTPDCValidation", "true")
	case ssl.DVDNS:
		params.Set("DNSDCValidation", "true")
	}
	return params
}

// ActivateCertificate activates via namecheap.ssl.activate.
func (c *Client) ActivateCertificate(ctx context.Context, a ssl.Activation) error {
	if err := a.Validate(); err != nil {
		return err
	}
	return c.call(ctx, "namecheap.ssl.activate", activationParams(a), &struct{}{})
}

// CertificateInfo fetches namecheap.ssl.getInfo.
func (c *Client) CertificateInfo(ctx context.Context, id string) (ssl.Certificate, error) {
	params := url.Values{}
	params.Set("CertificateID", id)
	var resp struct {
		Result struct {
			Status     string `xml:"Status,attr"`
			Type       string `xml:"Type,attr"`
			IssuedOn   string `xml:"IssuedOn,attr"`
			Expires    string `xml:"Expires,attr"`
			Activation string `xml:"ActivationExpireDate,attr"`
			Years      int    `xml:"Years,attr"`
		} `xml:"CommandResponse>SSLGetInfoResult"`
	}
	if err := c.call(ctx, "namecheap.ssl.getInfo", params, &resp); err != nil {
		return ssl.Certificate{}, err
	}
	r := resp.Result
	return ssl.Certificate{
		ID: id, Type: r.Type, Status: r.Status, Years: r.Years,
		Purchased: parseDate(r.IssuedOn), Expires: parseDate(r.Expires),
		ActivationExpires: parseDate(r.Activation),
	}, nil
}

// RenewCertificate renews via namecheap.ssl.renew.
func (c *Client) RenewCertificate(ctx context.Context, id string, p ssl.Purchase) (ssl.Certificate, error) {
	if err := p.Validate(); err != nil {
		return ssl.Certificate{}, err
	}
	params := url.Values{}
	params.Set("CertificateID", id)
	params.Set("SSLType", p.Type)
	params.Set("Years", strconv.Itoa(p.Years))
	var resp struct {
		Result struct {
			ID string `xml:"CertificateID,attr"`
		} `xml:"CommandResponse>SSLRenewResult"`
	}
	if err := c.call(ctx, "namecheap.ssl.renew", params, &resp); err != nil {
		return ssl.Certificate{}, err
	}
	return ssl.Certificate{ID: resp.Result.ID, Type: p.Type, Years: p.Years}, nil
}

// ReissueCertificate reissues via namecheap.ssl.reissue.
func (c *Client) ReissueCertificate(ctx context.Context, a ssl.Activation) error {
	if err := a.Validate(); err != nil {
		return err
	}
	return c.call(ctx, "namecheap.ssl.reissue", activationParams(a), &struct{}{})
}

// ApproverEmails lists valid DV approver addresses via
// namecheap.ssl.getApproverEmailList.
func (c *Client) ApproverEmails(ctx context.Context, domain, certType string) ([]string, error) {
	params := url.Values{}
	params.Set("DomainName", domain)
	params.Set("CertificateType", certType)
	var resp struct {
		Domain  []string `xml:"CommandResponse>GetApproverEmailListResult>Domainemails>email"`
		Generic []string `xml:"CommandResponse>GetApproverEmailListResult>Generalemails>email"`
	}
	if err := c.call(ctx, "namecheap.ssl.getApproverEmailList", params, &resp); err != nil {
		return nil, err
	}
	return append(resp.Domain, resp.Generic...), nil
}

// ResendApproverEmail resends the DV email via namecheap.ssl.resendApproverEmail.
func (c *Client) ResendApproverEmail(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("CertificateID", id)
	return c.call(ctx, "namecheap.ssl.resendApproverEmail", params, &struct{}{})
}

// RevokeCertificate revokes via namecheap.ssl.revokecertificate.
func (c *Client) RevokeCertificate(ctx context.Context, id, certType string) error {
	params := url.Values{}
	params.Set("CertificateID", id)
	params.Set("CertificateType", certType)
	return c.call(ctx, "namecheap.ssl.revokecertificate", params, &struct{}{})
}
