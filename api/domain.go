package api

// string, bool
type DomainManager interface {
	Domain(cluster string) string
	ExternalDomain(cluster string) string
	InternalDomain(cluster string) string
}

type NullDomainManager struct {
}

var _ DomainManager = &NullDomainManager{}

func (NullDomainManager) Domain(cluster string) string         { return "" }
func (NullDomainManager) ExternalDomain(cluster string) string { return "" }
func (NullDomainManager) InternalDomain(cluster string) string { return "" }
