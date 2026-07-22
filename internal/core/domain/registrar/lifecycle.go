package registrar

import (
	"errors"
	"fmt"
	"time"
)

// Contact is one WHOIS contact record.
type Contact struct {
	FirstName     string
	LastName      string
	Organization  string
	Address1      string
	Address2      string
	City          string
	StateProvince string
	PostalCode    string
	Country       string
	Phone         string // Namecheap format: +NN.NNNNNNNNNN
	Email         string
}

// Validate checks the fields Namecheap requires for every contact.
func (c Contact) Validate() error {
	required := map[string]string{
		"first name":     c.FirstName,
		"last name":      c.LastName,
		"address1":       c.Address1,
		"city":           c.City,
		"state/province": c.StateProvince,
		"postal code":    c.PostalCode,
		"country":        c.Country,
		"phone":          c.Phone,
		"email":          c.Email,
	}
	for field, v := range required {
		if v == "" {
			return fmt.Errorf("contact %s is required", field)
		}
	}
	return nil
}

// ContactSet holds the four WHOIS roles Namecheap requires.
type ContactSet struct {
	Registrant Contact
	Tech       Contact
	Admin      Contact
	AuxBilling Contact
}

// Validate checks all four roles.
func (s ContactSet) Validate() error {
	for role, c := range map[string]Contact{
		"registrant": s.Registrant, "tech": s.Tech,
		"admin": s.Admin, "aux billing": s.AuxBilling,
	} {
		if err := c.Validate(); err != nil {
			return fmt.Errorf("%s: %w", role, err)
		}
	}
	return nil
}

// UniformContacts uses one contact for all four roles.
func UniformContacts(c Contact) ContactSet {
	return ContactSet{Registrant: c, Tech: c, Admin: c, AuxBilling: c}
}

// Registration is a domain-registration request.
type Registration struct {
	Name     DomainName
	Years    int
	Contacts ContactSet
	Privacy  bool
}

// Validate checks the request before it hits the API.
func (r Registration) Validate() error {
	if r.Name.SLD == "" || r.Name.TLD == "" {
		return errors.New("domain name is required")
	}
	if r.Years < 1 || r.Years > 10 {
		return fmt.Errorf("years %d out of range 1..10", r.Years)
	}
	return r.Contacts.Validate()
}

// RegistrationResult is the outcome of domains.create.
type RegistrationResult struct {
	Domain        string
	Registered    bool
	ChargedAmount float64
	DomainID      string
	OrderID       string
	TransactionID string
}

// RenewalResult is the outcome of domains.renew.
type RenewalResult struct {
	DomainID      string
	ChargedAmount float64
	OrderID       string
	TransactionID string
	Expires       time.Time
}

// TLD describes a top-level domain Namecheap offers.
type TLD struct {
	Name         string
	MinYears     int
	MaxYears     int
	Registerable bool
}

// Transfer is an inbound domain transfer.
type Transfer struct {
	ID     string
	Name   string
	Status string
	// StatusID is Namecheap's numeric transfer status code.
	StatusID int
	Date     string
}
