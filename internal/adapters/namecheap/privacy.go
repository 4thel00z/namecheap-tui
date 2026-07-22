package namecheap

import (
	"context"
	"net/url"
	"strconv"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.PrivacyAPI = (*Client)(nil)

// ListPrivacy fetches namecheap.whoisguard.getList (all pages merged).
func (c *Client) ListPrivacy(ctx context.Context) ([]account.PrivacySubscription, error) {
	var out []account.PrivacySubscription
	for page := 1; ; page++ {
		params := url.Values{}
		params.Set("Page", strconv.Itoa(page))
		params.Set("PageSize", "100")
		var resp struct {
			Entries []struct {
				ID      string `xml:"ID,attr"`
				Domain  string `xml:"DomainName,attr"`
				Created string `xml:"Created,attr"`
				Expires string `xml:"Expires,attr"`
				Status  string `xml:"Status,attr"`
			} `xml:"CommandResponse>WhoisguardGetListResult>Whoisguard"`
			Paging struct {
				TotalItems  int `xml:"TotalItems"`
				CurrentPage int `xml:"CurrentPage"`
				PageSize    int `xml:"PageSize"`
			} `xml:"CommandResponse>Paging"`
		}
		if err := c.call(ctx, "namecheap.whoisguard.getList", params, &resp); err != nil {
			return nil, err
		}
		for _, e := range resp.Entries {
			out = append(out, account.PrivacySubscription{
				ID: e.ID, Domain: e.Domain, Status: e.Status,
				Created: parseDate(e.Created), Expires: parseDate(e.Expires),
			})
		}
		fetched := resp.Paging.CurrentPage * resp.Paging.PageSize
		if resp.Paging.PageSize == 0 || fetched >= resp.Paging.TotalItems {
			return out, nil
		}
	}
}

func (c *Client) privacyCall(ctx context.Context, command, id string, extra url.Values) error {
	params := url.Values{}
	params.Set("WhoisguardID", id)
	for k, vs := range extra {
		for _, v := range vs {
			params.Add(k, v)
		}
	}
	return c.call(ctx, command, params, &struct{}{})
}

// EnablePrivacy enables a subscription via namecheap.whoisguard.enable.
func (c *Client) EnablePrivacy(ctx context.Context, id, forwardTo string) error {
	extra := url.Values{}
	if forwardTo != "" {
		extra.Set("ForwardedToEmail", forwardTo)
	}
	return c.privacyCall(ctx, "namecheap.whoisguard.enable", id, extra)
}

// DisablePrivacy disables a subscription via namecheap.whoisguard.disable.
func (c *Client) DisablePrivacy(ctx context.Context, id string) error {
	return c.privacyCall(ctx, "namecheap.whoisguard.disable", id, nil)
}

// RenewPrivacy renews via namecheap.whoisguard.renew.
func (c *Client) RenewPrivacy(ctx context.Context, id string, years int) error {
	extra := url.Values{}
	extra.Set("Years", strconv.Itoa(years))
	return c.privacyCall(ctx, "namecheap.whoisguard.renew", id, extra)
}

// ChangePrivacyEmail rotates the contact address via
// namecheap.whoisguard.changeemailaddress.
func (c *Client) ChangePrivacyEmail(ctx context.Context, id string) error {
	return c.privacyCall(ctx, "namecheap.whoisguard.changeemailaddress", id, nil)
}

// AssignPrivacy attaches a subscription to a domain via namecheap.whoisguard.allot.
func (c *Client) AssignPrivacy(ctx context.Context, id, domain string) error {
	extra := url.Values{}
	extra.Set("DomainName", domain)
	return c.privacyCall(ctx, "namecheap.whoisguard.allot", id, extra)
}

// UnassignPrivacy detaches a subscription via namecheap.whoisguard.unallot.
func (c *Client) UnassignPrivacy(ctx context.Context, id string) error {
	return c.privacyCall(ctx, "namecheap.whoisguard.unallot", id, nil)
}

// DiscardPrivacy discards a subscription via namecheap.whoisguard.discard.
func (c *Client) DiscardPrivacy(ctx context.Context, id string) error {
	return c.privacyCall(ctx, "namecheap.whoisguard.discard", id, nil)
}
