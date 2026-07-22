package registrar_test

import (
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in      string
		sld     string
		tld     string
		wantErr bool
	}{
		{"example.com", "example", "com", false},
		{"EXAMPLE.COM", "example", "com", false},
		{"foo.co.uk", "foo", "co.uk", false},
		{"  example.com ", "example", "com", false},
		{"example", "", "", true},
		{"", "", "", true},
		{".com", "", "", true},
		{"exa mple.com", "", "", true},
	}
	for _, c := range cases {
		got, err := registrar.Parse(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("Parse(%q): want error, got %+v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q): unexpected error %v", c.in, err)
			continue
		}
		if got.SLD != c.sld || got.TLD != c.tld {
			t.Errorf("Parse(%q) = %q/%q, want %q/%q", c.in, got.SLD, got.TLD, c.sld, c.tld)
		}
	}
}

func TestDomainNameString(t *testing.T) {
	d, err := registrar.Parse("foo.co.uk")
	if err != nil {
		t.Fatal(err)
	}
	if d.String() != "foo.co.uk" {
		t.Errorf("String() = %q, want %q", d.String(), "foo.co.uk")
	}
}
