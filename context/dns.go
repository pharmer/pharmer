package context

import "github.com/appscode/go-dns/provider"

type NullDNSProvider struct {
}

var _ provider.Provider = &NullDNSProvider{}

func (NullDNSProvider) EnsureARecord(domain string, ip string) error { return nil }
func (NullDNSProvider) DeleteARecords(domain string) error           { return nil }
