package credential

type Softlayer struct {
	generic
}

func (c Softlayer) Username() string { return c.Data[SoftlayerUsername] }
func (c Softlayer) APIKey() string   { return c.Data[SoftlayerAPIKey] }
