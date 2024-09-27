package proxy

import (
	"bytes"
	"encoding/binary"
	"raft-kv-service/mylog"
)

type Body struct {
	KeyLen   uint32
	KeyValue []byte
}

func (b *Body) GetKeyLen() uint32 {
	return b.KeyLen
}

func (b *Body) SetKeyLen(keyLen uint32) {
	b.KeyLen = keyLen
}

func (b *Body) GetKeyValue() []byte {
	return b.KeyValue
}

func (b *Body) SetKeyValue(kv []byte) {
	b.KeyValue = kv
}

func (b *Body) GetKey() string {
	return string(b.KeyValue[:b.KeyLen])
}

func (b *Body) GetValue() string {
	return string(b.KeyValue[b.KeyLen:])
}

type BodyPack struct {
}

// 封包
func (dp *BodyPack) Pack(body Body) ([]byte, error) {
	// 1. 创建一个存放bytes字节的缓冲
	dataBuff := bytes.NewBuffer([]byte{})
	// 将 keyLen 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, body.GetKeyLen()); err != nil {
		return nil, err
	}
	// 将 key&value 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, body.GetKeyValue()); err != nil {
		return nil, err
	}
	return dataBuff.Bytes(), nil
}

// 拆包
func (dp *BodyPack) Unpack(data []byte) (*Body, error) {
	// 创建一个从输入二进制数据的ioreader
	dataReader := bytes.NewReader(data)

	body := &Body{}

	// 读KeyLen
	if err := binary.Read(dataReader, binary.LittleEndian, &body.KeyLen); err != nil {
		mylog.DPrintln("read keyLen error: ", err)
		return nil, err
	}
	body.KeyValue = data[4:]
	// if err := binary.Read(dataReader, binary.LittleEndian, &body.KeyValue); err != nil {
	// 	mylog.DPrintln("read key&value error: ", err)
	// }
	return body, nil
}

func NewBodyPack() *BodyPack {
	return &BodyPack{}
}
