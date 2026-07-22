package namecheap

import (
	"context"
	"net/url"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.AccountAPI = (*Client)(nil)

// Balances fetches namecheap.users.getBalances.
func (c *Client) Balances(ctx context.Context) (account.Balance, error) {
	var resp struct {
		Result struct {
			Currency     string  `xml:"Currency,attr"`
			Available    float64 `xml:"AvailableBalance,attr"`
			Total        float64 `xml:"AccountBalance,attr"`
			Earned       float64 `xml:"EarnedAmount,attr"`
			Withdrawable float64 `xml:"WithdrawableAmount,attr"`
		} `xml:"CommandResponse>UserGetBalancesResult"`
	}
	if err := c.call(ctx, "namecheap.users.getBalances", nil, &resp); err != nil {
		return account.Balance{}, err
	}
	r := resp.Result
	return account.Balance{
		Currency: r.Currency, Available: r.Available, Total: r.Total,
		Earned: r.Earned, Withdrawable: r.Withdrawable,
	}, nil
}

// Pricing fetches namecheap.users.getPricing for a product type
// (DOMAIN, SSLCERTIFICATE, WHOISGUARD), optionally narrowed by category
// (REGISTER, RENEW, …) and product name (com, positivessl, …).
func (c *Client) Pricing(ctx context.Context, productType, category, product string) ([]account.Price, error) {
	params := url.Values{}
	params.Set("ProductType", productType)
	if category != "" {
		params.Set("ProductCategory", category)
	}
	if product != "" {
		params.Set("ProductName", product)
	}
	var resp struct {
		Types []struct {
			Categories []struct {
				Name     string `xml:"Name,attr"`
				Products []struct {
					Name   string `xml:"Name,attr"`
					Prices []struct {
						Duration     int     `xml:"Duration,attr"`
						DurationType string  `xml:"DurationType,attr"`
						Regular      float64 `xml:"RegularPrice,attr"`
						Yours        float64 `xml:"YourPrice,attr"`
						Currency     string  `xml:"Currency,attr"`
					} `xml:"Price"`
				} `xml:"Product"`
			} `xml:"ProductCategory"`
		} `xml:"CommandResponse>UserGetPricingResult>ProductType"`
	}
	if err := c.call(ctx, "namecheap.users.getPricing", params, &resp); err != nil {
		return nil, err
	}
	var out []account.Price
	for _, t := range resp.Types {
		for _, cat := range t.Categories {
			for _, p := range cat.Products {
				for _, price := range p.Prices {
					out = append(out, account.Price{
						Product: p.Name, Category: cat.Name,
						Duration: price.Duration, DurationType: price.DurationType,
						Regular: price.Regular, Yours: price.Yours, Currency: price.Currency,
					})
				}
			}
		}
	}
	return out, nil
}

// ListAddresses fetches namecheap.users.address.getList.
func (c *Client) ListAddresses(ctx context.Context) ([]account.Address, error) {
	var resp struct {
		Entries []struct {
			ID      string `xml:"AddressId,attr"`
			Name    string `xml:"AddressName,attr"`
			Default bool   `xml:"IsDefault,attr"`
		} `xml:"CommandResponse>AddressGetListResult>List"`
	}
	if err := c.call(ctx, "namecheap.users.address.getList", nil, &resp); err != nil {
		return nil, err
	}
	out := make([]account.Address, len(resp.Entries))
	for i, e := range resp.Entries {
		out[i] = account.Address{ID: e.ID, Name: e.Name, Default: e.Default}
	}
	return out, nil
}

// GetAddress fetches namecheap.users.address.getInfo.
func (c *Client) GetAddress(ctx context.Context, id string) (account.Address, error) {
	params := url.Values{}
	params.Set("AddressId", id)
	var resp struct {
		Result struct {
			xmlContact        // contact fields flattened into the result element
			ID         string `xml:"AddressId"`
			Name       string `xml:"AddressName"`
			Zip        string `xml:"Zip"`
		} `xml:"CommandResponse>GetAddressInfoResult"`
	}
	if err := c.call(ctx, "namecheap.users.address.getInfo", params, &resp); err != nil {
		return account.Address{}, err
	}
	contact := resp.Result.contact()
	if contact.PostalCode == "" {
		contact.PostalCode = resp.Result.Zip
	}
	addrID := resp.Result.ID
	if addrID == "" {
		addrID = id
	}
	return account.Address{ID: addrID, Name: resp.Result.Name, Contact: contact}, nil
}

// addressParams serializes an address-book entry (note: the address API
// uses Zip instead of PostalCode).
func addressParams(a account.Address) url.Values {
	params := url.Values{}
	params.Set("AddressName", a.Name)
	c := a.Contact
	params.Set("FirstName", c.FirstName)
	params.Set("LastName", c.LastName)
	if c.Organization != "" {
		params.Set("Organization", c.Organization)
	}
	params.Set("Address1", c.Address1)
	if c.Address2 != "" {
		params.Set("Address2", c.Address2)
	}
	params.Set("City", c.City)
	params.Set("StateProvince", c.StateProvince)
	params.Set("StateProvinceChoice", "S")
	params.Set("Zip", c.PostalCode)
	params.Set("Country", c.Country)
	params.Set("Phone", c.Phone)
	params.Set("EmailAddress", c.Email)
	if a.Default {
		params.Set("DefaultYN", "1")
	}
	return params
}

// CreateAddress adds an entry via namecheap.users.address.create.
func (c *Client) CreateAddress(ctx context.Context, a account.Address) error {
	if err := a.Contact.Validate(); err != nil {
		return err
	}
	return c.call(ctx, "namecheap.users.address.create", addressParams(a), &struct{}{})
}

// UpdateAddress updates an entry via namecheap.users.address.update.
func (c *Client) UpdateAddress(ctx context.Context, a account.Address) error {
	if err := a.Contact.Validate(); err != nil {
		return err
	}
	params := addressParams(a)
	params.Set("AddressId", a.ID)
	return c.call(ctx, "namecheap.users.address.update", params, &struct{}{})
}

// DeleteAddress removes an entry via namecheap.users.address.delete.
func (c *Client) DeleteAddress(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("AddressId", id)
	return c.call(ctx, "namecheap.users.address.delete", params, &struct{}{})
}

// SetDefaultAddress marks an entry default via namecheap.users.address.setDefault.
func (c *Client) SetDefaultAddress(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("AddressId", id)
	return c.call(ctx, "namecheap.users.address.setDefault", params, &struct{}{})
}
