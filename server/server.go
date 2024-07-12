package kvraft

import (
	"bytes"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raft"
)

const RaftStateNumThreshold = 90
const Debug = false
const HandleTimeout = time.Duration(1000) * time.Millisecond
const MaxMapLen = 10000

const (
	GET int = iota
	PUT
	APPEND
)

func optype(op int) string {
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

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type result struct {
	LastId  uint64
	Err     Err
	Value   string
	ResTerm int // commit时的Term
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Optype  int
	IncrId  uint64 // 全局递增的Id，用于区分请求的新旧
	ClerkId int    // 客户端 Id，与全局递增 Id 拼接以标识唯一消息
	Key     string
	Value   string
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	waitCh  map[int]*chan result // 接收Raft提交条目的通道
	history map[int]result       // 存储此前的历史结果

	maxraftstate int // snapshot if log grows this big
	maxMapLen    int
	db           map[string]string
	lastApplied  int // apply的最后一个下标
	persister    *raft.Persister
	// Your definitions here.
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	opArgs := &Op{
		Optype:  GET,
		IncrId:  args.IncrId,
		ClerkId: args.ClerkId,
		Key:     args.Key,
	}
	res := kv.HandleOp(opArgs)
	reply.Err = res.Err
	reply.Value = res.Value
	DPrintf("%v Get( %v) Result(Err: %v, Key: %v, Value: %v)", kv.me, args.IncrId, res.Err, args.Key, res.Value)
}

func (kv *KVServer) Put(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	opArgs := &Op{
		Optype:  PUT,
		IncrId:  args.IncrId,
		ClerkId: args.ClerkId,
		Key:     args.Key,
		Value:   args.Value,
	}
	res := kv.HandleOp(opArgs)
	DPrintf("%v Put( %v) Result(Err: %v, Key: %v, Value: %v)", kv.me, args.IncrId, res.Err, args.Key, args.Value)
	reply.Err = res.Err
}

func (kv *KVServer) Append(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	opArgs := &Op{
		Optype:  APPEND,
		IncrId:  args.IncrId,
		ClerkId: args.ClerkId,
		Key:     args.Key,
		Value:   args.Value,
	}
	res := kv.HandleOp(opArgs)
	DPrintf("%v Append( %v) Result(Err: %v, Key: %v, Value: %v)", kv.me, args.IncrId, res.Err, args.Key, args.Value)
	reply.Err = res.Err
}

func (kv *KVServer) HandleOp(opArgs *Op) result {
	kv.mu.Lock()
	if history, exist := kv.history[opArgs.ClerkId]; exist && history.LastId == opArgs.IncrId {
		kv.mu.Unlock()
		return history
	}
	kv.mu.Unlock()
	sIdx, sTerm, isLeader := kv.rf.Start(*opArgs)
	if !isLeader {
		return result{Err: ErrWrongLeader, Value: ""}
	}

	kv.mu.Lock()
	newCh := make(chan result, 1)
	kv.waitCh[sIdx] = &newCh
	// DPrintf("L %v Clerk %v IncrId %v create new Channel %p", kv.me, opArgs.ClerkId, opArgs.IncrId, &newCh)
	kv.mu.Unlock()

	defer func() {
		kv.mu.Lock()
		delete(kv.waitCh, sIdx)
		close(newCh)
		kv.mu.Unlock()
	}()

	select {
	case <-time.After(HandleTimeout):
		DPrintf("L %v Clerk %v IncrId %v Timeout", kv.me, opArgs.ClerkId, opArgs.IncrId)
		return result{Err: ErrHandleTimeout}
	case msg, success := <-newCh:
		if success && msg.ResTerm == sTerm {
			// DPrintf("%v HandleOp Successful, ResTerm %v, Value: %v, Err: %v, LastId: %v", kv.me, msg.ResTerm, msg.Value, msg.Err, msg.LastId)
			return msg
		} else if !success {
			// 通道关闭
			DPrintf("L %v Clerk %v IncrId %v Channel Close", kv.me, opArgs.ClerkId, opArgs.IncrId)
			return result{Err: ErrChanClosed}
		} else {
			// 任期变更
			DPrintf("L %v Clerk %v IncrId %v Leader Changed", kv.me, opArgs.ClerkId, opArgs.IncrId)
			return result{Err: ErrLeaderChanged, Value: ""}
		}
	}

}

func (kv *KVServer) ApplyHandler() {
	for !kv.killed() {
		// 循环将从节点提交的条目应用到状态机
		_log := <-kv.applyCh
		if _log.CommandValid {
			op, ok := _log.Command.(Op)
			if !ok {
				DPrintf("L %v: Raft Log Convert Failed, Type Incompact", kv.me)
			}
			// else {
			// 	DPrintf("L %v: Get the Command Successful", kv.me)
			// }
			kv.mu.Lock()
			if _log.CommandIndex <= kv.lastApplied {
				kv.mu.Unlock()
				continue
			}
			kv.lastApplied = _log.CommandIndex
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
				// DPrintf("%v DBExecute, IsLeader: %v", kv.me, isLeader)
				res = kv.DBExecute(&op)
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
							DPrintf("L %v ApplyHandler Find Clerk %v IncrId %v Channel Closed", kv.me, op.ClerkId, op.IncrId)
						}
					}()
					res.ResTerm = _log.SnapshotTerm
					*ch <- res
				}()
				// kv.mu.Lock()
			}
			if kv.maxraftstate != -1 && kv.persister.RaftStateSize() >= kv.maxraftstate*RaftStateNumThreshold/100 {
				//  TODO 生成快照
				// DPrintf("RaftStateSize: %v", kv.persister.RaftStateSize())
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

func (kv *KVServer) Snapshot() []byte {
	buffer := new(bytes.Buffer)
	encoder := labgob.NewEncoder(buffer)

	encoder.Encode(kv.db)
	encoder.Encode(kv.history)

	return buffer.Bytes()
}

func (kv *KVServer) LoadSnapshot(snapshot []byte) {
	if snapshot == nil || len(snapshot) < 1 {
		DPrintf("%v Snapshot is Empty", kv.me)
		return
	}

	buffer := bytes.NewBuffer(snapshot)
	decoder := labgob.NewDecoder(buffer)

	db := make(map[string]string)
	historyDb := make(map[int]result)
	if decoder.Decode(&db) != nil || decoder.Decode(&historyDb) != nil {
		// DPrintf("%v Decode Snapshot Failed", kv.me)
	} else {
		kv.db = db
		kv.history = historyDb
		// DPrintf("%v Decode Snapshot Successfully", kv.me)
	}
}

func (kv *KVServer) DBExecute(op *Op) (res result) {
	// 需在加锁状态下调用
	DPrintf("%v Execute DBExecute, IncrId: %v, %v(%v: %v)", kv.me, op.IncrId, optype(op.Optype), op.Key, op.Value)
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
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.lastApplied = 0
	kv.persister = persister

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.
	kv.dead = 0
	kv.history = make(map[int]result)
	kv.waitCh = map[int]*chan result{}
	kv.maxMapLen = MaxMapLen
	kv.db = map[string]string{}
	kv.mu.Lock()
	kv.LoadSnapshot(persister.ReadSnapshot())
	kv.mu.Unlock()

	go kv.ApplyHandler()

	return kv
}
