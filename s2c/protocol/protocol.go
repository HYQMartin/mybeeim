// Code generated by protoc-gen-go.
// source: trans.proto
// DO NOT EDIT!

/*
Package protocol is a generated protocol buffer package.

It is generated from these files:
	trans.proto

It has these top-level messages:
	Command
	Message
*/
package protocol

import proto "code.google.com/p/goprotobuf/proto"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type Command struct {
	Sender           *uint64  `protobuf:"varint,1,opt,name=sender" json:"sender,omitempty"`
	Receiver         *uint64  `protobuf:"varint,2,opt,name=receiver" json:"receiver,omitempty"`
	Message          *Message `protobuf:"bytes,3,opt,name=message" json:"message,omitempty"`
	XXX_unrecognized []byte   `json:"-"`
}

func (m *Command) Reset()         { *m = Command{} }
func (m *Command) String() string { return proto.CompactTextString(m) }
func (*Command) ProtoMessage()    {}

func (m *Command) GetSender() uint64 {
	if m != nil && m.Sender != nil {
		return *m.Sender
	}
	return 0
}

func (m *Command) GetReceiver() uint64 {
	if m != nil && m.Receiver != nil {
		return *m.Receiver
	}
	return 0
}

func (m *Command) GetMessage() *Message {
	if m != nil {
		return m.Message
	}
	return nil
}

type Message struct {
	Data             *string `protobuf:"bytes,1,opt,name=data" json:"data,omitempty"`
	Name             *string `protobuf:"bytes,2,opt,name=name" json:"name,omitempty"`
	Password         *string `protobuf:"bytes,3,opt" json:"Password,omitempty"`
	Random           *uint32 `protobuf:"varint,4,opt,name=random" json:"random,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}

func (m *Message) GetData() string {
	if m != nil && m.Data != nil {
		return *m.Data
	}
	return ""
}

func (m *Message) GetName() string {
	if m != nil && m.Name != nil {
		return *m.Name
	}
	return ""
}

func (m *Message) GetPassword() string {
	if m != nil && m.Password != nil {
		return *m.Password
	}
	return ""
}

func (m *Message) GetRandom() uint32 {
	if m != nil && m.Random != nil {
		return *m.Random
	}
	return 0
}

func init() {
}
