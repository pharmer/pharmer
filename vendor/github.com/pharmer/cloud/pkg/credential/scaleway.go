package credential

type Scaleway struct {
	CommonSpec
}

func (c Scaleway) Organization() string { return c.Data[ScalewayOrganization] }
func (c Scaleway) Token() string        { return c.Data[ScalewayToken] }
