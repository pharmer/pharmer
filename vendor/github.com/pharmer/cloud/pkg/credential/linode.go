package credential

type Linode struct {
	CommonSpec
}

func (c Linode) APIToken() string { return c.Data[LinodeAPIToken] }
