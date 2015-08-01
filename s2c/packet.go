package s2c

//********Packet 属性********
type Packet struct {
	packetType uint32
	packetData []byte
}

//********Packet 方法********
func (p *Packet) SetType(t uint32) {
	p.packetType = t
}
func (p *Packet) GetType() (t uint32) {
	t = p.packetType
	return t
}
func (p *Packet) SetData(data []byte) {
	p.packetData = data
}
func (p *Packet) GetData() (data []byte) {
	data = p.packetData
	return data
}
