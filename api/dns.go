package api

import "github.com/appscode/go-dns/provider"

type FakeDNSProvider struct {
}

var _ provider.Provider = &FakeDNSProvider{}

func (FakeDNSProvider) EnsureARecord(domain string, ip string) error { return nil }
func (FakeDNSProvider) DeleteARecord(domain string, ip string) error { return nil }
func (FakeDNSProvider) DeleteARecords(domain string) error           { return nil }
