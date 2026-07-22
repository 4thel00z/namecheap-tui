package namecheap

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.DNSAPI = (*Client)(nil)

func sldTLD(name registrar.DomainName) url.Values {
	v := url.Values{}
	v.Set("SLD", name.SLD)
	v.Set("TLD", name.TLD)
	return v
}

type xmlHost struct {
	HostID  string `xml:"HostId,attr"`
	Name    string `xml:"Name,attr"`
	Type    string `xml:"Type,attr"`
	Address string `xml:"Address,attr"`
	MXPref  int    `xml:"MXPref,attr"`
	TTL     int    `xml:"TTL,attr"`
}

type getHostsResponse struct {
	// The live API emits lowercase <host>; docs show both cases.
	Lower []xmlHost `xml:"CommandResponse>DomainDNSGetHostsResult>host"`
	Upper []xmlHost `xml:"CommandResponse>DomainDNSGetHostsResult>Host"`
}

// GetHosts fetches all host records via namecheap.domains.dns.getHosts.
func (c *Client) GetHosts(ctx context.Context, name registrar.DomainName) ([]dns.HostRecord, error) {
	var resp getHostsResponse
	if err := c.call(ctx, "namecheap.domains.dns.getHosts", sldTLD(name), &resp); err != nil {
		return nil, err
	}
	hosts := append(resp.Lower, resp.Upper...)
	out := make([]dns.HostRecord, 0, len(hosts))
	for _, h := range hosts {
		rt, err := dns.ParseRecordType(h.Type)
		if err != nil {
			return nil, fmt.Errorf("api returned unknown record type %q: %w", h.Type, err)
		}
		out = append(out, dns.HostRecord{
			ID: h.HostID, Name: h.Name, Type: rt, Value: h.Address,
			TTL: h.TTL, MXPref: h.MXPref,
		})
	}
	return out, nil
}

// SetHosts replaces the domain's entire record set via
// namecheap.domains.dns.setHosts.
func (c *Client) SetHosts(ctx context.Context, name registrar.DomainName, records []dns.HostRecord) error {
	params := sldTLD(name)
	hasMX := false
	for i, r := range records {
		n := strconv.Itoa(i + 1)
		ttl := r.TTL
		if ttl == 0 {
			ttl = dns.DefaultTTL
		}
		params.Set("HostName"+n, r.Name)
		params.Set("RecordType"+n, string(r.Type))
		params.Set("Address"+n, r.Value)
		params.Set("TTL"+n, strconv.Itoa(ttl))
		if r.Type == dns.MX {
			hasMX = true
			pref := r.MXPref
			if pref == 0 {
				pref = 10
			}
			params.Set("MXPref"+n, strconv.Itoa(pref))
		}
	}
	if hasMX {
		params.Set("EmailType", "MX")
	}
	var resp struct {
		Result struct {
			IsSuccess bool `xml:"IsSuccess,attr"`
		} `xml:"CommandResponse>DomainDNSSetHostsResult"`
	}
	return c.call(ctx, "namecheap.domains.dns.setHosts", params, &resp)
}

type getNSListResponse struct {
	Result struct {
		UsingOurDNS bool     `xml:"IsUsingOurDNS,attr"`
		Nameservers []string `xml:"Nameserver"`
	} `xml:"CommandResponse>DomainDNSGetListResult"`
}

// GetNameservers reports the domain's nameserver configuration.
func (c *Client) GetNameservers(ctx context.Context, name registrar.DomainName) (dns.NameserverInfo, error) {
	var resp getNSListResponse
	if err := c.call(ctx, "namecheap.domains.dns.getList", sldTLD(name), &resp); err != nil {
		return dns.NameserverInfo{}, err
	}
	return dns.NameserverInfo{
		UsingOurDNS: resp.Result.UsingOurDNS,
		Nameservers: resp.Result.Nameservers,
	}, nil
}

// SetDefaultNS switches the domain to Namecheap's default nameservers.
func (c *Client) SetDefaultNS(ctx context.Context, name registrar.DomainName) error {
	return c.call(ctx, "namecheap.domains.dns.setDefault", sldTLD(name), &struct{}{})
}

// SetCustomNS switches the domain to custom nameservers.
func (c *Client) SetCustomNS(ctx context.Context, name registrar.DomainName, ns []string) error {
	params := sldTLD(name)
	params.Set("Nameservers", strings.Join(ns, ","))
	return c.call(ctx, "namecheap.domains.dns.setCustom", params, &struct{}{})
}

type getEmailFwdResponse struct {
	// The live API emits lowercase <forward mailbox=...>; accept both cases.
	Lower []xmlForward `xml:"CommandResponse>DomainDNSGetEmailForwardingResult>forward"`
	Upper []xmlForward `xml:"CommandResponse>DomainDNSGetEmailForwardingResult>Forward"`
}

type xmlForward struct {
	Mailbox     string `xml:"mailbox,attr"`
	MailboxName string `xml:"mailboxname,attr"`
	To          string `xml:",chardata"`
}

// GetEmailForwarding returns the domain's mailbox forwards.
func (c *Client) GetEmailForwarding(ctx context.Context, name registrar.DomainName) ([]dns.EmailForward, error) {
	params := url.Values{}
	params.Set("DomainName", name.String())
	var resp getEmailFwdResponse
	if err := c.call(ctx, "namecheap.domains.dns.getEmailForwarding", params, &resp); err != nil {
		return nil, err
	}
	fwds := append(resp.Lower, resp.Upper...)
	out := make([]dns.EmailForward, 0, len(fwds))
	for _, f := range fwds {
		mailbox := f.Mailbox
		if mailbox == "" {
			mailbox = f.MailboxName
		}
		out = append(out, dns.EmailForward{
			Mailbox: mailbox, ForwardTo: strings.TrimSpace(f.To),
		})
	}
	return out, nil
}

// SetEmailForwarding replaces the domain's mailbox forwards.
func (c *Client) SetEmailForwarding(ctx context.Context, name registrar.DomainName, fwds []dns.EmailForward) error {
	params := url.Values{}
	params.Set("DomainName", name.String())
	for i, f := range fwds {
		n := strconv.Itoa(i + 1)
		params.Set("MailBox"+n, f.Mailbox)
		params.Set("ForwardTo"+n, f.ForwardTo)
	}
	return c.call(ctx, "namecheap.domains.dns.setEmailForwarding", params, &struct{}{})
}
