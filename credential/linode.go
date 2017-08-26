package credential

type Linode struct {
	generic
}

func (c Linode) APIToken() string {
	return c.Data[LinodeAPIToken]
}
