package server

type Err string

const (
	OK               string = "OK"
	ErrNoKey         string = "ErrNoKey"
	ErrWrongLeader   string = "ErrWrongLeader"
	ErrLeaderChanged string = "ErrLeaderChanged"
	ErrHandleTimeout string = "ErrHandleTimeout"
	ErrChanClosed    string = "ErrChanClosed"
)

var ErrMap map[string]uint32 = map[string]uint32{
	OK:               200,
	ErrNoKey:         100,
	ErrWrongLeader:   101,
	ErrLeaderChanged: 102,
	ErrHandleTimeout: 103,
	ErrChanClosed:    104,
}

var ErrIdMap map[uint32]string = map[uint32]string{
	200: OK,
	100: ErrNoKey,
	101: ErrWrongLeader,
	102: ErrLeaderChanged,
	103: ErrHandleTimeout,
	104: ErrChanClosed,
}

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
