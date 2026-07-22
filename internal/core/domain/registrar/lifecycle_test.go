package registrar_test

import (
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

func contact() registrar.Contact {
	return registrar.Contact{
		FirstName: "Ada", LastName: "Lovelace", Address1: "1 Analytical Way",
		City: "London", StateProvince: "LDN", PostalCode: "E1 6AN",
		Country: "GB", Phone: "+44.2071234567", Email: "ada@example.org",
	}
}

func TestContactValidate(t *testing.T) {
	if err := contact().Validate(); err != nil {
		t.Fatalf("valid contact rejected: %v", err)
	}
	c := contact()
	c.Email = ""
	if err := c.Validate(); err == nil {
		t.Error("missing email accepted")
	}
}

func TestRegistrationValidate(t *testing.T) {
	name, _ := registrar.Parse("example.com")
	reg := registrar.Registration{
		Name: name, Years: 1,
		Contacts: registrar.UniformContacts(contact()),
	}
	if err := reg.Validate(); err != nil {
		t.Fatalf("valid registration rejected: %v", err)
	}
	reg.Years = 0
	if err := reg.Validate(); err == nil {
		t.Error("zero years accepted")
	}
	reg.Years = 1
	reg.Contacts.Tech.Phone = ""
	if err := reg.Validate(); err == nil {
		t.Error("incomplete tech contact accepted")
	}
}
