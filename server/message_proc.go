package server

import (
	"fmt"
	"raft-kv-service/mylog"
	"raft-kv-service/proxy"
	"raft-kv-service/proxy/siface"
	rrpc "raft-kv-service/rpc"
)

type MsgProcess struct {
	clientDataPack proxy.ClientDataPack
	clientBodyPack proxy.BodyPack
	serRespPack    proxy.SerRespPack
	syncMsgPack    proxy.SyncMsgPack
}

func NewMsgProcess() *MsgProcess {
	msgPross := &MsgProcess{
		clientDataPack: *proxy.NewClientDataPack(),
		clientBodyPack: *proxy.NewBodyPack(),
		serRespPack:    *proxy.NewSerRespPack(),
		syncMsgPack:    *proxy.NewSyncMsgPack(),
	}
	return msgPross
}

func (mp *MsgProcess) ClientMsgToBit(msg siface.IMessage) ([]byte, error) {
	return mp.clientDataPack.Pack(msg)
}

func (mp *MsgProcess) ClientMsgToOp(msg siface.IMessage) (*rrpc.Op, error) {
	cmsg, ok := msg.(*proxy.ClientMessage)
	if !ok {
		mylog.DPrintln("convert to clientmessage")
		return nil, fmt.Errorf("cannot convert to clientmessage")
	}
	body, err := mp.clientBodyPack.Unpack(msg.GetData())
	if err != nil {
		mylog.DPrintln("unpack binary data to body error: ", err)
		return nil, err
	}

	op := &rrpc.Op{
		Optype:  cmsg.GetMsgId(),
		ClerkId: cmsg.GetClerkId(),
		IncrId:  cmsg.GetIncrId(),
		Key:     body.GetKey(),
		Value:   body.GetValue(),
	}
	return op, nil
}

func (mp *MsgProcess) OpToBinary(op *rrpc.Op) ([]byte, error) {
	keyLen := len(op.Key)
	keyValue := make([]byte, 0, keyLen+len(op.Value))
	keyValue = append(keyValue, []byte(op.Key)...)
	keyValue = append(keyValue, []byte(op.Value)...)
	body := &proxy.Body{
		KeyValue: keyValue,
	}
	body.SetKeyLen(uint32(keyLen))
	binaryData, err := mp.clientBodyPack.Pack(*body)
	if err != nil {
		mylog.DPrintln("pack body error: ", err)
		return nil, err
	}
	msg := &proxy.ClientMessage{
		DataLen: uint32(len(binaryData)),
		OpId:    op.Optype,
		IncrId:  op.IncrId,
		ClerkId: op.ClerkId,
		Data:    binaryData,
	}
	bData, err := mp.clientDataPack.Pack(msg)
	if err != nil {
		mylog.DPrintln("pack client message error: ", err)
		return nil, err
	}

	return bData, nil
}

func (mp *MsgProcess) OpToResp(op *rrpc.Op, errId uint32) (*proxy.ServerResponse, error) {
	keyLen := len(op.Key)
	keyValue := make([]byte, 0, keyLen+len(op.Value))
	keyValue = append(keyValue, []byte(op.Key)...)
	keyValue = append(keyValue, []byte(op.Value)...)

	body := &proxy.Body{
		KeyValue: keyValue,
	}
	body.SetKeyLen(uint32(keyLen))
	binaryData, err := mp.clientBodyPack.Pack(*body)
	if err != nil {
		mylog.DPrintln("pack body error: ", err)
		return nil, err
	}
	resp := &proxy.ServerResponse{
		OpId:    op.Optype,
		DataLen: uint32(len(binaryData)),
		IncrId:  op.IncrId,
		ClerkId: op.ClerkId,
		Err:     errId,
		Data:    binaryData,
	}
	return resp, err
}

func (mp *MsgProcess) RespToReq(resp *proxy.ServerResponse) *proxy.ClientMessage {
	req := &proxy.ClientMessage{
		OpId:    resp.OpId,
		DataLen: resp.DataLen,
		IncrId:  resp.IncrId,
		ClerkId: resp.ClerkId,
		Data:    resp.Data,
	}
	return req
}

func (mp *MsgProcess) RespToBinary(resp *proxy.ServerResponse) ([]byte, error) {
	bs, err := mp.serRespPack.Pack(resp)
	if err != nil {
		mylog.DPrintln("Pack serverResp to bianry error: ", err)
		return nil, err
	}
	return bs, nil
}

func (mp *MsgProcess) SyncPack(msg proxy.SyncMsg) ([]byte, error) {
	return mp.syncMsgPack.Pack(msg)
}

func (mp *MsgProcess) SyncPackAddr(msg proxy.SyncMsg) ([]byte, error) {
	return mp.syncMsgPack.PackAddr(msg)
}

func (mp *MsgProcess) SyncHeadUnpack(data []byte) (*proxy.SyncMsg, error) {
	return mp.syncMsgPack.UnpackHead(data)
}

func (mp *MsgProcess) SyncBodyUnpack(data []byte, msg *proxy.SyncMsg) error {
	return mp.syncMsgPack.UnpackBody(data, msg)
}
