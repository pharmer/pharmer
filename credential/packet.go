package credential

type Packet struct {
	generic
}

func (c Packet) APIKey() string {
	return c.Data[PacketAPIKey]
}

func (c Packet) ProjectID() string {
	return c.Data[PacketProjectID]
}
