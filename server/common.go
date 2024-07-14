package server

const (
	OK               string = "OK"
	ErrNoKey         string = "ErrNoKey"
	ErrWrongLeader   string = "ErrWrongLeader"
	ErrHandleTimeout string = "ErrHandleTimeout"
	ErrChanClosed    string = "ErrChanClosed"
	ErrLeaderChanged string = "ErrLeaderChanged"
)

type Err string

// Put or Append
// type PutAppendArgs struct {
// 	Key   string
// 	Value string
// 	// You'll have to add definitions here.
// 	// Field names must start with capital letters,
// 	// otherwise RPC will break.
// 	IncrId  uint64
// 	ClerkId int32
// }

// type PutAppendReply struct {
// 	Err Err
// }

// type GetArgs struct {
// 	Key string
// 	// You'll have to add definitions here.
// 	IncrId  uint64
// 	ClerkId int32
// }

// type GetReply struct {
// 	Err   Err
// 	Value string
// }
