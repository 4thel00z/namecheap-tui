package namecheap

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.RegistrarAPI = (*Client)(nil)

// dateLayout is Namecheap's MM/DD/YYYY date format.
const dateLayout = "01/02/2006"

func parseDate(s string) time.Time {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

type xmlDomain struct {
	ID         string `xml:"ID,attr"`
	Name       string `xml:"Name,attr"`
	User       string `xml:"User,attr"`
	Created    string `xml:"Created,attr"`
	Expires    string `xml:"Expires,attr"`
	IsExpired  bool   `xml:"IsExpired,attr"`
	IsLocked   bool   `xml:"IsLocked,attr"`
	AutoRenew  bool   `xml:"AutoRenew,attr"`
	WhoisGuard string `xml:"WhoisGuard,attr"`
}

type getListResponse struct {
	Domains []xmlDomain `xml:"CommandResponse>DomainGetListResult>Domain"`
	Paging  struct {
		TotalItems  int `xml:"TotalItems"`
		CurrentPage int `xml:"CurrentPage"`
		PageSize    int `xml:"PageSize"`
	} `xml:"CommandResponse>Paging"`
}

// ListDomains fetches every page of namecheap.domains.getList.
func (c *Client) ListDomains(ctx context.Context) ([]registrar.Domain, error) {
	var out []registrar.Domain
	for page := 1; ; page++ {
		params := url.Values{}
		params.Set("Page", strconv.Itoa(page))
		params.Set("PageSize", "100")
		var resp getListResponse
		if err := c.call(ctx, "namecheap.domains.getList", params, &resp); err != nil {
			return nil, err
		}
		for _, d := range resp.Domains {
			name, err := registrar.Parse(d.Name)
			if err != nil {
				return nil, fmt.Errorf("api returned invalid domain %q: %w", d.Name, err)
			}
			out = append(out, registrar.Domain{
				ID: d.ID, Name: name, Owner: d.User,
				Created: parseDate(d.Created), Expires: parseDate(d.Expires),
				AutoRenew: d.AutoRenew, Locked: d.IsLocked,
				Privacy: d.WhoisGuard == "ENABLED", Expired: d.IsExpired,
			})
		}
		fetched := resp.Paging.CurrentPage * resp.Paging.PageSize
		if resp.Paging.PageSize == 0 || fetched >= resp.Paging.TotalItems {
			return out, nil
		}
	}
}

type checkResponse struct {
	Results []struct {
		Domain       string  `xml:"Domain,attr"`
		Available    bool    `xml:"Available,attr"`
		IsPremium    bool    `xml:"IsPremiumName,attr"`
		PremiumPrice float64 `xml:"PremiumRegistrationPrice,attr"`
	} `xml:"CommandResponse>DomainCheckResult"`
}

// CheckDomains checks availability via namecheap.domains.check.
func (c *Client) CheckDomains(ctx context.Context, names []registrar.DomainName) ([]registrar.Availability, error) {
	strs := make([]string, len(names))
	for i, n := range names {
		strs[i] = n.String()
	}
	params := url.Values{}
	params.Set("DomainList", strings.Join(strs, ","))
	var resp checkResponse
	if err := c.call(ctx, "namecheap.domains.check", params, &resp); err != nil {
		return nil, err
	}
	out := make([]registrar.Availability, 0, len(resp.Results))
	for _, r := range resp.Results {
		name, err := registrar.Parse(r.Domain)
		if err != nil {
			return nil, fmt.Errorf("api returned invalid domain %q: %w", r.Domain, err)
		}
		out = append(out, registrar.Availability{
			Name: name, Available: r.Available,
			Premium: r.IsPremium, PremiumPrice: r.PremiumPrice,
		})
	}
	return out, nil
}

type getInfoResponse struct {
	Result struct {
		Status     string `xml:"Status,attr"`
		ID         string `xml:"ID,attr"`
		DomainName string `xml:"DomainName,attr"`
		OwnerName  string `xml:"OwnerName,attr"`
		Details    struct {
			CreatedDate string `xml:"CreatedDate"`
			ExpiredDate string `xml:"ExpiredDate"`
		} `xml:"DomainDetails"`
		Whoisguard struct {
			Enabled string `xml:"Enabled,attr"`
		} `xml:"Whoisguard"`
		DNS struct {
			ProviderType string   `xml:"ProviderType,attr"`
			Nameservers  []string `xml:"Nameserver"`
		} `xml:"DnsDetails"`
	} `xml:"CommandResponse>DomainGetInfoResult"`
}

// DomainInfo fetches namecheap.domains.getInfo for one domain.
func (c *Client) DomainInfo(ctx context.Context, name registrar.DomainName) (registrar.Details, error) {
	params := url.Values{}
	params.Set("DomainName", name.String())
	var resp getInfoResponse
	if err := c.call(ctx, "namecheap.domains.getInfo", params, &resp); err != nil {
		return registrar.Details{}, err
	}
	r := resp.Result
	return registrar.Details{
		Domain: registrar.Domain{
			ID: r.ID, Name: name, Owner: r.OwnerName,
			Created: parseDate(r.Details.CreatedDate),
			Expires: parseDate(r.Details.ExpiredDate),
			Privacy: strings.EqualFold(r.Whoisguard.Enabled, "true"),
		},
		Status:      r.Status,
		DNSProvider: r.DNS.ProviderType,
		Nameservers: r.DNS.Nameservers,
	}, nil
}
