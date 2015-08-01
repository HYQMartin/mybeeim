package main

import (
	pro "beeim/business/businessProtocol"
	"beeim/s2c"
	"code.google.com/p/goprotobuf/proto"
	"fmt"
	"net"
	"time"
)

//return packet
var (
	ReturnPacket = &pro.Command{
		Random:    proto.uint32(0),
		Groupid:   proto.uint64(0),
		Message:   proto.String("0"),
		Result:    proto.Int32(0),
		ResultDes: proto.String("0"),
	}
)

// packet type 101
var (
	CreateGroup = &pro.Command{
		Random: proto.Uint32(0),
		Uid:    proto.Uint64(0),
	}
)

// packet type 102
var (
	PushMessage = &pro.Command{
		Random:  proto.Uint32(0),
		Uid:     proto.Uint64(0),
		Groupid: proto.uint64(0),
		Message: proto.String("0"),
	}
)

// packet type 103
var (
	LeaveGroup = &pro.Command{
		Random:  proto.Uint32(0),
		Uid:     proto.Uint64(0),
		Groupid: proto.uint64(0),
	}
)

// packet type 104
var (
	CloseDialog = &pro.Command{
		Random:  proto.Uint32(0),
		Groupid: proto.uint64(0),
	}
)

// packet type 105
var (
	LogIn = &pro.Command{
		Random: proto.Uint32(0),
		Uid:    proto.Uint64(0),
	}
)

// packet type 106
var (
	LogOut = &pro.Command{
		Random: proto.Uint32(0),
		Uid:    proto.Uint64(0),
	}
)

// packet type 107
var (
	InqueryHistory = &pro.Command{
		Random: proto.Uint32(0),
		Uid:    proto.Uint64(0),
	}
)

func connectToServer() net.Conn {

	var RsaPublicKey []byte
	conn, err := net.Dial("tcp", "192.168.1.104:1212")
	if err != nil {
		fmt.Println("dial failed", err)
	} else {
		fmt.Println("dial successfully", conn)
	}
	packetR := s2c.NewPacketReader(conn)
	packetR.AESKEY = []byte("AESKEY7890123456")
	packetW := s2c.NewPacketWriter(conn)
	packetW.AESKEY = []byte("AESKEY7890123456")
	//---------------------------------------------------------------------------------------------
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
	//I have receive rsa public key, send aeskey
	*ReturnPacket.Random = s2c.GenRandom()
	*ReturnPacket.Result = 1
	*ReturnPacket.ResultDesciption = "AESKEY7890123456"
	returnPacket := s2c.MakePacket(ReturnPacket, 100)
	if err := packetW.WritePacket(returnPacket, RsaPublicKey); err != nil {
		return nil
	}
	if err := packetW.Flush(); err != nil {
		fmt.Println("packetWrter flush error", err)

	}
	return conn
	//传101-107号包

	//等待各自的回馈包

}

//read
//write
func createGroup(parm1, parm2) {
/*	keyvalue[]  kv = new keyvalue[2];
	kv.put("usr_name",parm1);
	kv.put("usr_psw",parm2);

	ParmFormater pf = new ParmFormater();
	String parmStr = pf.format(kv);

	Protocol p = new Protocol();
	Request r = new Request(requestCode,parmStr);
	p.setRequest(r);

	p.setTimeout();
	try{
	p.sendRequest();
}catch(TimeoutException te){*/

}
	int resultCOde = p.getResultCode();

	if(resultCode == SUCCESS){
		String result = p.getResult();
		parseResult(result);
	}
}
func pushMessage() {

}
func leaveGroup() {

}
func closeDialog() {

}
func logIn() {

}
func logOut() {

}
func inqueryHistory() {

}
func main() {
	conns = connectToServer()
	fmt.Println(conns)
}
