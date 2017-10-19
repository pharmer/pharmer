package v1alpha1

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

type FakeDomainManager struct {
}

var _ DomainManager = &FakeDomainManager{}

func (FakeDomainManager) Domain(cluster string) string                   { return "" }
func (FakeDomainManager) ExternalDomain(cluster string) string           { return "" }
func (FakeDomainManager) InternalDomain(cluster string) string           { return "" }
func (FakeDomainManager) WebhookAuthenticationURL(cluster string) string { return "" }
func (FakeDomainManager) WebhookAuthorizationURL(cluster string) string  { return "" }
func (FakeDomainManager) PublicAPIHttpEndpoint(cluster string) string    { return "" }
func (FakeDomainManager) CompassIPs() []string                           { return []string{} }
