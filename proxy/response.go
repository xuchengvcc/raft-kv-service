package proxy

import (
	"bytes"
	"encoding/binary"
	"errors"
	"raft-kv-service/mylog"
	"raft-kv-service/proxy/siface"
	"raft-kv-service/proxy/utils"
)

type ServerResponse struct {
	OpId     uint32
	DataLen  uint32
	IncrId   uint64
	ClerkId  int32
	ServerId int32
	Err      uint32
	Data     []byte
}

func NewSerRespPackage(id uint32, clerkId int32, incrId uint64, serverId int32, err uint32, data []byte) *ServerResponse {
	return &ServerResponse{
		OpId:     id,
		DataLen:  uint32(len(data)),
		IncrId:   incrId,
		ClerkId:  clerkId,
		ServerId: serverId,
		Err:      err,
		Data:     data,
	}
}

// 获取消息OpId
func (m *ServerResponse) GetMsgId() uint32 {
	return m.OpId
}

// 获取消息长度
func (m *ServerResponse) GetDataLen() uint32 {
	return m.DataLen
}

// 获取客户端Id
func (m *ServerResponse) GetClerkId() int32 {
	return m.ClerkId
}

func (m *ServerResponse) GetServerId() int32 {
	return m.ServerId
}

func (m *ServerResponse) GetErrId() uint32 {
	return m.Err
}

func (m *ServerResponse) GetIncrId() uint64 {
	return m.IncrId
}

// 获取数据
func (m *ServerResponse) GetData() []byte {
	return m.Data
}

func (m *ServerResponse) SetMsgId(id uint32) {
	m.OpId = id
}

func (m *ServerResponse) SetDataLen(datalen uint32) {
	m.DataLen = datalen
}

func (m *ServerResponse) SetIncrId(incrId uint64) {
	m.IncrId = incrId
}

func (m *ServerResponse) SetClerkId(clerkId int32) {
	m.ClerkId = clerkId
}

func (m *ServerResponse) SetServerId(serverId int32) {
	m.ServerId = serverId
}

func (m *ServerResponse) SetErrId(err uint32) {
	m.Err = err
}

func (m *ServerResponse) SetData(data []byte) {
	m.Data = data
}

const respHeadLen uint32 = 28

type SerRespPack struct {
}

func (dp *SerRespPack) GetHeadLen() uint32 {
	return respHeadLen
}

// 封包
func (dp *SerRespPack) Pack(msg siface.IMessage) ([]byte, error) {
	respMsg, ok := msg.(*ServerResponse)
	if !ok {
		mylog.DPrintln("msg type not support")
		return nil, errors.New("msg type not support, only support [ServerResponse]")
	}
	// 1. 创建一个存放bytes字节的缓冲
	dataBuff := bytes.NewBuffer([]byte{})

	// 将MsgId 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, respMsg.GetMsgId()); err != nil {
		return nil, err
	}
	// 将 dataLen 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, respMsg.GetDataLen()); err != nil {
		return nil, err
	}

	// 将 incrId 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, respMsg.GetIncrId()); err != nil {
		return nil, err
	}

	// 将clerkId吸入databuff
	if err := binary.Write(dataBuff, binary.LittleEndian, respMsg.GetClerkId()); err != nil {
		return nil, err
	}

	if err := binary.Write(dataBuff, binary.LittleEndian, respMsg.GetServerId()); err != nil {
		return nil, err
	}

	// 将errId吸入databuff
	if err := binary.Write(dataBuff, binary.LittleEndian, respMsg.GetErrId()); err != nil {
		return nil, err
	}

	// 将 data 写进 databuff 中
	if err := binary.Write(dataBuff, binary.LittleEndian, respMsg.GetData()); err != nil {
		return nil, err
	}
	return dataBuff.Bytes(), nil
}

// 拆包
func (dp *SerRespPack) Unpack(data []byte) (siface.IMessage, error) {
	// 创建一个从输入二进制数据的ioreader
	dataReader := bytes.NewReader(data)

	msg := &ServerResponse{}

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

	if err := binary.Read(dataReader, binary.LittleEndian, &msg.IncrId); err != nil {
		mylog.DPrintln("read incrId error: ", err)
		return nil, err
	}

	if err := binary.Read(dataReader, binary.LittleEndian, &msg.ClerkId); err != nil {
		mylog.DPrintln("read clerkId error: ", err)
		return nil, err
	}

	if err := binary.Read(dataReader, binary.LittleEndian, &msg.ServerId); err != nil {
		mylog.DPrintln("read serverId error: ", err)
		return nil, err
	}

	if err := binary.Read(dataReader, binary.LittleEndian, &msg.Err); err != nil {
		mylog.DPrintln("read ErrId error: ", err)
		return nil, err
	}

	// 判断datalen是否超过所允许的最大包长度
	if utils.GlobalObject.MaxPackageSize > 0 && msg.DataLen > utils.GlobalObject.MaxPackageSize {
		return nil, errors.New("too Large msg data recv")
	}

	return msg, nil
}

func NewSerRespPack() *SerRespPack {
	return &SerRespPack{}
}
