package main

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPayload_Decode(t *testing.T) {
	buff := bytes.NewBuffer([]byte{})
	index := uint32(2)
	total := uint32(3)
	current := uint32(1024)
	min := uint32(1000)
	max := uint32(1080)
	_ = binary.Write(buff, binary.BigEndian, index)
	_ = binary.Write(buff, binary.BigEndian, total)
	_ = binary.Write(buff, binary.BigEndian, current)
	_ = binary.Write(buff, binary.BigEndian, min)
	_ = binary.Write(buff, binary.BigEndian, max)
	_ = binary.Write(buff, binary.BigEndian, make([]byte, 1004))
	p := new(Payload)
	p.Decode(buff.Bytes())

	assert.EqualValues(t, index, p.Index)
	assert.EqualValues(t, total, p.Total)
	assert.EqualValues(t, current, p.Current)
	assert.EqualValues(t, min, p.Min)
	assert.EqualValues(t, max, p.Max)
}

func TestPayload_Encode(t *testing.T) {
	index := uint32(2)
	total := uint32(3)
	current := uint32(1024)
	min := uint32(1000)
	max := uint32(1080)
	p := &Payload{
		Index:   index,
		Total:   total,
		Current: current,
		Min:     min,
		Max:     max,
	}
	data := p.Encode()
	assert.Len(t, data, 996)

	buff := bytes.NewBuffer(data)
	var i uint32
	_ = binary.Read(buff, binary.BigEndian, &i)
	assert.EqualValues(t, index, i)
	_ = binary.Read(buff, binary.BigEndian, &i)
	assert.EqualValues(t, total, i)
	_ = binary.Read(buff, binary.BigEndian, &i)
	assert.EqualValues(t, current, i)
	_ = binary.Read(buff, binary.BigEndian, &i)
	assert.EqualValues(t, min, i)
	_ = binary.Read(buff, binary.BigEndian, &i)
	assert.EqualValues(t, max, i)
}
