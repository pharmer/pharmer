package credential

type Vultr struct {
	CommonSpec
}

func (c Vultr) Token() string { return c.Data[VultrAPIToken] }
