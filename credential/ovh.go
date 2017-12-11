package credential

type Ovh struct {
	CommonSpec
}

func (c Ovh) Username() string { return c.Data[OvhUsername] }

func (c Ovh) Password() string { return c.Data[OvhPassword] }

func (c Ovh) TenantID() string { return c.Data[OvhTenantID] }
