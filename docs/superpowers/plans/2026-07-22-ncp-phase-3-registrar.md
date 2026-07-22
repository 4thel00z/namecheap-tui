# ncp Phase 3 — Registrar Lifecycle Implementation Plan

> Executed inline; same conventions as Phase 2's plan (interfaces + test
> intentions; patterns established by the existing codebase).

**Goal:** Domain registration, renewal, reactivation, WHOIS contacts,
registrar lock, TLD listing, and inbound transfers.

## Task 1: registrar domain types

- Extend `internal/core/domain/registrar/`:
  - `Contact{FirstName, LastName, Organization, Address1, Address2, City,
    StateProvince, PostalCode, Country, Phone, Email}` + `Validate()`
    (required: FirstName, LastName, Address1, City, StateProvince,
    PostalCode, Country, Phone, Email).
  - `ContactSet{Registrant, Tech, Admin, AuxBilling Contact}` +
    `Validate()`; `UniformContacts(c Contact) ContactSet` helper.
  - `Registration{Name DomainName, Years int, Contacts ContactSet,
    Privacy bool}` + `Validate()` (years 1–10).
  - `RegistrationResult{Domain, Registered, ChargedAmount, DomainID,
    OrderID, TransactionID}`; `RenewalResult{DomainID, ChargedAmount,
    OrderID, TransactionID, Expires time.Time}`.
  - `TLD{Name string, MinYears, MaxYears int, Registerable bool}`.
  - `Transfer{ID, Name, Status string, StatusID int, Date string}`.

## Task 2: ports

- Extend `RegistrarAPI` with: `RegisterDomain(ctx, Registration)
  (RegistrationResult, error)`, `RenewDomain(ctx, name, years)
  (RenewalResult, error)`, `ReactivateDomain(ctx, name) error`,
  `GetContacts(ctx, name) (ContactSet, error)`, `SetContacts(ctx, name,
  ContactSet) error`, `LockStatus(ctx, name) (bool, error)`,
  `SetLock(ctx, name, locked bool) error`, `TLDs(ctx) ([]TLD, error)`.
- New `TransferAPI`: `CreateTransfer(ctx, name, eppCode string, years int)
  (Transfer, error)`, `TransferStatus(ctx, id string) (Transfer, error)`,
  `ListTransfers(ctx) ([]Transfer, error)`, `ResubmitTransfer(ctx, id) error`.

## Task 3: namecheap adapter

- `domains.go` additions: `domains.create` (indexed per-role contact params:
  Registrant/Tech/Admin/AuxBilling + FirstName…EmailAddress; AddFreeWhoisguard
  + WGEnabled when Privacy), `domains.renew`, `domains.reactivate`,
  `domains.getContacts`/`setContacts`, `domains.getRegistrarLock`/
  `setRegistrarLock` (LockAction=LOCK|UNLOCK), `domains.getTldList`.
- New `transfer.go`: `domains.transfer.create|getStatus|getList|updateStatus`
  (Resubmit=true).
- Fixtures per command; tests assert params (all four roles serialized,
  LockAction values) and response mapping.

## Task 4: services

- `DomainService` additions: `Register`, `Renew`, `Reactivate`, `Contacts`,
  `SetContacts`, `Lock`, `SetLock`, `TLDs` (TLD list cached 24h; every write
  invalidates the `domains` cache kind).
- New `TransferService`: `Create`, `Status`, `List`, `Resubmit`.
- Tests: cache invalidation after register/renew/lock, validation rejects
  bad years/contacts before hitting the API.

## Task 5: CLI

- `domains register <domain>` — huh wizard for the contact (uniform across
  roles) unless `--contacts-file` (YAML/JSON) given; `--years`, `--privacy`;
  prints charged amount.
- `domains renew <domain> --years N`, `domains reactivate <domain>`,
  `domains lock get|on|off <domain>`, `domains tlds`,
  `domains contacts get <domain>` (table/JSON/YAML) and
  `contacts set <domain> -f contacts.yaml`.
- `transfer create <domain> <epp-code> [--years]`, `transfer status <id>
  [--resubmit]`, `transfer list`.
- Contacts YAML: `{registrant:{first_name,…}, tech:{…}, admin:{…},
  aux_billing:{…}}`; a single top-level contact applies to all roles.
- Tests: flag paths with fakes (wizard path exercised manually).

## Task 6: wiring + verification + merge

- App factories for TransferService; register commands; full green; merge.
