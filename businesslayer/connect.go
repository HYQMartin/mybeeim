package main

import (
	"beeim/s2c"
	"fmt"
	"net"
)

func main() {
	conn, err := net.Dial("tcp", "192.168.1.107:3434")
	if err != nil {
		fmt.Println("dial failed", err)
	} else {
		fmt.Println("dial successfully", conn)
	}
	var recognize s2c.Packet
	recognize.SetType(100)
	recognize.SetData([]byte("IamBussnessLayer"))

	packetR := s2c.NewPacketReader(conn)
	packetR.AESKEY = []byte("AESKEYFORBUSINES")
	packetW := s2c.NewPacketWriter(conn)
	packetW.AESKEY = []byte("AESKEYFORBUSINES")
	fmt.Println(recognize)
	if err := packetW.WritePacket(&recognize, nil); err != nil {
		return
	}
	if err := packetW.Flush(); err != nil {
		fmt.Println("packetWrter flush error", err)
		return

	}

}
