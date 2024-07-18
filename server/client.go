package server

import (
	"context"
	"errors"
	"fmt"
	"raft-kv-service/raft"
	"sync"
	"time"

	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var id int32 = 100
var mu sync.Mutex

const RPCRetryTime = time.Millisecond * 5
const WrongLeaderRetryTime = time.Millisecond * 2

type Clerk struct {
	servers []ServiceClient
	// You will have to modify this struct.
	leader  int
	incrId  uint64
	clerkId int32
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

func (c *Clerk) connectToServer(addrs []string) error {
	failed := make([]int, 0)
	for i, addr := range addrs {
		raft.DPrintf("Client Try Connect to Server %v: %s", i, addr)
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			raft.DPrintf("Client did not connect to %s: %v", addr, err)
			failed = append(failed, i)
			continue
		}
		if conn == nil {
			return fmt.Errorf("no server listening at %s", addr)
		}
		c.servers[i] = NewServiceClient(conn)
	}

	for len(failed) > 0 {
		i := failed[len(failed)-1]
		conn, err := grpc.NewClient(addrs[i], grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			raft.DPrintf("Client Retry connect to %s: %v", addrs[i], err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		c.servers[i] = NewServiceClient(conn)
		failed = failed[:len(failed)-1]
	}
	return nil
}

func MakeClerk(addrs []string) *Clerk {
	ck := new(Clerk)
	ck.servers = make([]ServiceClient, len(addrs))
	err := ck.connectToServer(addrs)
	if err != nil {
		panic(err)
	}
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
func (ck *Clerk) Get(key string) (string, error) {

	// You will have to modify this function.
	args := GetArgs{
		Key:     key,
		IncrId:  ck.getUid(),
		ClerkId: ck.clerkId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	retryTime := 100
	for {
		raft.DPrintf("Clerk %v Send Get Request(Key: %v) to %v", ck.clerkId, key, ck.leader)
		reply, err := ck.servers[ck.leader].Get(ctx, &args)
		if err != nil {
			raft.DPrintf("Client %v Get Err: %v", ck.clerkId, err)
			return "", err
		}
		retryTime--
		if retryTime < 0 {
			return reply.Value, errors.New("MaxRetry Times")
		}
		if reply.Err == ErrWrongLeader || reply.Err == ErrLeaderChanged {
			ck.leader = (ck.leader + 1) % len(ck.servers)
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
			return reply.Value, nil
		}
		return reply.Value, nil
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
func (ck *Clerk) Put(key string, value string) error {
	// You will have to modify this function.
	args := PutAppendArgs{
		Key:     key,
		Value:   value,
		IncrId:  ck.getUid(),
		ClerkId: ck.clerkId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	retryTime := 100
	for {
		reply, err := ck.servers[ck.leader].Put(ctx, &args)
		if err != nil {
			raft.DPrintf("Client %v Put Err: %v", ck.clerkId, err)
			return err
		}
		raft.DPrintf("Clerk %v Send %v Put Request(Key: %v,Value: %v), Err: %v", ck.clerkId, ck.leader, key, value, reply.Err)
		retryTime--
		if retryTime < 0 {
			return errors.New("MaxRetry Times")
		}
		if reply.Err == ErrWrongLeader || reply.Err == ErrLeaderChanged {
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
		return nil
	}
}

func (ck *Clerk) Append(key string, value string) error {
	// You will have to modify this function.
	args := PutAppendArgs{
		Key:     key,
		Value:   value,
		IncrId:  ck.getUid(),
		ClerkId: ck.clerkId,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	retryTime := 100
	for {
		reply, err := ck.servers[ck.leader].Append(ctx, &args)
		if err != nil {
			raft.DPrintf("Client %v Append Err: %v", ck.clerkId, err)
			return err
		}
		raft.DPrintf("Clerk %v Send %v Append Request(Key: %v,Value: %v), Err: %v", ck.clerkId, ck.leader, key, value, reply.Err)
		retryTime--
		if retryTime < 0 {
			return errors.New("MaxRetry Times")
		}
		if reply.Err == ErrWrongLeader || reply.Err == ErrLeaderChanged {
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
		return nil
	}
}

// func (ck *Clerk) Put(key string, value string) error {
// 	return ck.PutAppend(key, value, ck.servers[ck.leader].Put)
// }
// func (ck *Clerk) Append(key string, value string) error {
// 	return ck.PutAppend(key, value, ck.servers[ck.leader].Append)
// }
