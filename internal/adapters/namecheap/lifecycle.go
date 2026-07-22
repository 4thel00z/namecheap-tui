package namecheap

import (
	"context"
	"net/url"
	"strconv"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

// contactParams serializes a contact under a role prefix (e.g. "Registrant").
func contactParams(params url.Values, role string, c registrar.Contact) {
	params.Set(role+"FirstName", c.FirstName)
	params.Set(role+"LastName", c.LastName)
	if c.Organization != "" {
		params.Set(role+"OrganizationName", c.Organization)
	}
	params.Set(role+"Address1", c.Address1)
	if c.Address2 != "" {
		params.Set(role+"Address2", c.Address2)
	}
	params.Set(role+"City", c.City)
	params.Set(role+"StateProvince", c.StateProvince)
	params.Set(role+"PostalCode", c.PostalCode)
	params.Set(role+"Country", c.Country)
	params.Set(role+"Phone", c.Phone)
	params.Set(role+"EmailAddress", c.Email)
}

func contactSetParams(params url.Values, s registrar.ContactSet) {
	contactParams(params, "Registrant", s.Registrant)
	contactParams(params, "Tech", s.Tech)
	contactParams(params, "Admin", s.Admin)
	contactParams(params, "AuxBilling", s.AuxBilling)
}

type createResponse struct {
	Result struct {
		Domain        string  `xml:"Domain,attr"`
		Registered    bool    `xml:"Registered,attr"`
		ChargedAmount float64 `xml:"ChargedAmount,attr"`
		DomainID      string  `xml:"DomainID,attr"`
		OrderID       string  `xml:"OrderID,attr"`
		TransactionID string  `xml:"TransactionID,attr"`
	} `xml:"CommandResponse>DomainCreateResult"`
}

// RegisterDomain registers a new domain via namecheap.domains.create.
func (c *Client) RegisterDomain(ctx context.Context, reg registrar.Registration) (registrar.RegistrationResult, error) {
	if err := reg.Validate(); err != nil {
		return registrar.RegistrationResult{}, err
	}
	params := url.Values{}
	params.Set("DomainName", reg.Name.String())
	params.Set("Years", strconv.Itoa(reg.Years))
	contactSetParams(params, reg.Contacts)
	if reg.Privacy {
		params.Set("AddFreeWhoisguard", "yes")
		params.Set("WGEnabled", "yes")
	}
	var resp createResponse
	if err := c.call(ctx, "namecheap.domains.create", params, &resp); err != nil {
		return registrar.RegistrationResult{}, err
	}
	r := resp.Result
	return registrar.RegistrationResult{
		Domain: r.Domain, Registered: r.Registered, ChargedAmount: r.ChargedAmount,
		DomainID: r.DomainID, OrderID: r.OrderID, TransactionID: r.TransactionID,
	}, nil
}

type renewResponse struct {
	Result struct {
		DomainID      string  `xml:"DomainID,attr"`
		ChargedAmount float64 `xml:"ChargedAmount,attr"`
		OrderID       string  `xml:"OrderID,attr"`
		TransactionID string  `xml:"TransactionID,attr"`
		Details       struct {
			ExpiredDate string `xml:"ExpiredDate"`
		} `xml:"DomainDetails"`
	} `xml:"CommandResponse>DomainRenewResult"`
}

// RenewDomain extends a registration via namecheap.domains.renew.
func (c *Client) RenewDomain(ctx context.Context, name registrar.DomainName, years int) (registrar.RenewalResult, error) {
	params := url.Values{}
	params.Set("DomainName", name.String())
	params.Set("Years", strconv.Itoa(years))
	var resp renewResponse
	if err := c.call(ctx, "namecheap.domains.renew", params, &resp); err != nil {
		return registrar.RenewalResult{}, err
	}
	r := resp.Result
	return registrar.RenewalResult{
		DomainID: r.DomainID, ChargedAmount: r.ChargedAmount,
		OrderID: r.OrderID, TransactionID: r.TransactionID,
		Expires: parseDate(r.Details.ExpiredDate),
	}, nil
}

// ReactivateDomain re-activates an expired domain via namecheap.domains.reactivate.
func (c *Client) ReactivateDomain(ctx context.Context, name registrar.DomainName) error {
	params := url.Values{}
	params.Set("DomainName", name.String())
	return c.call(ctx, "namecheap.domains.reactivate", params, &struct{}{})
}

type xmlContact struct {
	FirstName     string `xml:"FirstName"`
	LastName      string `xml:"LastName"`
	Organization  string `xml:"OrganizationName"`
	Address1      string `xml:"Address1"`
	Address2      string `xml:"Address2"`
	City          string `xml:"City"`
	StateProvince string `xml:"StateProvince"`
	PostalCode    string `xml:"PostalCode"`
	Country       string `xml:"Country"`
	Phone         string `xml:"Phone"`
	Email         string `xml:"EmailAddress"`
}

func (x xmlContact) contact() registrar.Contact {
	return registrar.Contact{
		FirstName: x.FirstName, LastName: x.LastName, Organization: x.Organization,
		Address1: x.Address1, Address2: x.Address2, City: x.City,
		StateProvince: x.StateProvince, PostalCode: x.PostalCode,
		Country: x.Country, Phone: x.Phone, Email: x.Email,
	}
}

type getContactsResponse struct {
	Result struct {
		Registrant xmlContact `xml:"Registrant"`
		Tech       xmlContact `xml:"Tech"`
		Admin      xmlContact `xml:"Admin"`
		AuxBilling xmlContact `xml:"AuxBilling"`
	} `xml:"CommandResponse>DomainContactsResult"`
}

// GetContacts fetches WHOIS contacts via namecheap.domains.getContacts.
func (c *Client) GetContacts(ctx context.Context, name registrar.DomainName) (registrar.ContactSet, error) {
	params := url.Values{}
	params.Set("DomainName", name.String())
	var resp getContactsResponse
	if err := c.call(ctx, "namecheap.domains.getContacts", params, &resp); err != nil {
		return registrar.ContactSet{}, err
	}
	return registrar.ContactSet{
		Registrant: resp.Result.Registrant.contact(),
		Tech:       resp.Result.Tech.contact(),
		Admin:      resp.Result.Admin.contact(),
		AuxBilling: resp.Result.AuxBilling.contact(),
	}, nil
}

// SetContacts replaces WHOIS contacts via namecheap.domains.setContacts.
func (c *Client) SetContacts(ctx context.Context, name registrar.DomainName, contacts registrar.ContactSet) error {
	if err := contacts.Validate(); err != nil {
		return err
	}
	params := url.Values{}
	params.Set("DomainName", name.String())
	contactSetParams(params, contacts)
	return c.call(ctx, "namecheap.domains.setContacts", params, &struct{}{})
}

type lockResponse struct {
	Result struct {
		Status bool `xml:"RegistrarLockStatus,attr"`
	} `xml:"CommandResponse>DomainGetRegistrarLockResult"`
}

// LockStatus reports the registrar lock via namecheap.domains.getRegistrarLock.
func (c *Client) LockStatus(ctx context.Context, name registrar.DomainName) (bool, error) {
	params := url.Values{}
	params.Set("DomainName", name.String())
	var resp lockResponse
	if err := c.call(ctx, "namecheap.domains.getRegistrarLock", params, &resp); err != nil {
		return false, err
	}
	return resp.Result.Status, nil
}

// SetLock toggles the registrar lock via namecheap.domains.setRegistrarLock.
func (c *Client) SetLock(ctx context.Context, name registrar.DomainName, locked bool) error {
	params := url.Values{}
	params.Set("DomainName", name.String())
	action := "UNLOCK"
	if locked {
		action = "LOCK"
	}
	params.Set("LockAction", action)
	return c.call(ctx, "namecheap.domains.setRegistrarLock", params, &struct{}{})
}

type tldListResponse struct {
	TLDs []struct {
		Name         string `xml:"Name,attr"`
		MinYears     int    `xml:"MinRegisterYears,attr"`
		MaxYears     int    `xml:"MaxRegisterYears,attr"`
		Registerable bool   `xml:"IsApiRegisterable,attr"`
	} `xml:"CommandResponse>Tlds>Tld"`
}

// TLDs lists offered TLDs via namecheap.domains.getTldList.
func (c *Client) TLDs(ctx context.Context) ([]registrar.TLD, error) {
	var resp tldListResponse
	if err := c.call(ctx, "namecheap.domains.getTldList", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]registrar.TLD, len(resp.TLDs))
	for i, t := range resp.TLDs {
		out[i] = registrar.TLD{
			Name: t.Name, MinYears: t.MinYears, MaxYears: t.MaxYears,
			Registerable: t.Registerable,
		}
	}
	return out, nil
}
