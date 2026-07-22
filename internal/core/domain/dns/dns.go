// Package dns holds the zone aggregate: host records, staged change sets,
// email forwarding, and nameserver info.
package dns

import (
	"fmt"
	"strings"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

// RecordType is a Namecheap-supported DNS record type.
type RecordType string

const (
	A      RecordType = "A"
	AAAA   RecordType = "AAAA"
	ALIAS  RecordType = "ALIAS"
	CAA    RecordType = "CAA"
	CNAME  RecordType = "CNAME"
	MX     RecordType = "MX"
	MXE    RecordType = "MXE"
	NS     RecordType = "NS"
	TXT    RecordType = "TXT"
	URL    RecordType = "URL"
	URL301 RecordType = "URL301"
	FRAME  RecordType = "FRAME"
)

var recordTypes = map[RecordType]bool{
	A: true, AAAA: true, ALIAS: true, CAA: true, CNAME: true, MX: true,
	MXE: true, NS: true, TXT: true, URL: true, URL301: true, FRAME: true,
}

// ParseRecordType validates and normalizes a record type string.
func ParseRecordType(s string) (RecordType, error) {
	t := RecordType(strings.ToUpper(strings.TrimSpace(s)))
	if !recordTypes[t] {
		return "", fmt.Errorf("unsupported record type %q", s)
	}
	return t, nil
}

// DefaultTTL is Namecheap's default record TTL.
const DefaultTTL = 1799

// HostRecord is one DNS record in a zone.
type HostRecord struct {
	ID     string     // HostId assigned by Namecheap (informational)
	Name   string     // host part, e.g. "@", "www"
	Type   RecordType // validated record type
	Value  string     // address / target
	TTL    int        // 60..60000; 0 means DefaultTTL
	MXPref int        // MX preference, only meaningful for MX records
}

// Validate checks the record is well-formed.
func (r HostRecord) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("record name is required")
	}
	if !recordTypes[r.Type] {
		return fmt.Errorf("unsupported record type %q", r.Type)
	}
	if r.Value == "" {
		return fmt.Errorf("record value is required")
	}
	if r.TTL != 0 && (r.TTL < 60 || r.TTL > 60000) {
		return fmt.Errorf("ttl %d out of range 60..60000", r.TTL)
	}
	return nil
}

// key identifies a record for matching within a changeset.
func (r HostRecord) key() string {
	return r.Name + "\x00" + string(r.Type) + "\x00" + r.Value
}

// Zone is the full record set of one domain.
type Zone struct {
	Domain  registrar.DomainName
	Records []HostRecord
}

// RecordUpdate replaces Old with New.
type RecordUpdate struct {
	Old HostRecord
	New HostRecord
}

// ChangeSet is a staged set of zone mutations, applied atomically by Merge.
type ChangeSet struct {
	Add    []HostRecord
	Remove []HostRecord
	Update []RecordUpdate
}

// Empty reports whether the changeset stages no mutations.
func (cs ChangeSet) Empty() bool {
	return len(cs.Add) == 0 && len(cs.Remove) == 0 && len(cs.Update) == 0
}

// Merge applies cs to current and returns the resulting record set.
// Remove and Update targets must exist (matched by name+type+value);
// Add rejects exact duplicates. current is not mutated.
func Merge(current []HostRecord, cs ChangeSet) ([]HostRecord, error) {
	out := make([]HostRecord, len(current))
	copy(out, current)

	index := func() map[string]int {
		m := make(map[string]int, len(out))
		for i, r := range out {
			m[r.key()] = i
		}
		return m
	}

	for _, u := range cs.Update {
		idx := index()
		i, ok := idx[u.Old.key()]
		if !ok {
			return nil, fmt.Errorf("cannot update %s %s %s: no such record", u.Old.Type, u.Old.Name, u.Old.Value)
		}
		out[i] = u.New
	}
	for _, r := range cs.Remove {
		idx := index()
		i, ok := idx[r.key()]
		if !ok {
			return nil, fmt.Errorf("cannot remove %s %s %s: no such record", r.Type, r.Name, r.Value)
		}
		out = append(out[:i], out[i+1:]...)
	}
	for _, r := range cs.Add {
		if _, dup := index()[r.key()]; dup {
			return nil, fmt.Errorf("cannot add %s %s %s: identical record exists", r.Type, r.Name, r.Value)
		}
		out = append(out, r)
	}
	return out, nil
}

// EmailForward maps a mailbox to a destination address.
type EmailForward struct {
	Mailbox   string // local part, e.g. "info"
	ForwardTo string // destination email
}

// NameserverInfo describes which nameservers a domain uses.
type NameserverInfo struct {
	UsingOurDNS bool // true = Namecheap BasicDNS/default
	Nameservers []string
}

// RegisteredNS is a personal nameserver registered under a domain.
type RegisteredNS struct {
	Host     string
	IP       string
	Statuses []string
}
