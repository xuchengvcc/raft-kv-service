package server

import (
	"bytes"
	"context"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"raft-kv-service/raft"

	"raft-kv-service/labgob"

	grpc "google.golang.org/grpc"
)

const RaftStateNumThreshold = 90
const HandleTimeout = time.Duration(500) * time.Millisecond

const (
	GET int32 = iota
	PUT
	APPEND
)

func optype(op int32) string {
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

type result struct {
	LastId  uint64
	Err     string
	Value   string
	ResTerm int64 // commit时的Term
}

type readIdxOp struct {
	Op        *raft.Op
	ReadIdx   int64
	ReadIdxCh *chan result
}

type KVServer struct {
	mu      sync.Mutex
	getmu   sync.Mutex
	me      int32
	rf      *raft.Raftserver
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()
	UnimplementedServiceServer

	waitCh  map[int64]*chan result // 接收Raft提交条目的通道
	history map[int32]result       // 存储此前的历史结果

	maxraftstate int // snapshot if log grows this big
	db           map[string]string
	lastApplied  int64 // apply的最后一个下标
	persister    *raft.Persister
	// Your definitions here.
	readIdxQue []*readIdxOp
}

func (kv *KVServer) Get(ctx context.Context, args *GetArgs) (*GetReply, error) {
	// Your code here.
	raft.DPrintf("Server %v Get Request", kv.me)
	reply := &GetReply{}
	commitIndex, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return reply, nil
		// return reply, errWrongLeader
	}
	opArgs := &raft.Op{
		Optype:  GET,
		IncrId:  args.IncrId,
		ClerkId: args.ClerkId,
		Key:     args.Key,
	}
	res := kv.HandleOp(opArgs)
	raft.DPrintf("Server %v Handler Over %v", kv.me, res.ResTerm)
	if res.ResTerm == -1 {
		temp := make(chan result, 1)
		raft.DPrintf("Server %v Prepare Send SingleHeartBeat", kv.me)
		stillLeader := kv.rf.SingleHeartBeat()
		raft.DPrintf("Server %v SingleHeartBeat Over stillLeader?%v", kv.me, stillLeader)
		kv.getmu.Lock()
		kv.readIdxQue = append(kv.readIdxQue, &readIdxOp{Op: opArgs, ReadIdx: commitIndex, ReadIdxCh: &temp})
		kv.getmu.Unlock()
		if stillLeader {
			raft.DPrintf("Still Leader? %v", stillLeader)
			go kv.processReadIndex(kv.lastApplied)
			select {
			case res = <-temp:
				raft.DPrintf("Result %v", res)
			case <-time.After(HandleTimeout):
				res.Err = ErrHandleTimeout
			}
			kv.getmu.Lock()
			close(temp)
			kv.getmu.Unlock()
		} else {
			reply.Err = ErrWrongLeader
			kv.getmu.Lock()
			kv.readIdxQue = kv.readIdxQue[:0]
			kv.getmu.Unlock()
			return reply, nil
		}
	}
	reply.Err = res.Err
	reply.Value = res.Value
	raft.DPrintf("%v Get( %v) Result(Err: %v, Key: %v, Value: %v)", kv.me, args.IncrId, res.Err, args.Key, res.Value)
	return reply, nil
}

func (kv *KVServer) Put(ctx context.Context, args *PutAppendArgs) (*PutAppendReply, error) {
	// Your code here.
	raft.DPrintf("Server %v Put Request", kv.me)
	reply := &PutAppendReply{}
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return reply, nil
	}
	opArgs := &raft.Op{
		Optype:  PUT,
		IncrId:  args.IncrId,
		ClerkId: args.ClerkId,
		Key:     args.Key,
		Value:   args.Value,
	}
	res := kv.HandleOp(opArgs)
	reply.Err = res.Err
	raft.DPrintf("%v Put( %v) Result(Err: %v, Key: %v, Value: %v)", kv.me, args.IncrId, res.Err, args.Key, args.Value)
	return reply, nil
}

func (kv *KVServer) Append(ctx context.Context, args *PutAppendArgs) (*PutAppendReply, error) {
	// Your code here.
	raft.DPrintf("Server %v Append Request", kv.me)
	reply := &PutAppendReply{}
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		raft.DPrintf("KVServer %v is not Leader", kv.me)
		reply.Err = ErrWrongLeader
		return reply, nil
	}
	opArgs := &raft.Op{
		Optype:  APPEND,
		IncrId:  args.IncrId,
		ClerkId: args.ClerkId,
		Key:     args.Key,
		Value:   args.Value,
	}
	res := kv.HandleOp(opArgs)
	reply.Err = res.Err
	raft.DPrintf("%v Append( %v) Result(Err: %v, Key: %v, Value: %v)", kv.me, args.IncrId, res.Err, args.Key, args.Value)
	return reply, nil
}

func (kv *KVServer) HandleOp(opArgs *raft.Op) result {
	kv.mu.Lock()
	if history, exist := kv.history[opArgs.ClerkId]; exist && history.LastId == opArgs.IncrId {
		kv.mu.Unlock()
		return history
	}
	kv.mu.Unlock()
	if opArgs.Optype == GET {
		return result{ResTerm: -1}
	}
	sIdx, sTerm, isLeader := kv.rf.Start(opArgs)
	if !isLeader {
		return result{Err: ErrWrongLeader, Value: ""}
	}

	kv.mu.Lock()
	newCh := make(chan result, 1)
	kv.waitCh[sIdx] = &newCh
	// raft.DPrintf("L %v Clerk %v IncrId %v create new Channel %p", kv.me, opArgs.ClerkId, opArgs.IncrId, &newCh)
	kv.mu.Unlock()

	defer func() {
		kv.mu.Lock()
		delete(kv.waitCh, sIdx)
		close(newCh)
		kv.mu.Unlock()
	}()

	select {
	case <-time.After(HandleTimeout):
		raft.DPrintf("L %v Clerk %v IncrId %v Timeout", kv.me, opArgs.ClerkId, opArgs.IncrId)
		return result{Err: ErrHandleTimeout}
	case msg, success := <-newCh:
		if success && msg.ResTerm == sTerm {
			// raft.DPrintf("%v HandleOp Successful, ResTerm %v, Value: %v, Err: %v, LastId: %v", kv.me, msg.ResTerm, msg.Value, msg.Err, msg.LastId)
			return msg
		} else if !success {
			// 通道关闭
			raft.DPrintf("L %v Clerk %v IncrId %v Channel Close", kv.me, opArgs.ClerkId, opArgs.IncrId)
			return result{Err: ErrChanClosed}
		} else {
			// 任期变更
			raft.DPrintf("L %v Clerk %v IncrId %v Leader Changed", kv.me, opArgs.ClerkId, opArgs.IncrId)
			return result{Err: ErrLeaderChanged, Value: ""}
		}
	}

}

func (kv *KVServer) ApplyHandler() {
	for !kv.killed() {
		// 循环将从节点提交的条目应用到状态机
		_log := <-kv.applyCh
		if _log.CommandValid {
			// op, ok := _log.Command.(raft.Op)
			// if !ok {
			// 	raft.DPrintf("L %v: Raft Log Convert Failed, Type Incompact", kv.me)
			// }
			op := _log.Command
			// else {
			// 	raft.DPrintf("L %v: Get the Command Successful", kv.me)
			// }
			kv.mu.Lock()
			if _log.CommandIndex <= kv.lastApplied {
				kv.mu.Unlock()
				continue
			}
			kv.lastApplied = _log.CommandIndex

			// TODO： CommandIndex 超过了lastApplied，处理ReadIndex队列
			go kv.processReadIndex(_log.CommandIndex)

			// 首先判断此 log 是否需要被应用
			var res result
			need := false
			if clerkMap, exist := kv.history[op.ClerkId]; exist {
				if clerkMap.LastId == op.IncrId {
					// 用历史记录
					res = clerkMap
				} else if clerkMap.LastId < op.IncrId {
					need = true
				}
			} else {
				need = true
			}
			// _, isLeader := kv.rf.GetState()
			if need {
				// raft.DPrintf("%v DBExecute, IsLeader: %v", kv.me, isLeader)
				res = kv.DBExecute(op)
				res.ResTerm = _log.SnapshotTerm
				kv.history[op.ClerkId] = res // 更新历史
			}
			// if !isLeader {
			// 	// 如果不是Leader，检查下个log
			// 	kv.mu.Unlock()
			// 	continue
			// }
			// Leader 需要额外通知handler处理clerk回复
			ch, exist := kv.waitCh[_log.CommandIndex]
			if exist {
				// kv.mu.Unlock()
				func() {
					defer func() {
						if recover() != nil {
							raft.DPrintf("L %v ApplyHandler Find Clerk %v IncrId %v Channel Closed", kv.me, op.ClerkId, op.IncrId)
						}
					}()
					res.ResTerm = _log.SnapshotTerm
					*ch <- res
				}()
				// kv.mu.Lock()
			}
			if kv.maxraftstate != -1 && kv.persister.RaftStateSize() >= kv.maxraftstate*RaftStateNumThreshold/100 {
				//  TODO 生成快照
				// raft.DPrintf("RaftStateSize: %v", kv.persister.RaftStateSize())
				snapshot := kv.Snapshot()
				kv.rf.Snapshot(_log.CommandIndex, snapshot)
			}

			kv.mu.Unlock()
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
}

func (kv *KVServer) processReadIndex(commitIndex int64) {
	i := 0
	kv.getmu.Lock()
	for ; i < len(kv.readIdxQue) && commitIndex >= kv.readIdxQue[i].ReadIdx; i++ {
		kv.mu.Lock()
		res := kv.DBExecute(kv.readIdxQue[i].Op)
		kv.mu.Unlock()
		*(kv.readIdxQue[i].ReadIdxCh) <- res
	}
	kv.readIdxQue = kv.readIdxQue[i:]
	kv.getmu.Unlock()
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
		raft.DPrintf("%v Snapshot is Empty", kv.me)
		return
	}

	buffer := bytes.NewBuffer(snapshot)
	decoder := labgob.NewDecoder(buffer)

	db := make(map[string]string)
	historyDb := make(map[int32]result)
	if decoder.Decode(&db) != nil || decoder.Decode(&historyDb) != nil {
		// raft.DPrintf("%v Decode Snapshot Failed", kv.me)
	} else {
		kv.db = db
		kv.history = historyDb
		// raft.DPrintf("%v Decode Snapshot Successfully", kv.me)
	}
}

func (kv *KVServer) DBExecute(op *raft.Op) (res result) {
	// 需在加锁状态下调用
	raft.DPrintf("%v Execute DBExecute, IncrId: %v, %v(%v: %v)", kv.me, op.IncrId, optype(op.Optype), op.Key, op.Value)
	res.LastId = op.IncrId
	switch op.Optype {
	case GET:
		val, exist := kv.db[op.Key]
		if exist {
			res.Value = val
			// TODO: 记录请求成功
			return
		} else {
			res.Err = ErrNoKey
			res.Value = ""
			// TODO: 记录请求失败
			return
		}
	case PUT:
		kv.db[op.Key] = op.Value
		// TODO：记录
		return
	case APPEND:
		val, exist := kv.db[op.Key]
		if exist {
			kv.db[op.Key] = val + op.Value
			// TODO：记录
			return
		} else {
			kv.db[op.Key] = op.Value
			// TODO：记录
			return
		}
	}
	return
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

func (kv *KVServer) serve(lis net.Listener) {
	s := grpc.NewServer()
	RegisterServiceServer(s, kv)
	raft.DPrintf("Server %v listening at %v", kv.me, lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Server %v failed to serve: %v", kv.me, err)
	}
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
func StartKVServer(peerAddrs []string, me int32, persister *raft.Persister, addr string, lis net.Listener, slis net.Listener, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(raft.Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.lastApplied = 0
	kv.persister = persister

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(peerAddrs, me, persister, kv.applyCh, addr, lis)

	// You may need initialization code here.
	kv.dead = 0
	kv.history = make(map[int32]result)
	kv.waitCh = map[int64]*chan result{}
	kv.db = map[string]string{}
	kv.mu.Lock()
	kv.LoadSnapshot(persister.ReadSnapshot())
	kv.mu.Unlock()
	go kv.serve(slis)
	// go persister.SaveSchedule()
	go kv.ApplyHandler()

	return kv
}
