package namecheap

import (
	"context"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.NSAPI = (*Client)(nil)

// CreateNS registers a personal nameserver via namecheap.domains.ns.create.
func (c *Client) CreateNS(ctx context.Context, domain registrar.DomainName, host, ip string) error {
	params := sldTLD(domain)
	params.Set("Nameserver", host)
	params.Set("IP", ip)
	return c.call(ctx, "namecheap.domains.ns.create", params, &struct{}{})
}

// UpdateNS changes a personal nameserver's IP via namecheap.domains.ns.update.
func (c *Client) UpdateNS(ctx context.Context, domain registrar.DomainName, host, oldIP, newIP string) error {
	params := sldTLD(domain)
	params.Set("Nameserver", host)
	params.Set("OldIP", oldIP)
	params.Set("IP", newIP)
	return c.call(ctx, "namecheap.domains.ns.update", params, &struct{}{})
}

// DeleteNS removes a personal nameserver via namecheap.domains.ns.delete.
func (c *Client) DeleteNS(ctx context.Context, domain registrar.DomainName, host string) error {
	params := sldTLD(domain)
	params.Set("Nameserver", host)
	return c.call(ctx, "namecheap.domains.ns.delete", params, &struct{}{})
}

type nsInfoResponse struct {
	Result struct {
		Nameserver string   `xml:"Nameserver,attr"`
		IP         string   `xml:"IP,attr"`
		Statuses   []string `xml:"NameserverStatuses>Status"`
	} `xml:"CommandResponse>DomainNSInfoResult"`
}

// NSInfo fetches a personal nameserver's details via namecheap.domains.ns.getInfo.
func (c *Client) NSInfo(ctx context.Context, domain registrar.DomainName, host string) (dns.RegisteredNS, error) {
	params := sldTLD(domain)
	params.Set("Nameserver", host)
	var resp nsInfoResponse
	if err := c.call(ctx, "namecheap.domains.ns.getInfo", params, &resp); err != nil {
		return dns.RegisteredNS{}, err
	}
	return dns.RegisteredNS{
		Host:     resp.Result.Nameserver,
		IP:       resp.Result.IP,
		Statuses: resp.Result.Statuses,
	}, nil
}
