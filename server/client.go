package kvraft

import (
	"crypto/rand"
	"math/big"
	"sync"
	"time"

	"6.5840/labrpc"
)

var id = 100
var mu sync.Mutex

const RPCRetryTime = time.Millisecond * 5

type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	leader  int
	incrId  uint64
	clerkId int
}

func (c *Clerk) getUid() uint64 {
	t := time.Now()
	timestamp := uint64(t.UTC().UnixNano())
	// timestamp := strconv.FormatInt(nrand(), 10)
	return timestamp
}

func (c *Clerk) getClerkId() int {
	mu.Lock()
	defer mu.Unlock()
	id++
	return id
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	// You'll have to add code here.
	ck.leader = 0
	ck.incrId = ck.getUid()
	ck.clerkId = ck.getClerkId()
	return ck
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
func (ck *Clerk) Get(key string) string {

	// You will have to modify this function.
	args := GetArgs{
		Key:     key,
		IncrId:  ck.getUid(),
		ClerkId: ck.clerkId,
	}
	for {
		reply := GetReply{}
		// log.Printf("Clerk %v Send Get Request(Key: %v) to %v", ck.clerkId, key, ck.leader)
		ok := ck.servers[ck.leader].Call("KVServer.Get", &args, &reply)
		if !ok || reply.Err == ErrWrongLeader || reply.Err == ErrLeaderChanged {
			ck.leader = (ck.leader + 1) % len(ck.servers)
			// time.Sleep(RPCRetryTime)
			continue
		}
		switch reply.Err {
		case ErrHandleTimeout:
			time.Sleep(RPCRetryTime)
			continue
		case ErrChanClosed:
			time.Sleep(RPCRetryTime)
			continue
		case ErrNoKey:
			return reply.Value
		}
		return reply.Value
	}
	// return reply.Value
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.
	args := PutAppendArgs{
		Key:     key,
		Value:   value,
		IncrId:  ck.getUid(),
		ClerkId: ck.clerkId,
	}
	for {
		reply := PutAppendReply{}
		ok := ck.servers[ck.leader].Call("KVServer."+op, &args, &reply)
		// log.Printf("Clerk %v Send %v to %v Request(Key: %v,Value: %v), Err: %v", ck.clerkId, op, ck.leader, key, value, reply.Err)
		if !ok || reply.Err == ErrWrongLeader || reply.Err == ErrLeaderChanged {
			ck.leader = (ck.leader + 1) % len(ck.servers)
			// time.Sleep(RPCRetryTime)
			continue
		}
		switch reply.Err {
		case ErrHandleTimeout:
			time.Sleep(RPCRetryTime)
			continue
		case ErrChanClosed:
			time.Sleep(RPCRetryTime)
			continue
		}
		return
	}

}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
