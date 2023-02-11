package main

import (
	"bytes"
	"encoding/binary"
)

type Payload struct {
	// send multiple packets in case there's a packet loss
	Index uint32
	Total uint32
	// test MTU
	Current uint32
	Min     uint32
	Max     uint32
}

const (
	// payload header size is 20
	payloadHeaderSize = uint32(20)
	// udp header size is 8
	udpHeaderSize = uint32(8)
	// ip header size is 20
	ipHeaderSize = uint32(20)
	// link header size is 14
	linkHeaderSize = uint32(14)

	basicHeaderSize = udpHeaderSize + ipHeaderSize
	headerSize      = payloadHeaderSize + basicHeaderSize
)

func (p *Payload) Encode() []byte {
	buff := bytes.NewBuffer([]byte{})
	_ = binary.Write(buff, binary.BigEndian, p.Index)
	_ = binary.Write(buff, binary.BigEndian, p.Total)
	_ = binary.Write(buff, binary.BigEndian, p.Current)
	_ = binary.Write(buff, binary.BigEndian, p.Min)
	_ = binary.Write(buff, binary.BigEndian, p.Max)
	_ = binary.Write(buff, binary.BigEndian, make([]byte, p.Current-headerSize))

	return buff.Bytes()
}

func (p *Payload) Decode(data []byte) {
	if p == nil {
		panic("cannot decode to nil")
	}
	buff := bytes.NewReader(data)
	_ = binary.Read(buff, binary.BigEndian, &p.Index)
	_ = binary.Read(buff, binary.BigEndian, &p.Total)
	_ = binary.Read(buff, binary.BigEndian, &p.Current)
	_ = binary.Read(buff, binary.BigEndian, &p.Min)
	_ = binary.Read(buff, binary.BigEndian, &p.Max)
}
