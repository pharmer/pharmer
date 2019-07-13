package credential

type Hetzner struct {
	CommonSpec
}

func (c Hetzner) Username() string { return c.Data[HertznerUsername] }
func (c Hetzner) Password() string { return c.Data[HertznerPassword] }
