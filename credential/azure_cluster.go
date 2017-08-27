package credential

type Azure struct {
	CommonSpec
}

func (c Azure) ClientID() string       { return c.Data[AzureClientID] }
func (c Azure) ClientSecret() string   { return c.Data[AzureClientSecret] }
func (c Azure) SubscriptionID() string { return c.Data[AzureSubscriptionID] }
func (c Azure) TenantID() string       { return c.Data[AzureTenantID] }
