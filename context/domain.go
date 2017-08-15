package context

// string, bool
type DomainManager interface {
	Domain(cluster string) string
	ExternalDomain(cluster string) string
	InternalDomain(cluster string) string
	WebhookAuthenticationURL(cluster string) string
	WebhookAuthorizationURL(cluster string) string
	PublicAPIHttpEndpoint(cluster string) string
	CompassIPs() []string
}

type NullDomainManager struct {
}

var _ DomainManager = &NullDomainManager{}

func (NullDomainManager) Domain(cluster string) string                   { return "" }
func (NullDomainManager) ExternalDomain(cluster string) string           { return "" }
func (NullDomainManager) InternalDomain(cluster string) string           { return "" }
func (NullDomainManager) WebhookAuthenticationURL(cluster string) string { return "" }
func (NullDomainManager) WebhookAuthorizationURL(cluster string) string  { return "" }
func (NullDomainManager) PublicAPIHttpEndpoint(cluster string) string    { return "" }
func (NullDomainManager) CompassIPs() []string                           { return []string{} }
