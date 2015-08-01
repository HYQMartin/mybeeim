/*
  4byte(packagehead)+4byte(packagetype)+4byte(packagelength)+data(protobuf)+4byte(CRC32)
  包格式
*/
package s2c

import (
	"beeim/s2c/protocol"
	"beeim/uitl/aes"
	"beeim/uitl/rsa"
	"bufio"
	"code.google.com/p/goprotobuf/proto"
	"encoding/binary"
	"errors"
	//"fmt"
	"hash/crc32"
	"net"
	//"time"
)

const (
	Packethead = 0x56565656
	//PacketBufSize   = 1024
	MaxPacketLength = 0x00ffffff
)

//var Packethead int
var PacketBufSize int

//var MaxPacketLength int

var (
	ErrorMysql        = errors.New("mysql err!")
	ErrorDataTransfer = errors.New("packet data protobuf transfer error!") //protobuf解析出错
	ErrorDataLength   = errors.New("packet was too long!")                 //data长度过长
	ErrorPacketHead   = errors.New("packet error!")                        //头错
	ErrorDataCRC      = errors.New("CRC32 error!")
	ErrorGetOneError  = errors.New("GetOne from FeedBackInServerTable error!")
)

//------------------------------------------
//********PacketReader 属性********
type PacketReader struct {
	br     *bufio.Reader
	AESKEY []byte
}

//********PacketReader 方法********
//********读取包头********
func (p *PacketReader) readHead() (err error) {
	bufhead := make([]byte, 4)
	hasRead := int(0)
	for {

		n, err := p.br.Read(bufhead[hasRead:])

		if err != nil {
			ImServerLog("Read head error : ", err)
			return err
		}
		hasRead += n
		if hasRead >= len(bufhead) {
			break
		}
	}
	packethead := binary.BigEndian.Uint32(bufhead)
	if packethead != Packethead {
		return ErrorDataTransfer
	}
	return nil
}

//********读取包类型*********
func (p *PacketReader) readType() (packetType uint32, err error) {
	bufType := make([]byte, 4)
	hasRead := int(0)
	for {
		n, err := p.br.Read(bufType[hasRead:])
		if err != nil {
			ImServerLog("Read type error: ", err)
			return 0, err
		}
		hasRead += n
		if hasRead >= len(bufType) {
			break
		}
	}
	packetType = binary.BigEndian.Uint32(bufType)
	return packetType, nil
}

//********读取包长度********
func (p *PacketReader) readLength() (packetSize uint32, err error) {
	bufLength := make([]byte, 4)
	hasRead := int(0)
	for {
		n, err := p.br.Read(bufLength[hasRead:])
		if err != nil {
			ImServerLog("Read length error : ", err)
			return 0, err
		}
		hasRead += n
		if hasRead >= len(bufLength) {
			break
		}
	}
	packetSize = binary.BigEndian.Uint32(bufLength)
	return packetSize, nil
}

//********读取包数据********
func (p *PacketReader) readData(data []byte) error {
	hasRead := uint32(0)
	for {
		n, err := p.br.Read(data[hasRead:])
		if err != nil {
			ImServerLog("Read data error : ", err)
			return err
		}
		hasRead += uint32(n)
		if hasRead >= uint32(len(data)) {
			break
		}
	}
	return nil
}

//********检验包数据********
func (p *PacketReader) checkData(data []byte) error {
	bufCRC := make([]byte, 4)
	hasRead := int(0)
	for {
		n, err := p.br.Read(bufCRC[hasRead:])
		if err != nil {
			ImServerLog("Read CRC32 error : ", err)
			return err
		}
		hasRead += n
		if hasRead >= len(bufCRC) {
			break
		}
	}
	bufCRC32 := binary.BigEndian.Uint32(bufCRC)
	srcCRC := crc32.ChecksumIEEE(data)
	if bufCRC32 != srcCRC {
		ImServerLog("The packet CRC32 is not correct.")
		return ErrorDataCRC
	}
	return nil
}

//********读取整个包********
func (p *PacketReader) ReadPacket() (Packet *Packet, err error) {
	newpacket := NewPacket()
	err = p.readHead()
	if err != nil {
		return nil, err
	}
	packetType, err := p.readType()
	if err != nil {
		return nil, err
	}
	//------------------------------------------------
	newpacket.SetType(packetType)
	packetLength, err := p.readLength()
	if err != nil {
		return nil, err
	}

	if packetLength > MaxPacketLength {
		ImServerLog("Packet is too long.")
		return nil, ErrorDataLength
	}
	packetData := make([]byte, packetLength)

	err = p.readData(packetData)
	if err != nil {
		return nil, err
	}

	if newpacket.GetType() == 0 {
		newpacket.SetData(packetData)
	} else {
		if newpacket.GetType() == 1 {
			ppacketData, err := rsa.RsaDecrypt(packetData)
			if err != nil {
				ImServerLog("rsaDecrypt err" + err.Error())
				return nil, err
			}
			newpacket.SetData(ppacketData)
		} else {
			ppacketData, err := aes.AesDecrypt(packetData, p.AESKEY)
			if err != nil {
				ImServerLog("aesDecrypt err" + err.Error())
				return nil, err
			}
			newpacket.SetData(ppacketData)
		}
	}
	err = p.checkData(packetData)
	if err != nil {
		ImServerLog("check err" + err.Error())
		return nil, ErrorDataCRC
	}
	return newpacket, nil
}

//------------------------------------------
//********PacketWriter 属性********
type PacketWriter struct {
	bw     *bufio.Writer
	AESKEY []byte
}

//********PacketWriter 方法********

//********写包********
func (w *PacketWriter) WritePacket(packet *Packet, rsaPublicKey []byte) error {
	if packet.GetType() != 0 {
		if packet.GetType() == 1 {
			rsadata, err := rsa.RsaEncrypt(packet.GetData(), rsaPublicKey)
			if err != nil {
				ImServerLog("rsaencrypt err" + err.Error())
				return err
			}
			packet.SetData(rsadata)
		} else {
			aesdata, err := aes.AesEncrypt(packet.GetData(), w.AESKEY)
			if err != nil {
				ImServerLog("aesencrypt err" + err.Error())
				return err
			}
			packet.SetData(aesdata)
		}
	}
	bufio := make([]byte, 0, PacketBufSize)
	//写包头
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, Packethead)
	bufio = append(bufio, buf...)

	//写包类型
	binary.BigEndian.PutUint32(buf, packet.GetType())
	bufio = append(bufio, buf...)

	//写数据长度
	binary.BigEndian.PutUint32(buf, uint32(len(packet.GetData())))
	bufio = append(bufio, buf...)

	//写数据
	databuf := packet.GetData()
	bufio = append(bufio, databuf...)

	//写CRC32校验
	intCRC := crc32.ChecksumIEEE(packet.GetData())
	binary.BigEndian.PutUint32(buf, intCRC)
	bufio = append(bufio, buf...)
	//发送数据
	//fmt.Println("------------------这是包的长度：", len(packet.GetData()), "---------", packet.GetData())
	_, err := w.bw.Write(bufio)
	if err != nil {
		return err
	}
	err = w.bw.Flush()
	if err != nil {
		return err
	}
	return nil
}

//********缓冲清除********
/*func (w *PacketWriter) Flush() error {
	return w.bw.Flush()
}
*/
//********创建新Packet包********
func NewPacket() *Packet {
	Packet := &Packet{0, nil}
	return Packet
}

//********新建PacketWriter********
func NewPacketWriter(writeConn net.Conn) *PacketWriter {
	w := &PacketWriter{
		AESKEY: make([]byte, 16),
		bw:     bufio.NewWriterSize(writeConn, PacketBufSize),
	}
	return w
}

//********新建PacketReader********
func NewPacketReader(readConn net.Conn) *PacketReader {
	r := &PacketReader{
		AESKEY: make([]byte, 16),
		br:     bufio.NewReaderSize(readConn, PacketBufSize),
	}
	//readConn.SetDeadline(time.Now().Add(time.Second * 2))
	//readConn.SetReadDeadline(time.Now().Add(time.Second * 30))
	//r.br = bufio.NewReaderSize(readConn, PacketBufSize)
	return r
}

//********拆分包结构//客户端读取公钥和random********
func GetCommand(packet *Packet) (cmd *protocol.Command, err error) {
	Cmd := &protocol.Command{}
	err = proto.Unmarshal(packet.GetData(), Cmd)
	if err != nil {
		ImServerLog("Protobuf : get command error.", err)
		return nil, err
	}
	return Cmd, nil
}

//********构建包结构********
func MakePacket(cmd *protocol.Command, Type uint32) *Packet {
	data, err := proto.Marshal(cmd)
	if err != nil {
		ImServerLog("Protobuf : make packet error.", err)
		return nil
	}
	packet := NewPacket()
	packet.SetType(Type)
	packet.SetData(data)
	return packet
}
