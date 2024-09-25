package proxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"raft-kv-service/mylog"
	"raft-kv-service/proxy/siface"
	"raft-kv-service/proxy/utils"
)

const requestLen uint32 = 20

type ClientDataPack struct {
}

func (dp *ClientDataPack) GetHeadLen() uint32 {
	return requestLen
}

// 封包
func (dp *ClientDataPack) Pack(msg siface.IMessage) ([]byte, error) {
	cliMsg, ok := msg.(*ClientMessage)
	if !ok {
		mylog.DPrintln("msg type not support")
		return nil, errors.New("msg type not support, only support [ClientMessage]")
	}
	// 1. 创建一个存放bytes字节的缓冲
	dataBuff := bytes.NewBuffer([]byte{})

	// 将 MsgId 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, cliMsg.GetMsgId()); err != nil {
		return nil, err
	}
	// 将 dataLen 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, cliMsg.GetDataLen()); err != nil {
		return nil, err
	}

	// 将 IncrId 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, cliMsg.GetIncrId()); err != nil {
		return nil, err
	}

	// 将 clerkId 写入databuff
	if err := binary.Write(dataBuff, binary.LittleEndian, cliMsg.GetClerkId()); err != nil {
		return nil, err
	}

	// 将 data 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, cliMsg.GetData()); err != nil {
		return nil, err
	}
	return dataBuff.Bytes(), nil
}

// 拆包
func (dp *ClientDataPack) Unpack(data []byte) (siface.IMessage, error) {
	// 创建一个从输入二进制数据的ioreader
	dataReader := bytes.NewReader(data)

	msg := &ClientMessage{}

	// 读MsgId
	if err := binary.Read(dataReader, binary.LittleEndian, &msg.OpId); err != nil {
		mylog.DPrintln("read id error: ", err)
		return nil, err
	}

	// 读dataLen
	if err := binary.Read(dataReader, binary.LittleEndian, &msg.DataLen); err != nil {
		mylog.DPrintln("read datalen error: ", err)
		return nil, err
	}

	// 读incrId
	if err := binary.Read(dataReader, binary.LittleEndian, &msg.IncrId); err != nil {
		mylog.DPrintln("read incrId error: ", err)
		return nil, err
	}

	if err := binary.Read(dataReader, binary.LittleEndian, &msg.ClerkId); err != nil {
		mylog.DPrintln("read clerkId error: ", err)
		return nil, err
	}

	// 判断datalen是否超过所允许的最大包长度
	if utils.GlobalObject.MaxPackageSize > 0 && msg.DataLen > utils.GlobalObject.MaxPackageSize {
		return nil, errors.New("too Large msg data recv")
	}
	return msg, nil
}

func NewClientDataPack() *ClientDataPack {
	return &ClientDataPack{}
}
