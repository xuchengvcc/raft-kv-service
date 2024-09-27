package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	cm "raft-kv-service/common"
	"raft-kv-service/mylog"
	"raft-kv-service/persister"
	"raft-kv-service/proxy"
	"raft-kv-service/raft"
	rrpc "raft-kv-service/rpc"

	"raft-kv-service/labgob"
)

const RaftStateNumThreshold = 90
const HandleTimeout = time.Duration(1000) * time.Millisecond

const (
	GET uint32 = iota
	PUT
	APPEND
)

func optype(op uint32) string {
	switch op {
	case GET:
		return "GET"
	case PUT:
		return "PUT"
	case APPEND:
		return "APPEND"
	}
	return "UNKNOWN"
}

type KVServer struct {
	mu     sync.Mutex
	connMu sync.Mutex
	// getmu   sync.Mutex
	me      int32
	rf      *raft.Raftserver
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()
	UnimplementedServiceServer

	// cp *pool.ChannelPool[result]
	// waitCh  map[int64]*chan result         // 接收Raft提交条目的通道
	history map[int32]proxy.ServerResponse // 存储此前的历史结果

	incrMapLogIdx map[uint64]int64 // 指示一个opration开始时的全局 log index

	maxraftstate int // snapshot if log grows this big
	db           map[string]string
	lastApplied  int64 // apply的最后一个下标
	persister    *persister.Persister
	// Your definitions here.
	// readIdxQue []*readIdxOp

	// prepare
	raftprepare   chan struct{}
	Serverprepare chan struct{}
	initialized   bool

	// ip, port
	ip       string
	sendPort int32
	syncPort int32

	// 处理队列和待发送队列
	msgProcess  *MsgProcess
	processQue  chan *rrpc.Op
	sendQueues  chan *proxy.ServerResponse
	readQue     chan *rrpc.Op
	clientConns map[int32]net.Conn
	// addrMap     map[string]string
}

func (kv *KVServer) handleOpQue() {
	mylog.DPrintf("%d Start to handle Op queue", kv.me)
	for !kv.killed() {
		op := <-kv.processQue
		if op == nil {
			continue
		}
		kv.mu.Lock()
		if history, exist := kv.history[op.ClerkId]; exist && history.IncrId == op.IncrId {
			kv.mu.Unlock()
			kv.sendQueues <- &history
			continue
		}
		kv.mu.Unlock()
		if op.Optype == GET {
			// TODO: 发送到读请求等待队列
			kv.readQue <- op
			continue
		}
		_, sTerm, isLeader := kv.rf.Start(op)
		if !isLeader {
			resp, err := kv.msgProcess.OpToResp(op, cm.ErrMap[cm.ErrLeaderChanged])
			if err != nil {
				mylog.DPrintln("convert op to response error: ", err)
				continue
			}
			kv.sendQueues <- resp
			continue
		}
		kv.mu.Lock()
		// 记录该 opration 开始时的term，用于校验执行结束term是否发生变化
		kv.incrMapLogIdx[op.IncrId] = sTerm
		kv.mu.Unlock()
	}
}

// 定期扫描请求等待队列，将达到apply index的读请求执行并放到待发送队列
// follower比leader多一步：向leader请求当前的commit index
func (kv *KVServer) handleReadQue() {
	var readOp *rrpc.Op = nil
	count := 0
	for !kv.killed() {
		if readOp == nil {
			readOp = <-kv.readQue
			if readOp == nil {
				continue
			}
		}
		kv.mu.Lock()
		startIdx, exist := kv.incrMapLogIdx[readOp.IncrId]
		kv.mu.Unlock()
		if !exist {
			if startCommitIdx, isleader := kv.rf.GetState(); isleader {
				confirmLeader := kv.rf.SingleHeartBeat()
				if !confirmLeader {
					// 如果不是leader了，返回 ErrLeaderChanged 错误
					resp, err := kv.msgProcess.OpToResp(readOp, cm.ErrMap[cm.ErrLeaderChanged])
					if err != nil {
						mylog.DPrintln(err)
					}
					kv.sendQueues <- resp
					readOp = nil
					continue
				} else {
					// 记录当前的 commit_index
					kv.mu.Lock()
					kv.incrMapLogIdx[readOp.IncrId] = startCommitIdx
					kv.mu.Unlock()
				}
			} else {
				// 需要向leader请求当前的commit_index
				ch := make(chan int64, 1)
				err := kv.rf.SendCommitIndex(ch)
				idx := <-ch
				if err != nil && err == cm.ErrorWrongLeader {
					// Leader 不正确的处理, 将Leader切换错误告知集群管理
					resp, err := kv.msgProcess.OpToResp(readOp, cm.ErrMap[cm.ErrLeaderChanged])
					if err != nil {
						mylog.DPrintln(err)
					}
					kv.sendQueues <- resp
					readOp = nil
					continue
				} else if err != nil && count < 3 {
					count++
					mylog.DPrintln("try get commitIndex from leader error: ", err)
					time.Sleep(50 * time.Millisecond)
					continue
				} else if err != nil {
					count = 0
					resp, err := kv.msgProcess.OpToResp(readOp, cm.ErrMap[cm.ErrHandleTimeout])
					if err != nil {
						mylog.DPrintln(err)
					}
					kv.sendQueues <- resp
					readOp = nil
					continue
				}
				kv.mu.Lock()
				kv.incrMapLogIdx[readOp.IncrId] = idx
				kv.mu.Unlock()
			}
		}
		count = 0
		kv.mu.Lock()
		if !exist {
			startIdx = kv.incrMapLogIdx[readOp.IncrId]
		}
		if startIdx <= kv.lastApplied {
			resp := kv.DBExecute(readOp)
			kv.sendQueues <- &resp
			delete(kv.incrMapLogIdx, readOp.IncrId)
			readOp = nil
		} else {
			// 等待 50 ms下次检查应用的索引是否达到开始时提交的
			time.Sleep(50 * time.Millisecond)
		}
		kv.mu.Unlock()
	}
}

func (kv *KVServer) ApplyHandler() {
	mylog.DPrintf("%v Start ApplyHandler", kv.me)
	for !kv.killed() {
		// 循环将从节点提交的条目应用到状态机
		_log := <-kv.applyCh
		mylog.DPrintf("%v get log: %v", kv.me, _log.Command)
		if _log.CommandValid {
			op := _log.Command

			kv.mu.Lock()
			if _log.CommandIndex <= kv.lastApplied {
				kv.mu.Unlock()
				continue
			}
			kv.lastApplied = _log.CommandIndex

			// TODO： CommandIndex 超过了lastApplied，处理ReadIndex队列
			// go kv.processReadIndex(_log.CommandIndex)

			// 首先判断此 log 是否需要被应用
			var resp proxy.ServerResponse
			need := false
			if kv.history == nil {
				log.Fatal("kv.history is nil")
			}
			if op == nil {
				mylog.DPrintf("log: %v", _log)
				log.Fatal("op is nil")
			}
			if clerkMap, exist := kv.history[op.ClerkId]; exist {
				if clerkMap.IncrId == op.IncrId {
					// 用历史记录
					resp = clerkMap
				} else if clerkMap.IncrId < op.IncrId {
					need = true
				}
			} else {
				need = true
			}
			// _, isLeader := kv.rf.GetState()
			if need {
				// mylog.DPrintf("%v DBExecute, IsLeader: %v", kv.me, isLeader)
				resp = kv.DBExecute(op)
				if startTerm, exist := kv.incrMapLogIdx[resp.IncrId]; exist {
					if startTerm != _log.SnapshotTerm {
						resp.Err = cm.ErrMap[cm.ErrLeaderChanged]
					} else {
						mylog.DPrintf("%v Execute result: %v Update to history", kv.me, resp)
						kv.history[op.ClerkId] = resp // 更新历史
					}
					delete(kv.incrMapLogIdx, resp.IncrId)
				}
			}
			if startTerm, exist := kv.incrMapLogIdx[resp.IncrId]; exist {
				if startTerm != _log.SnapshotTerm && resp.Err != cm.ErrMap[cm.ErrLeaderChanged] {
					resp.Err = cm.ErrMap[cm.ErrLeaderChanged]
				}
				delete(kv.incrMapLogIdx, resp.IncrId)
			}
			kv.mu.Unlock()

			kv.sendQueues <- &resp

			if kv.maxraftstate != -1 && kv.persister.RaftStateSize() >= kv.maxraftstate*RaftStateNumThreshold/100 {
				// mylog.DPrintf("RaftStateSize: %v", kv.persister.RaftStateSize())
				snapshot := kv.Snapshot()
				kv.rf.Snapshot(_log.CommandIndex, snapshot)
			}

			// 发送消息

		} else if _log.SnapshotValid {
			kv.mu.Lock()
			if _log.SnapshotIndex >= kv.lastApplied {
				kv.LoadSnapshot(_log.Snapshot)
				kv.lastApplied = _log.SnapshotIndex
			}
			kv.mu.Unlock()
		}
	}
	// defer kv.cp.Close()
}

func (kv *KVServer) Snapshot() []byte {
	buffer := new(bytes.Buffer)
	encoder := labgob.NewEncoder(buffer)

	encoder.Encode(kv.db)
	encoder.Encode(kv.history)

	return buffer.Bytes()
}

func (kv *KVServer) LoadSnapshot(snapshot []byte) {
	if snapshot == nil || len(snapshot) < 1 {
		mylog.DPrintf("%v Snapshot is Empty", kv.me)
		return
	}

	buffer := bytes.NewBuffer(snapshot)
	decoder := labgob.NewDecoder(buffer)

	db := make(map[string]string)
	historyDb := make(map[int32]proxy.ServerResponse)
	if decoder.Decode(&db) != nil || decoder.Decode(&historyDb) != nil {
		// mylog.DPrintf("%v Decode Snapshot Failed", kv.me)
	} else {
		kv.db = db
		kv.history = historyDb
		// mylog.DPrintf("%v Decode Snapshot Successfully", kv.me)
	}
}

func (kv *KVServer) DBExecute(op *rrpc.Op) proxy.ServerResponse {
	// 需在加锁状态下调用
	mylog.DPrintf("%v Execute DBExecute, IncrId: %v, %v(%s: %s)", kv.me, op.IncrId, optype(op.Optype), op.Key, op.Value)
	// resp, err := kv.msgProcess.OpToResp()
	var errId uint32 = 200
	switch op.Optype {
	case GET:
		val, exist := kv.db[op.Key]
		if exist {
			op.Value = val
			// TODO: 记录请求成功
		} else {
			errId = cm.ErrMap[cm.ErrNoKey]
			op.Value = ""
			// TODO: 记录请求失败
		}
	case PUT:
		kv.db[op.Key] = op.Value
		// TODO：记录
	case APPEND:
		val, exist := kv.db[op.Key]
		if exist {
			kv.db[op.Key] = val + op.Value
		} else {
			kv.db[op.Key] = op.Value
		}
	}
	resp, err := kv.msgProcess.OpToResp(op, errId)
	if err != nil {
		mylog.DPrintln("convert op to response error: ", err)
	}
	return *resp
}

func (kv *KVServer) sendStatus(conn *net.TCPConn) error {
	commitIndex, isLeader := kv.rf.GetState()
	message := &proxy.SyncMsg{
		ServerId:    kv.me,
		CommitIndex: commitIndex,
		IsLeader:    isLeader,
		IpLen:       0,
	}
	var bitData []byte
	var err error
	if !kv.initialized {
		message.IpLen = int16(len(kv.ip))
		message.Port = kv.sendPort
		// message.SyncPort = kv.syncPort
		message.Ip = kv.ip
		bitData, err = kv.msgProcess.SyncPackAddr(*message)
	} else {
		bitData, err = kv.msgProcess.SyncPack(*message)
	}
	if err != nil {
		mylog.DPrintln("pack syncMsg to binary data error: ", err)
		return err
	}
	if _, err = conn.Write(bitData); err != nil {
		mylog.DPrintln("send sync state msg error: ", err)
		return err
	}

	// receive nginx confirm message (only for initialization)
	if !kv.initialized {
		buffer := make([]byte, 4)
		// timeout

		timeout := time.Now().Add(5 * time.Second)
		err := conn.SetReadDeadline(timeout)
		if err != nil {
			mylog.DPrintln("failed to set deadline, error: ", err)
			return err
		}

		_, err = io.ReadFull(conn, buffer)

		if err != nil {
			mylog.DPrintln("failed to read from sync conn, error: ", err)
			return err
		}

		dataReader := bytes.NewReader(buffer)
		var code uint32
		err = binary.Read(dataReader, binary.LittleEndian, &code)
		if err != nil {
			mylog.DPrintln("read state confirm error: ", err)
			return err
		}
		mylog.DPrintf("%d Received response from sync conn, %s\n", kv.me, cm.ErrIdMap[code])
		kv.initialized = true
	}
	return nil
}

// 定时向反向代理发送节点状态信息
func (kv *KVServer) sendStatusSchedule(proxyAddr string) {
	pAddr, err := net.ResolveTCPAddr("tcp", proxyAddr)
	if err != nil {
		mylog.DPrintln("resolving TCP address error: ", err)
		panic(fmt.Sprintf("resolving TCP address error: %s", err))
	}
	serverAddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", kv.ip, kv.syncPort))
	if err != nil {
		mylog.DPrintln("resolving sever address error: ", err)
		panic(fmt.Sprintf("resolving server address error: %s", err))
	}

	conn, err := net.DialTCP("tcp", serverAddr, pAddr)
	if err != nil {
		mylog.DPrintln("dialing tcp error: ", err)
		panic(fmt.Sprintf("dialing tcp error: %s", err))
	}
	defer conn.Close()

	mylog.DPrintln("Raft node started, sending status to nginx...")

	for !kv.killed() {
		err = kv.sendStatus(conn)
		if err != nil && (err == io.EOF || err == io.ErrUnexpectedEOF || strings.Contains(err.Error(), "use of closed network connection")) {
			conn.Close()
			conn, err = net.DialTCP("tcp", serverAddr, pAddr)
			if err != nil {
				mylog.DPrintln("dialing tcp error: ", err)
				panic(err)
			}
			kv.initialized = false
		}
		time.Sleep(1 * time.Second)
	}
}

// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

func (kv *KVServer) handleClientResponse() {
	for {
		resp := <-kv.sendQueues
		if resp == nil {
			continue
		}
		bitData, err := kv.msgProcess.RespToBinary(resp)
		if err != nil {
			mylog.DPrintln("pack op to binary data error: ", err)
			// TODO: 告知客户端错误信息
			continue
		}
		kv.connMu.Lock()
		conn, exist := kv.clientConns[resp.ClerkId]
		kv.connMu.Unlock()
		if exist {
			if _, err := conn.Write(bitData); err != nil {
				mylog.DPrintln("send result data error: ", err)
				// TODO: 根据错误类型进行决策，不能直接return
				return
			}
		}
	}

}

// 处理来自代理的消息，解包并将数据组装成server处理的数据结构 Op，放到server的处理队列中
func (kv *KVServer) handleClientConnection(conn net.Conn) {
	defer conn.Close()

	for !kv.killed() {
		headBuffer := make([]byte, kv.msgProcess.clientDataPack.GetHeadLen())
		_, err := io.ReadFull(conn, headBuffer)
		if err != nil {
			mylog.DPrintln("failed to read from client error: ", err)
			// TODO:根据错误类型进行判断如何处理该连接
			return
		}
		// unpack
		msg, err := kv.msgProcess.clientDataPack.Unpack(headBuffer)
		if err != nil {
			mylog.DPrintln("unpack msghead error: ", err)
			// TODO:根据错误类型告知客户端出现错误
			break
		}
		buffer := make([]byte, msg.GetDataLen())
		_, err = io.ReadFull(conn, buffer)
		if err != nil {
			mylog.DPrintln("failed to read body from stream error: ", err)
			// TODO:根据错误类型告知客户端出现错误
			break
		}
		msg.SetData(buffer)
		op, err := kv.msgProcess.ClientMsgToOp(msg)
		if err != nil {
			mylog.DPrintln("convert to Op error: ", err)
			// TODO:根据错误类型告知客户端出现错误
			break
		}

		kv.connMu.Lock()
		if _, exist := kv.clientConns[op.ClerkId]; !exist {
			kv.clientConns[op.ClerkId] = conn
		}
		kv.connMu.Unlock()

		kv.processQue <- op
	}

}

func (kv *KVServer) serve() {
	ip_port := fmt.Sprintf("%s:%d", kv.ip, kv.sendPort)
	lis, err := net.Listen("tcp", ip_port)
	if err != nil {
		mylog.DPrintf("server %d listenning at %s error: %v", kv.me, ip_port, err)
		return
	}
	defer lis.Close()

	mylog.DPrintf("server %d listenning at %s success", kv.me, ip_port)
	for {
		conn, err := lis.Accept()
		if err != nil {
			mylog.DPrintln("failed to accept connection: ", err)
			continue
		}
		mylog.DPrintf("Server %d get new connection: %v", kv.me, conn)
		go kv.handleClientConnection(conn)
	}

}

func (kv *KVServer) isPrepare() {
	defer func() {
		close(kv.raftprepare)
		close(kv.Serverprepare)
	}()
	struc := <-kv.raftprepare
	kv.Serverprepare <- struc
}

// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
func StartKVServer(peerAddrs map[int32]string, me int32, persister *persister.Persister, addr map[string]string, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(rrpc.Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.lastApplied = 0
	kv.persister = persister

	// You may need initialization code here.
	ip_port := strings.Split(addr["server"], ":")
	kv.ip = ip_port[0]
	port, err := strconv.Atoi(ip_port[1])
	if err != nil {
		mylog.DPrintln("port convrt error: ", err)
		panic(err)
	}
	kv.sendPort = int32(port)

	sync_ip_port := strings.Split(addr["sync"], ":")
	sync_port, err := strconv.Atoi(sync_ip_port[1])
	if err != nil {
		mylog.DPrintln("sync_port convert error: ", err)
		panic(err)
	}
	kv.syncPort = int32(sync_port)
	kv.initialized = false

	// kv.cp = pool.NewChannelPool[result](5)
	kv.applyCh = make(chan raft.ApplyMsg, 1)
	kv.raftprepare = make(chan struct{}, 1)
	kv.Serverprepare = make(chan struct{}, 1)
	kv.rf = raft.Make(peerAddrs, me, persister, kv.applyCh, addr["raft"], kv.raftprepare)

	// You may need initialization code here.
	kv.dead = 0
	kv.incrMapLogIdx = make(map[uint64]int64)
	kv.history = make(map[int32]proxy.ServerResponse)
	// kv.waitCh = make(map[int64]*chan result)
	kv.db = make(map[string]string)

	kv.msgProcess = NewMsgProcess()
	kv.readQue = make(chan *rrpc.Op, 100)                 // 容量为 100 的待读取请求消息
	kv.processQue = make(chan *rrpc.Op, 100)              // 容量为 100 的待处理消息
	kv.sendQueues = make(chan *proxy.ServerResponse, 100) // 容量为 100 的发送队列
	kv.clientConns = make(map[int32]net.Conn)
	// kv.addrMap = addr

	kv.mu.Lock()
	snap, lastIndex, _ := persister.ReadSnapshot()
	kv.lastApplied = lastIndex
	kv.LoadSnapshot(snap)
	kv.mu.Unlock()
	go kv.serve()
	// go persister.SaveSchedule()
	go kv.ApplyHandler()
	sync := true
	if sync {
		go kv.sendStatusSchedule(addr["cluster_sync"]) // 定时同步状态信息
	}
	go kv.handleClientResponse() // 从结果队列取出结果并发送
	go kv.handleReadQue()        // 处理读请求任务
	go kv.handleOpQue()          // 从请求队列取出任务并执行
	go kv.isPrepare()            // 通知服务器已准备就绪

	return kv
}

func (kv *KVServer) Stop() {
	kv.rf.Kill()
	kv.Kill()
	kv.connMu.Lock()
	for _, conn := range kv.clientConns {
		conn.Close()
	}
	kv.connMu.Unlock()
	// for _, ch := range kv.waitCh {
	// 	close(*ch)
	// }
	close(kv.sendQueues)
	close(kv.processQue)
	close(kv.readQue)
}
