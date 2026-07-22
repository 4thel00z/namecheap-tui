package namecheap

import (
	"context"
	"net/url"
	"strconv"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.TransferAPI = (*Client)(nil)

type transferCreateResponse struct {
	Result struct {
		ID       string `xml:"TransferID,attr"`
		Status   string `xml:"TransferStatus,attr"`
		StatusID int    `xml:"StatusID,attr"`
	} `xml:"CommandResponse>DomainTransferCreateResult"`
}

// CreateTransfer starts an inbound transfer via namecheap.domains.transfer.create.
func (c *Client) CreateTransfer(ctx context.Context, name registrar.DomainName, eppCode string, years int) (registrar.Transfer, error) {
	params := url.Values{}
	params.Set("DomainName", name.String())
	params.Set("Years", strconv.Itoa(years))
	params.Set("EPPCode", eppCode)
	var resp transferCreateResponse
	if err := c.call(ctx, "namecheap.domains.transfer.create", params, &resp); err != nil {
		return registrar.Transfer{}, err
	}
	return registrar.Transfer{
		ID: resp.Result.ID, Name: name.String(),
		Status: resp.Result.Status, StatusID: resp.Result.StatusID,
	}, nil
}

type transferStatusResponse struct {
	Result struct {
		ID       string `xml:"TransferID,attr"`
		Status   string `xml:"Status,attr"`
		StatusID int    `xml:"StatusID,attr"`
	} `xml:"CommandResponse>DomainTransferGetStatusResult"`
}

// TransferStatus fetches namecheap.domains.transfer.getStatus.
func (c *Client) TransferStatus(ctx context.Context, id string) (registrar.Transfer, error) {
	params := url.Values{}
	params.Set("TransferID", id)
	var resp transferStatusResponse
	if err := c.call(ctx, "namecheap.domains.transfer.getStatus", params, &resp); err != nil {
		return registrar.Transfer{}, err
	}
	return registrar.Transfer{
		ID: id, Status: resp.Result.Status, StatusID: resp.Result.StatusID,
	}, nil
}

type transferListResponse struct {
	Transfers []struct {
		ID       string `xml:"ID,attr"`
		Name     string `xml:"DomainName,attr"`
		Status   string `xml:"Status,attr"`
		StatusID int    `xml:"StatusID,attr"`
		Date     string `xml:"Date,attr"`
	} `xml:"CommandResponse>TransferGetListResult>Transfer"`
}

// ListTransfers fetches namecheap.domains.transfer.getList.
func (c *Client) ListTransfers(ctx context.Context) ([]registrar.Transfer, error) {
	params := url.Values{}
	params.Set("PageSize", "100")
	var resp transferListResponse
	if err := c.call(ctx, "namecheap.domains.transfer.getList", params, &resp); err != nil {
		return nil, err
	}
	out := make([]registrar.Transfer, len(resp.Transfers))
	for i, t := range resp.Transfers {
		out[i] = registrar.Transfer{
			ID: t.ID, Name: t.Name, Status: t.Status,
			StatusID: t.StatusID, Date: t.Date,
		}
	}
	return out, nil
}

// ResubmitTransfer retries a failed transfer via
// namecheap.domains.transfer.updateStatus.
func (c *Client) ResubmitTransfer(ctx context.Context, id string) error {
	params := url.Values{}
	params.Set("TransferID", id)
	params.Set("Resubmit", "true")
	return c.call(ctx, "namecheap.domains.transfer.updateStatus", params, &struct{}{})
}
