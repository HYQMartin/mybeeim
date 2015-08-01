package main

import (
	"beeim/s2c"
	"beeim/s2c/protocol"
	"code.google.com/p/goprotobuf/proto"
	"fmt"
	"net"
	"time"
)

var uid = uint64(444)
var receiverdialog = uint64(333444)

//客户端也需要重发接口
//********Type=1: Client->IM: name+APPKEY+AESKEY********RSA加密
var (
	requestMess = &protocol.Message{
		Data:     proto.String("Here put in AES KEY."),
		Name:     proto.String("Token"),
		Password: proto.String("Here put in APPKEY."),
		Random:   proto.Uint32(0), //give a random to server,and waiting for returnpacket which has this random.
	}
	requestCMD = &protocol.Command{
		Sender:   proto.Uint64(0), //sender's uid
		Receiver: proto.Uint64(0), //0=server
		Message:  requestMess,
	}
)

//********Type=2/4: Client<->IM: sender+receiver's name+data+random********AES加密
var (
	messMess = &protocol.Message{
		Data: proto.String("Message data"),
		//		Name:   proto.String("receiver's name/sender's name"), //Client->IM:here place in receiver's name. IM->Client:here place in sender's name
		Random: proto.Uint32(0),
	}
	messageCMD = &protocol.Command{
		Sender:   proto.Uint64(0),
		Receiver: proto.Uint64(0), //0<=> server
		Message:  messMess,
	}
)

//********Type=3: IM<->Client: data="yes",random********AES加密
var (
	returnMess = &protocol.Message{
		Data:   proto.String("Here push word as Received or Correct"), //Server has receive such a packet.
		Random: proto.Uint32(0),                                       //回馈的时候 random唯一标识收到的包

	}
	returnCMD = &protocol.Command{
		Sender:   proto.Uint64(0), //0<=> server
		Receiver: proto.Uint64(0), // here place in Client's uid
		Message:  returnMess,
	}
)

func connectToServer() net.Conn {

	var RsaPublicKey []byte
	conn, err := net.Dial("tcp", "192.168.1.106:1212")
	//conn, err := net.Dial("tcp", "222.16.62.159:1212")

	if err != nil {
		fmt.Println("dial failed", err)
	} else {
		fmt.Println("dial successfully", conn)
	}
	packetR := s2c.NewPacketReader(conn)
	packetR.AESKEY = []byte("AESKEYooooaaaaab")
	packetW := s2c.NewPacketWriter(conn)
	packetW.AESKEY = []byte("AESKEYooooaaaaab")
	var publicPacket s2c.Packet
	//first step,dial and receive a public key
	for {
		//fmt.Println("------------------------------------", i)
		if publicKeyPacket, err := packetR.ReadPacket(); err == nil {
			if publicKeyPacket.GetType() == 0 {
				publicPacket = *publicKeyPacket
				break
			}
		} else {
			fmt.Println("read type 0 packet err", err)
			return nil
		}
	}
	//get public key from the publicpacket
	publicCMD, err := s2c.GetCommand(&publicPacket)
	if err != nil {
		fmt.Println("publicPacket getcommand err", err)
	}
	RsaPublicKey = []byte(publicCMD.Message.GetData())
	//-------------------------
	//fmt.Println("packet data", publicPacket.GetData())
	//I have receive rsa public key, send app key next

	*requestCMD.Sender = uid
	*requestCMD.Receiver = 0
	*requestCMD.Message.Data = "AESKEYooooaaaaab"
	*requestCMD.Message.Name = "tokenismartin"
	*requestCMD.Message.Password = "appkeyismartin"
	*requestCMD.Message.Random = s2c.GenRandom()
	requestPacket := s2c.MakePacket(requestCMD, 1)
	if err := packetW.WritePacket(requestPacket, RsaPublicKey); err != nil {
		return nil
	}
	if err := packetW.Flush(); err != nil {
		fmt.Println("packetWrter flush error", err)

	}
	//has sent type 1 packet and wait for type 3 packet
	var getReturnMessage string
	for {
		//fmt.Println("------------------------------------", i)
		if returnPacket, err := packetR.ReadPacket(); err == nil {
			if returnPacket.GetType() == 3 {
				returnCMD, err := s2c.GetCommand(returnPacket)
				getReturnMessage = returnCMD.Message.GetData()
				if err != nil {
					fmt.Println("type 3 packet getcommand err", err)
				}
				fmt.Println("this is returnCMD", returnCMD)
				break
			}
		} else {
			fmt.Println("read type 3 packet err", err)
			return nil
		}
	}
	if getReturnMessage != "1:Connect successfully." {
		return nil
	} else {
		//log in successfully, can send type 2,3,4,5 packet
		//now send type 5 packet per 5seconds
		go func() {
			for {
				time.Sleep(time.Second * 10)
				var heartPacket s2c.Packet
				heartPacket.SetType(5)
				heartPacket.SetData([]byte("This is heart packet."))
				if err := packetW.WritePacket(&heartPacket, nil); err != nil {
					return
				}
				if err := packetW.Flush(); err != nil {
					fmt.Println("packetWrter flush error", err)
					return
				}
			}
		}()
	}
	return conn
}

//读出传过来的包，读sender，data

func read(conn net.Conn) {

	packetR := s2c.NewPacketReader(conn)
	packetR.AESKEY = []byte("AESKEYooooaaaaab")
	packetW := s2c.NewPacketWriter(conn)
	packetW.AESKEY = []byte("AESKEYooooaaaaab")
	for {
		var readPacket s2c.Packet
		if readpacket, err := packetR.ReadPacket(); err == nil {
			readPacket = *readpacket
		} else {
			fmt.Println("read  -- packet err", err)
			return
		}
		readPacketCMD, err := s2c.GetCommand(&readPacket)
		fmt.Println("-----------read packet-------------", readPacket, readPacketCMD)

		if err != nil {
			fmt.Println("----readPackt getcommand err", err)
		}
		fmt.Println("-------------client receive a packet as----------------------- ", readPacket.GetType(), readPacketCMD)
		//一旦收到二号或者四号包则返回一个三号回馈包
		if readPacket.GetType() == 2 || readPacket.GetType() == 4 {
			fmt.Println("++++++++++++++++++++++++++++++++++++++++++")
			*returnCMD.Sender = readPacketCMD.GetReceiver()
			*returnCMD.Receiver = 0
			*returnCMD.Message.Random = readPacketCMD.Message.GetRandom()
			*returnCMD.Message.Data = "client receive a type 2and4 successfully"
			returnPacket := s2c.MakePacket(returnCMD, 3)
			if err := packetW.WritePacket(returnPacket, nil); err != nil {
				return
			}
			if err := packetW.Flush(); err != nil {
				fmt.Println("read packet and return packet 3 flush error", err)
				return
			}
		}

	}
}

//写包，写receiver, message  这里name是receiver name
func write(conn net.Conn) {
	for {
		time.Sleep(time.Second * 1)
		packetW := s2c.NewPacketWriter(conn)
		packetW.AESKEY = []byte("AESKEYooooaaaaab")
		*messageCMD.Message.Data = "hello world"
		*messageCMD.Message.Random = s2c.GenRandom() //这里要改成随机数
		*messageCMD.Sender = uint64(uid)
		*messageCMD.Receiver = uint64(receiverdialog) // 发name过去  让服务器自己找receiver uid
		messagePacket := s2c.MakePacket(messageCMD, 2)
		fmt.Println("messagecmd", messagePacket.GetType(), messageCMD)
		if err := packetW.WritePacket(messagePacket, nil); err != nil {
			fmt.Println("Fail to write a message packet ")
			conn.Close()
			return
		}
		if err := packetW.Flush(); err != nil {
			fmt.Println("packetWrter flush error", err)
			conn.Close()
			return
		}
		time.Sleep(time.Second * 1)
	}
}
func main() {
	conn := connectToServer()
	fmt.Println(conn)
	go write(conn)
	go read(conn)
	time.Sleep(time.Minute * 60)
}
