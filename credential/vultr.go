package credential

type Vultr struct {
	generic
}

func (c Vultr) Token() string { return c.Data[VultrAPIToken] }
