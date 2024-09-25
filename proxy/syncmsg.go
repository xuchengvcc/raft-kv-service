package proxy

import (
	"bytes"
	"encoding/binary"
	"raft-kv-service/mylog"
)

const headLen int32 = 15

type SyncMsg struct {
	CommitIndex int64
	IsLeader    bool
	IpLen       int16
	ServerId    int32
	Port        int32
	// SyncPort    int32
	Ip string
}

type SyncMsgPack struct {
}

func (smp *SyncMsgPack) GetHeadLen() int32 {
	return headLen
}

func (smp *SyncMsgPack) PackAddr(msg SyncMsg) ([]byte, error) {
	dataBuff := bytes.NewBuffer([]byte{})
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.CommitIndex); err != nil {
		return nil, err
	}
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.IsLeader); err != nil {
		return nil, err
	}
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.IpLen); err != nil {
		return nil, err
	}
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.ServerId); err != nil {
		return nil, err
	}
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.Port); err != nil {
		return nil, err
	}
	// if err := binary.Write(dataBuff, binary.LittleEndian, msg.SyncPort); err != nil {
	// 	return nil, err
	// }
	bs := dataBuff.Bytes()
	bs = append(bs, []byte(msg.Ip)...)
	return bs, nil
}

func (smp *SyncMsgPack) Pack(msg SyncMsg) ([]byte, error) {
	dataBuff := bytes.NewBuffer([]byte{})
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.CommitIndex); err != nil {
		return nil, err
	}
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.IsLeader); err != nil {
		return nil, err
	}
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.IpLen); err != nil {
		return nil, err
	}
	if err := binary.Write(dataBuff, binary.LittleEndian, msg.ServerId); err != nil {
		return nil, err
	}
	return dataBuff.Bytes(), nil
}

func (smp *SyncMsgPack) UnpackHead(data []byte) (*SyncMsg, error) {
	dataReader := bytes.NewReader(data)
	msg := &SyncMsg{}

	if err := binary.Read(dataReader, binary.LittleEndian, &msg.CommitIndex); err != nil {
		mylog.DPrintln("read commitIndex error: ", err)
		return nil, err
	}

	if err := binary.Read(dataReader, binary.LittleEndian, &msg.IsLeader); err != nil {
		mylog.DPrintln("read isLeader error: ", err)
		return nil, err
	}

	if err := binary.Read(dataReader, binary.LittleEndian, &msg.IpLen); err != nil {
		mylog.DPrintln("read IpLen error: ", err)
		return nil, err
	}

	if err := binary.Read(dataReader, binary.LittleEndian, &msg.ServerId); err != nil {
		mylog.DPrintln("read serverId error: ", err)
		return nil, err
	}
	return msg, nil

}

func (smp *SyncMsgPack) UnpackBody(data []byte, msg *SyncMsg) error {
	dataReader := bytes.NewReader(data[:4])
	if err := binary.Read(dataReader, binary.LittleEndian, &msg.Port); err != nil {
		mylog.DPrintln("read port error: ", err)
		return err
	}
	msg.Ip = string(data[4:])
	return nil

}

func NewSyncMsgPack() *SyncMsgPack {
	return &SyncMsgPack{}
}
