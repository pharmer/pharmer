package credential

type Swift struct {
	CommonSpec
}

func (c Swift) Username() string      { return c.Data[SwiftUsername] }
func (c Swift) Key() string           { return c.Data[SwiftKey] }
func (c Swift) TenantName() string    { return c.Data[SwiftTenantName] }
func (c Swift) TenantAuthURL() string { return c.Data[SwiftTenantAuthURL] }
func (c Swift) Domain() string        { return c.Data[SwiftDomain] }
func (c Swift) Region() string        { return c.Data[SwiftRegion] }
func (c Swift) TenantId() string      { return c.Data[SwiftTenantId] }
func (c Swift) TenantDomain() string  { return c.Data[SwiftTenantDomain] }
func (c Swift) TrustId() string       { return c.Data[SwiftTrustId] }
func (c Swift) StorageURL() string    { return c.Data[SwiftStorageURL] }
func (c Swift) AuthToken() string     { return c.Data[SwiftAuthToken] }
