package credential

type Scaleway struct {
	generic
}

func (c Scaleway) Organization() string {
	return c.Data[ScalewayOrganization]
}

func (c Scaleway) Token() string {
	return c.Data[ScalewayToken]
}
