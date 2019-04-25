package credential

type DigitalOcean struct {
	CommonSpec
}

func (c DigitalOcean) Token() string { return c.Data[DigitalOceanToken] }
