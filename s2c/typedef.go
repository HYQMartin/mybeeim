package s2c

import (
	"beeim/s2c/protocol"
	"code.google.com/p/goprotobuf/proto"
)

//********Type=0: IM->Client : 发送公钥********不加密
var (
	PublicMess = &protocol.Message{
		Data: proto.String(`-----BEGIN PUBLIC KEY-----
MIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKBgQCbmNv4mBsuPb4xzuUwT2TxEbjG
va76zLrF//NRiUKV/T8mKtfv/R+Q+7FGwYkZbDNl5bO0UF0MR69u8ZE1cBAzn74z
JGqQVs9QrRgp3VTYM/9s8nOvogcz6+l1amEY+djV3iztHOxyptl1Eq8d3r82kcwf
ybxRlNMfwNhT7VpvqQIDAQAB
-----END PUBLIC KEY-----`),
	}
	PublicCMD = &protocol.Command{
		Sender:   proto.Uint64(0), //0=IM server
		Receiver: proto.Uint64(0), //Send to anyone who connect to IM server
		Message:  PublicMess,
	}
)

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

//********Type=3: IM->Client: data="yes",random********AES加密
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

//********Type=5:Client->IM:心跳包********AES加密
