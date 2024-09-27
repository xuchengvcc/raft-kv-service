package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	cm "raft-kv-service/common"
	"raft-kv-service/mylog"
	"raft-kv-service/proxy"
	"sync"
	"sync/atomic"
	"time"
)

var id int32 = 100
var mu sync.Mutex

const RPCRetryTime = time.Millisecond * 5
const WrongLeaderRetryTime = time.Millisecond * 3

type Clerk struct {
	// servers []ServiceClient
	// You will have to modify this struct.
	// leader int
	// incrId       uint64
	clerkId      int32
	connections  map[uint32]net.Conn
	maxRetryTime int // 最大请求重试次数
	// 封包、拆包
	packer     proxy.ClientDataPack
	bodypacker proxy.BodyPack
	respPacker proxy.SerRespPack

	muP               sync.Mutex
	muG               sync.Mutex
	muM               sync.Mutex
	putOffset         int
	getOffset         int
	responsePutBuffer []byte
	responseGetBuffer []byte
	responseCh        map[uint64]chan *proxy.Result

	// close
	dead int32
}

func (ck *Clerk) Kill() {
	atomic.StoreInt32(&ck.dead, 1)
	// Your code here, if desired.
}

func (ck *Clerk) killed() bool {
	z := atomic.LoadInt32(&ck.dead)
	return z == 1
}

func (c *Clerk) getUid() uint64 {
	t := time.Now()
	timestamp := uint64(t.UTC().UnixNano())
	// timestamp := strconv.FormatInt(nrand(), 10)
	return timestamp
}

func (c *Clerk) getClerkId() int32 {
	mu.Lock()
	defer mu.Unlock()
	id++
	return id
}

// func (c *Clerk) connectToServer(addrs []string) error {
// 	failed := make([]int, 0)
// 	for i, addr := range addrs {
// 		mylog.DPrintf("Client Try Connect to Server %v: %s", i, addr)
// 		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
// 		if err != nil {
// 			mylog.DPrintf("Client did not connect to %s: %v", addr, err)
// 			failed = append(failed, i)
// 			continue
// 		}
// 		if conn == nil {
// 			return fmt.Errorf("no server listening at %s", addr)
// 		}
// 		c.servers[i] = NewServiceClient(conn)
// 	}

// 	for len(failed) > 0 {
// 		i := failed[len(failed)-1]
// 		conn, err := grpc.NewClient(addrs[i], grpc.WithTransportCredentials(insecure.NewCredentials()))
// 		if err != nil {
// 			mylog.DPrintf("Client Retry connect to %s: %v", addrs[i], err)
// 			time.Sleep(500 * time.Millisecond)
// 			continue
// 		}
// 		c.servers[i] = NewServiceClient(conn)
// 		failed = failed[:len(failed)-1]
// 	}
// 	return nil
// }

func (c *Clerk) connectToProxy(addrMap map[uint32]string) error {
	failed := make(map[uint32]string)
	for op, addr := range addrMap {
		mylog.DPrintf("Client %d try connect to proxy %s", c.clerkId, addr)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			failed[op] = addr
			mylog.DPrintf("%d connect to proxy: %s failed", c.clerkId, addr)
			continue
		}
		c.connections[op] = conn
	}
	count := 0
	for len(failed) > 0 && count < 10 {
		for op, addr := range failed {
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				count++
				time.Sleep(500 * time.Millisecond)
				continue
			}
			c.connections[op] = conn
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d connect to proxy failed", c.clerkId)
	}
	if _, exist := c.connections[APPEND]; !exist {
		c.connections[APPEND] = c.connections[PUT]
	}
	return nil
}

func MakeClerk(addrs map[uint32]string) *Clerk {
	ck := new(Clerk)
	// ck.servers = make([]ServiceClient, len(addrs))
	// err := ck.connectToServer(addrs)
	// You'll have to add code here.
	// ck.leader = 0
	// ck.incrId = ck.getUid()
	ck.clerkId = ck.getClerkId()
	ck.maxRetryTime = 1
	ck.connections = make(map[uint32]net.Conn)
	ck.packer = *proxy.NewClientDataPack()
	ck.bodypacker = *proxy.NewBodyPack()
	ck.respPacker = *proxy.NewSerRespPack()
	ck.responseGetBuffer = make([]byte, 2048)
	ck.responsePutBuffer = make([]byte, 2048)
	ck.responseCh = make(map[uint64]chan *proxy.Result)
	err := ck.connectToProxy(addrs)
	if err != nil {
		panic(err)
	}
	go ck.receiveGetResponse()
	go ck.receivePutAppendResponse()
	return ck
}

func (ck *Clerk) Stop() {
	ck.Kill()
	for _, conn := range ck.connections {
		conn.Close()
	}
	for _, ch := range ck.responseCh {
		close(ch)
	}
	mylog.DPrintf("clerk %d stoped\n", ck.clerkId)
}

func (ck *Clerk) prepareData(keyValue []byte, op, keyLen uint32) ([]byte, uint64, error) {
	body := &proxy.Body{
		KeyValue: keyValue,
	}

	body.SetKeyLen(keyLen)
	bodyData, err := ck.bodypacker.Pack(*body)
	if err != nil {
		return nil, 0, fmt.Errorf("body pack error: %s", err)
	}

	msg := &proxy.ClientMessage{
		DataLen: uint32(len(bodyData)),
		OpId:    op,
		IncrId:  ck.getUid(),
		ClerkId: ck.clerkId,
		Data:    bodyData,
	}
	ck.muM.Lock()
	if _, exist := ck.responseCh[msg.GetIncrId()]; !exist {
		ck.responseCh[msg.GetIncrId()] = make(chan *proxy.Result, 1)
	}
	ck.muM.Unlock()
	binaryData, err := ck.packer.Pack(msg)
	if err != nil {
		return nil, 0, fmt.Errorf("request pack error: %s", err)
	}
	return binaryData, msg.IncrId, nil
}

func (ck *Clerk) packData(keyValue []byte, op, keyLen uint32) ([]byte, uint64, error) {
	if op != GET && op != PUT && op != APPEND {
		return nil, 0, fmt.Errorf("optype not support, only support get/put/append")
	}
	return ck.prepareData(keyValue, op, keyLen)
}

// 将消息体内容解析，并通过channel传递
func (ck *Clerk) processResponse(incrId uint64, errId uint32, binaryData []byte) error {
	body, err := ck.bodypacker.Unpack(binaryData)
	// mylog.DPrintln("[Client] get result Error: ", cm.ErrIdMap[errId], ", key: ", body.GetKey(), ", value: ", body.GetValue())
	if err != nil {
		return err
	}
	res := &proxy.Result{
		ErrId: errId,
		Bbody: *body,
	}
	ck.muM.Lock()
	if ch, ok := ck.responseCh[incrId]; ok {
		ch <- res
		close(ch)
		delete(ck.responseCh, incrId)
	}
	ck.muM.Unlock()
	return nil
}

func (ck *Clerk) processGetBuffer() {
	for {
		if ck.getOffset < int(ck.respPacker.GetHeadLen()) {
			return
		}
		dataLen := binary.LittleEndian.Uint32(ck.responseGetBuffer[4:8])

		// 消息不完整，等待完整数据
		if uint32(ck.getOffset) < dataLen+ck.respPacker.GetHeadLen() {
			return
		}

		msg, err := ck.respPacker.Unpack(ck.responseGetBuffer[:ck.respPacker.GetHeadLen()])
		smsg := msg.(*proxy.ServerResponse)
		if err != nil {
			mylog.DPrintln("unpack reponse error: ", err)
			return
		}
		data := make([]byte, smsg.GetDataLen())
		copy(data, ck.responseGetBuffer[ck.respPacker.GetHeadLen():ck.respPacker.GetHeadLen()+dataLen])
		smsg.SetData(data)
		err = ck.processResponse(smsg.IncrId, smsg.Err, smsg.GetData())
		if err != nil {
			mylog.DPrintln("processBody error: ", err)
		}

		// 移除已处理部分，保留未处理部分
		copy(ck.responseGetBuffer, ck.responseGetBuffer[ck.respPacker.GetHeadLen()+dataLen:ck.getOffset])
		ck.getOffset -= int(dataLen + ck.respPacker.GetHeadLen())
	}
}

func (ck *Clerk) processPutBuffer() {
	for {
		if ck.putOffset < int(ck.respPacker.GetHeadLen()) {
			return
		}

		// opId := binary.LittleEndian.Uint32(ck.responsePutBuffer[0:4])
		dataLen := binary.LittleEndian.Uint32(ck.responsePutBuffer[4:8])

		// 消息不完整，等待完整数据
		if uint32(ck.putOffset) < dataLen+ck.respPacker.GetHeadLen() {
			return
		}

		msg, err := ck.respPacker.Unpack(ck.responsePutBuffer[:ck.respPacker.GetHeadLen()])
		smsg := msg.(*proxy.ServerResponse)
		if err != nil {
			mylog.DPrintln("unpack reponse error: ", err)
			return
		}
		smsg.SetData(ck.responsePutBuffer[ck.respPacker.GetHeadLen() : ck.respPacker.GetHeadLen()+dataLen])
		err = ck.processResponse(smsg.IncrId, smsg.Err, smsg.GetData())
		if err != nil {
			mylog.DPrintln("processBody error: ", err)
		}

		// 移除已处理部分，保留未处理部分
		copy(ck.responsePutBuffer, ck.responsePutBuffer[ck.respPacker.GetHeadLen()+dataLen:ck.putOffset])
		ck.putOffset -= int(dataLen + ck.respPacker.GetHeadLen())
	}
}

func (ck *Clerk) receivePutAppendResponse() {
	for !ck.killed() {
		ck.muP.Lock()
		n, err := ck.connections[PUT].Read(ck.responsePutBuffer[ck.putOffset:])
		ck.muP.Unlock()

		if err != nil {
			mylog.DPrintln("reading from server error: ", err)
			// TODO: 如果是连接断开，每隔一段时间尝试重新连接
			return
		}

		ck.putOffset += n
		ck.processPutBuffer()
	}
}

func (ck *Clerk) receiveGetResponse() {
	for !ck.killed() {
		ck.muG.Lock()
		n, err := ck.connections[GET].Read(ck.responseGetBuffer[ck.getOffset:])
		ck.muG.Unlock()

		if err != nil {
			mylog.DPrintln("reading from server error: ", err)
			// TODO: 如果是连接断开，每隔一段时间尝试重新连接
			return
		}

		ck.getOffset += n
		// mylog.DPrintln("[Client] getBuffer: ", ck.responseGetBuffer[:ck.getOffset])
		ck.processGetBuffer()
	}
}

func (ck *Clerk) sendRequest(binaryMsg []byte, op uint32) error {
	if _, exist := ck.connections[op]; !exist {
		return errors.New("connection dont exists")
	}
	_, err := ck.connections[op].Write(binaryMsg)
	if err != nil {
		return err
	}
	return nil
}

// 重试 ck.maxRetryTime 次
func (ck *Clerk) retryRequest(op, keyLen uint32, keyValue []byte, timeout time.Duration) (*proxy.Result, error) {
	var lastErr error
	binaryData, incrId, err := ck.packData(keyValue, op, keyLen)
	if err != nil {
		mylog.DPrintln("pack data error: ", err)
		// 数据装包错误，该通道不会再用，关闭通道
		ck.muM.Lock()
		if ch, exist := ck.responseCh[incrId]; exist {
			close(ch)
			delete(ck.responseCh, incrId)
		}
		ck.muM.Unlock()
		return nil, err
	}
	ch := ck.responseCh[incrId]
	for attempt := 0; attempt < ck.maxRetryTime; attempt++ {
		err := ck.sendRequest(binaryData, op)
		if err != nil {
			lastErr = err
			continue
		}

		select {
		case resp := <-ch:
			return resp, nil
		case <-time.After(timeout):
			mylog.DPrintf("Attemp %d: Request timed out after %v \n", attempt, timeout)
			lastErr = errors.New("request time out")
		}
	}

	return nil, lastErr
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer."+op, &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string, timeout time.Duration) (string, error) {
	res, err := ck.retryRequest(GET, uint32(len(key)), []byte(key), timeout)
	if err != nil {
		return "", err
	}
	if res.ErrId != cm.ErrMap[cm.OK] {
		return "", fmt.Errorf(cm.ErrIdMap[res.ErrId])
	}
	return res.GetValue(), nil
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Put(key string, value string, timeout time.Duration) error {
	// You will have to modify this function.
	keyValue := make([]byte, 0, len(key)+len(value))
	keyValue = append(keyValue, []byte(key)...)
	keyValue = append(keyValue, []byte(value)...)
	res, err := ck.retryRequest(PUT, uint32(len(key)), keyValue, timeout)
	if err != nil {
		return err
	}
	if res.ErrId != cm.ErrMap[cm.OK] {
		return fmt.Errorf(cm.ErrIdMap[res.ErrId], err)
	}
	return nil
}

func (ck *Clerk) Append(key string, value string, timeout time.Duration) error {
	// You will have to modify this function.
	keyValue := make([]byte, 0, len(key)+len(value))
	keyValue = append(keyValue, []byte(key)...)
	keyValue = append(keyValue, []byte(value)...)
	res, err := ck.retryRequest(APPEND, uint32(len(key)), keyValue, timeout)
	if res.ErrId != cm.ErrMap[cm.OK] {
		return fmt.Errorf(cm.ErrIdMap[res.ErrId], err)
	}
	return err
}

// func (ck *Clerk) Put(key string, value string) error {
// 	return ck.PutAppend(key, value, ck.servers[ck.leader].Put)
// }
// func (ck *Clerk) Append(key string, value string) error {
// 	return ck.PutAppend(key, value, ck.servers[ck.leader].Append)
// }
