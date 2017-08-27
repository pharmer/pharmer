package credential

type DigitalOcean struct {
	generic
}

func (c DigitalOcean) Token() string { return c.Data[DigitalOceanToken] }
