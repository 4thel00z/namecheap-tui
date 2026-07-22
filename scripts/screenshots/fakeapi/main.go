// Command fakeapi serves canned Namecheap XML responses for ncp screenshots.
package main

import (
	"fmt"
	"log"
	"net/http"
)

const envelope = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="%s">
%s
  </CommandResponse>
</ApiResponse>`

var responses = map[string]string{
	"namecheap.domains.getList": `    <DomainGetListResult>
      <Domain ID="1" Name="ravenhold.dev" User="4thel00z" Created="08/09/2021" Expires="08/09/2026" IsExpired="false" IsLocked="false" AutoRenew="false" WhoisGuard="ENABLED" IsPremium="false" IsOurDNS="true" />
      <Domain ID="2" Name="lambdafactory.io" User="4thel00z" Created="10/02/2022" Expires="10/02/2026" IsExpired="false" IsLocked="true" AutoRenew="true" WhoisGuard="ENABLED" IsPremium="false" IsOurDNS="true" />
      <Domain ID="3" Name="4thel00z.com" User="4thel00z" Created="03/14/2019" Expires="03/14/2027" IsExpired="false" IsLocked="true" AutoRenew="true" WhoisGuard="ENABLED" IsPremium="false" IsOurDNS="true" />
      <Domain ID="4" Name="moonlink.gg" User="4thel00z" Created="06/01/2023" Expires="06/01/2027" IsExpired="false" IsLocked="false" AutoRenew="false" WhoisGuard="NOTPRESENT" IsPremium="false" IsOurDNS="false" />
      <Domain ID="5" Name="konan.sh" User="4thel00z" Created="11/30/2024" Expires="11/30/2027" IsExpired="false" IsLocked="false" AutoRenew="true" WhoisGuard="NOTPRESENT" IsPremium="false" IsOurDNS="true" />
      <Domain ID="6" Name="metalbrew.dev" User="4thel00z" Created="01/15/2025" Expires="01/15/2028" IsExpired="false" IsLocked="true" AutoRenew="true" WhoisGuard="ENABLED" IsPremium="false" IsOurDNS="true" />
    </DomainGetListResult>
    <Paging>
      <TotalItems>6</TotalItems>
      <CurrentPage>1</CurrentPage>
      <PageSize>100</PageSize>
    </Paging>`,

	"namecheap.domains.check": `    <DomainCheckResult Domain="ravenhold.com" Available="true" IsPremiumName="false" PremiumRegistrationPrice="0" />
    <DomainCheckResult Domain="ravenhold.io" Available="false" IsPremiumName="false" PremiumRegistrationPrice="0" />
    <DomainCheckResult Domain="ravenhold.sh" Available="true" IsPremiumName="true" PremiumRegistrationPrice="149.00" />
    <DomainCheckResult Domain="ravenhold.ai" Available="true" IsPremiumName="false" PremiumRegistrationPrice="0" />`,

	"namecheap.domains.getInfo": `    <DomainGetInfoResult Status="Ok" ID="1" DomainName="ravenhold.dev" OwnerName="4thel00z" IsOwner="true" IsPremium="false">
      <DomainDetails>
        <CreatedDate>08/09/2021</CreatedDate>
        <ExpiredDate>08/09/2026</ExpiredDate>
        <NumYears>1</NumYears>
      </DomainDetails>
      <Whoisguard Enabled="True">
        <ID>19231</ID>
      </Whoisguard>
      <DnsDetails ProviderType="NAMECHEAP" IsUsingOurDNS="true">
        <Nameserver>dns1.registrar-servers.com</Nameserver>
        <Nameserver>dns2.registrar-servers.com</Nameserver>
      </DnsDetails>
    </DomainGetInfoResult>`,

	"namecheap.domains.dns.getHosts": `    <DomainDNSGetHostsResult Domain="ravenhold.dev" IsUsingOurDNS="true">
      <host HostId="101" Name="@" Type="A" Address="76.76.21.21" MXPref="10" TTL="1800" />
      <host HostId="102" Name="www" Type="CNAME" Address="ravenhold.dev." MXPref="10" TTL="1800" />
      <host HostId="103" Name="api" Type="A" Address="76.76.21.42" MXPref="10" TTL="300" />
      <host HostId="104" Name="@" Type="MX" Address="in1-smtp.messagingengine.com" MXPref="10" TTL="3600" />
      <host HostId="105" Name="@" Type="TXT" Address="v=spf1 include:spf.messagingengine.com ~all" MXPref="10" TTL="3600" />
      <host HostId="106" Name="status" Type="CNAME" Address="stats.uptimerobot.com." MXPref="10" TTL="1800" />
    </DomainDNSGetHostsResult>`,

	"namecheap.domains.dns.setHosts": `    <DomainDNSSetHostsResult Domain="ravenhold.dev" IsSuccess="true" />`,

	"namecheap.ssl.getList": `    <SSLListResult>
      <SSL CertificateID="91231" HostName="ravenhold.dev" SSLType="PositiveSSL" PurchaseDate="03/02/2026" ExpireDate="03/02/2027" ActivationExpireDate="" IsExpiredYN="false" Status="active" Years="1" />
      <SSL CertificateID="88412" HostName="*.lambdafactory.io" SSLType="PositiveSSL Wildcard" PurchaseDate="09/14/2025" ExpireDate="09/14/2026" ActivationExpireDate="" IsExpiredYN="false" Status="active" Years="1" />
      <SSL CertificateID="93710" HostName="4thel00z.com" SSLType="EssentialSSL" PurchaseDate="07/20/2026" ExpireDate="" ActivationExpireDate="10/20/2026" IsExpiredYN="false" Status="newpurchase" Years="1" />
    </SSLListResult>
    <Paging>
      <TotalItems>3</TotalItems>
      <CurrentPage>1</CurrentPage>
      <PageSize>100</PageSize>
    </Paging>`,

	"namecheap.domains.transfer.getList": `    <TransferGetListResult>
      <Transfer ID="55120" DomainName="moonlink.gg" User="4thel00z" TransferDate="07/18/2026" OrderID="812331" StatusID="5" Status="INPROGRESS" StatusDate="07/21/2026" Date="07/18/2026" />
      <Transfer ID="54007" DomainName="vantablack.dev" User="4thel00z" TransferDate="06/24/2026" OrderID="798102" StatusID="9" Status="COMPLETED" StatusDate="06/30/2026" Date="06/24/2026" />
    </TransferGetListResult>`,

	"namecheap.users.address.getList": `    <AddressGetListResult>
      <List AddressId="18827" AddressName="Home" IsDefault="true" />
      <List AddressId="18828" AddressName="Office" IsDefault="false" />
      <List AddressId="19102" AddressName="Registrar-Ops" IsDefault="false" />
    </AddressGetListResult>`,

	"namecheap.users.getBalances": `    <UserGetBalancesResult Currency="USD" AvailableBalance="245.50" AccountBalance="245.50" EarnedAmount="31.70" WithdrawableAmount="120.00" FundsRequiredForAutoRenew="42.96" />`,

	"namecheap.users.getPricing": `    <UserGetPricingResult>
      <ProductType Name="domains">
        <ProductCategory Name="register">
          <Product Name="com">
            <Price Duration="1" DurationType="YEAR" Price="10.28" RegularPrice="13.98" YourPrice="10.28" Currency="USD" />
          </Product>
          <Product Name="dev">
            <Price Duration="1" DurationType="YEAR" Price="12.98" RegularPrice="15.98" YourPrice="12.98" Currency="USD" />
          </Product>
          <Product Name="io">
            <Price Duration="1" DurationType="YEAR" Price="32.88" RegularPrice="39.98" YourPrice="32.88" Currency="USD" />
          </Product>
          <Product Name="sh">
            <Price Duration="1" DurationType="YEAR" Price="35.98" RegularPrice="39.98" YourPrice="35.98" Currency="USD" />
          </Product>
        </ProductCategory>
        <ProductCategory Name="renew">
          <Product Name="com">
            <Price Duration="1" DurationType="YEAR" Price="14.58" RegularPrice="15.98" YourPrice="14.58" Currency="USD" />
          </Product>
          <Product Name="dev">
            <Price Duration="1" DurationType="YEAR" Price="14.98" RegularPrice="17.98" YourPrice="14.98" Currency="USD" />
          </Product>
          <Product Name="io">
            <Price Duration="1" DurationType="YEAR" Price="36.88" RegularPrice="44.98" YourPrice="36.88" Currency="USD" />
          </Product>
          <Product Name="sh">
            <Price Duration="1" DurationType="YEAR" Price="39.98" RegularPrice="45.98" YourPrice="39.98" Currency="USD" />
          </Product>
        </ProductCategory>
      </ProductType>
    </UserGetPricingResult>`,

	"namecheap.whoisguard.getList": `    <WhoisguardGetListResult>
      <Whoisguard ID="19231" DomainName="ravenhold.dev" Created="08/09/2021" Expires="08/09/2027" Status="ENABLED" />
      <Whoisguard ID="19232" DomainName="lambdafactory.io" Created="10/02/2022" Expires="10/02/2027" Status="ENABLED" />
      <Whoisguard ID="19233" DomainName="4thel00z.com" Created="03/14/2019" Expires="03/14/2028" Status="ENABLED" />
      <Whoisguard ID="19234" DomainName="metalbrew.dev" Created="01/15/2025" Expires="01/15/2028" Status="ENABLED" />
    </WhoisguardGetListResult>
    <Paging>
      <TotalItems>4</TotalItems>
      <CurrentPage>1</CurrentPage>
      <PageSize>100</PageSize>
    </Paging>`,
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cmd := r.URL.Query().Get("Command")
		body, ok := responses[cmd]
		if !ok {
			body = "" // generic OK envelope for write commands parsed into struct{}{}
		}
		w.Header().Set("Content-Type", "application/xml")
		_, _ = fmt.Fprintf(w, envelope, cmd, body)
		log.Printf("%s", cmd)
	})
	log.Fatal(http.ListenAndServe("127.0.0.1:8811", nil))
}
