# ncp Phase 4 — SSL, Privacy, Account Implementation Plan

> Executed inline; conventions per Phase 2/3 plans.

**Goal:** Complete the API surface: SSL certificates, domain privacy
(whoisguard), account balances/pricing, and the address book.

## Task 1: domain types + ports

- `internal/core/domain/ssl/`: `Certificate{ID, Host, Type, Status, Years,
  Purchased, Expires, ActivationExpires, Expired}`, `Purchase{Type string,
  Years int}`, `Activation{CertificateID, CSR, WebServerType, DVMethod
  (email|http|dns), ApproverEmail}` + `Validate()`.
- `internal/core/domain/account/` additions: `Balance{Currency, Available,
  Total, Earned, Withdrawable}`, `Price{Product, Category, Duration,
  DurationType, Regular, Yours, Currency}`, `Address{ID, Name, Default,
  Contact registrar.Contact}`, `PrivacySubscription{ID, Domain, Created,
  Expires, Status}`.
- Ports: `SSLAPI` (List, Create, Activate, Info, Renew, Reissue,
  ApproverEmails, ResendApproverEmail, Revoke), `PrivacyAPI` (List, Enable,
  Disable, Renew, ChangeEmail, Assign, Unassign, Discard), `AccountAPI`
  (Balances, Pricing(productType, category, product), ListAddresses,
  GetAddress, CreateAddress, UpdateAddress, DeleteAddress,
  SetDefaultAddress).

## Task 2: namecheap adapters + fixtures

- `ssl.go`: `namecheap.ssl.getList|create|activate|getInfo|renew|reissue|
  getApproverEmailList|resendApproverEmail|revokecertificate`. Activation DV
  method maps to `ApproverEmail` / `HTTPDCValidation=true` /
  `DNSDCValidation=true`.
- `privacy.go`: `namecheap.whoisguard.getList|enable|disable|renew|
  changeemailaddress|allot|unallot|discard`.
- `users.go`: `namecheap.users.getBalances|getPricing`;
  `address.go`: `namecheap.users.address.*` (contact params reuse
  `contactParams`; plus `AddressName`, `EmailAddress`).
- Fixture per read command; param assertions for writes.

## Task 3: services + CLI + wiring

- `SSLService`, `PrivacyService`, `AccountService` (balances cached 5m,
  pricing 24h, addresses uncached).
- CLI: `ssl list|create|activate|info|renew|reissue|approvers|
  resend-approver|revoke`, `privacy list|enable|disable|renew|change-email|
  assign|unassign|discard`, `account balance|pricing`, `address
  list|add|edit|rm|default|info` (address add/edit reuse the contact wizard
  / `-f` file from Phase 3).
- Wire factories in main.go; full green; merge.
