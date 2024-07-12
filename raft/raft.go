package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"bytes"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"6.5840/labgob"
	"6.5840/labrpc"
)

const ImmidiateTime = 3
const HeartbeatTime = 50
const ElectionTimeoutRangeBottom = 150
const ElectionTimeoutRangeTop = 300
const CheckTimeInter = 15

func RandomElectionTimeout() int {
	return ElectionTimeoutRangeBottom + rand.Intn(ElectionTimeoutRangeTop-ElectionTimeoutRangeBottom)
}

func HeartbeatTimeThreshold() int {
	// return (int)(HeartbeatTime * 1.5) // + rand.Intn(HeartbeatTime)
	// return HeartbeatTime + rand.Intn(HeartbeatTime)
	return ElectionTimeoutRangeTop + rand.Intn(HeartbeatTime)
}

const (
	Leader int = iota
	Follower
	Candidate
)

type Entry struct {
	Term    int
	Command interface{}
}

func roleName(idx int) string {
	switch idx {
	case Leader:
		return "L"
	case Follower:
		return "F"
	case Candidate:
		return "C"
	default:
		return "Unknown"
	}
}

func AppendOrHeartbeat(entries []Entry) string {
	if entries == nil {
		return "Heartbeat"
	} else {
		return "AppendEntries"
	}
}

func (rf *Raft) ResetHeartTimer(timeGap int) {
	rf.HeartbeatTimer.Reset(time.Duration(timeGap) * time.Millisecond)
}

// 调用转换时需要加锁
func (rf *Raft) GlobalToLocal(globalIndex int) int {
	return globalIndex - rf.lastIncludedIndex
}

func (rf *Raft) LocalToGlobal(localIndex int) int {
	return localIndex + rf.lastIncludedIndex
}

type InstallSnapshotArgs struct {
	Term                int // Leader Term
	LeaderId            int
	LastIncludedIndex   int
	LastIncludedTerm    int
	LastIncludedCommand interface{}
	Data                []byte
	Done                bool
}

type InstallSnapshotReply struct {
	Term int // currentTerm
}

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 3D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 3D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (3A, 3B, 3C).
	// Look at the paper's Figure 2 for a description of what

	// state a Raft server must maintain.
	applyCh     chan ApplyMsg
	log         []Entry // 日志
	currentTerm int     // 最新任期
	votedFor    int     // 收到的投票请求的候选者Id

	commitIndex int // 已提交的最高Index
	lastApplied int // 提交到状态机的最高Index

	nextIndex  []int // 用全局索引
	matchIndex []int // 用全局索引

	heartbeatTimeStamp time.Time
	HeartbeatTimer     *time.Timer // RPC不能过多，且提交速度要快，因此不采用固定周期心跳
	electionTimeStamp  time.Time   // 选举开始时间戳
	electionTimeout    int
	role               int
	voteCount          int

	condApply *sync.Cond

	// 3D SnapShot
	snapShot          []byte // 快照
	lastIncludedIndex int    // 日志中的最高索引
	lastIncludedTerm  int    // 日志中的最高Term
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	// Your code here (3A).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	isleader := rf.role == Leader
	term := rf.currentTerm
	return term, isleader
}

func (rf *Raft) CommitCheck() {
	for !rf.killed() {
		// DPrintf("%v %v CommitCheck Try Get Lock", roleName(rf.role), rf.me)
		rf.mu.Lock()
		// DPrintf("%v %v CommitCheck Get the Lock", roleName(rf.role), rf.me)
		// 使用缓存，不然导致死锁
		for rf.commitIndex <= rf.lastApplied {
			rf.condApply.Wait()
		}
		buffer := make([]ApplyMsg, 0, rf.commitIndex-rf.lastApplied)
		tmpApplied := rf.lastApplied
		for rf.commitIndex > tmpApplied {
			tmpApplied++
			message := ApplyMsg{
				CommandValid: rf.lastIncludedIndex < tmpApplied,
				Command:      rf.log[rf.GlobalToLocal(tmpApplied)].Command,
				CommandIndex: tmpApplied,
				SnapshotTerm: rf.log[rf.GlobalToLocal(tmpApplied)].Term,
			}
			buffer = append(buffer, message)
			DPrintf("%v %v add Command %v(Idx: %v) to Buffer", roleName(rf.role), rf.me, message.Command, message.CommandIndex)
		}
		rf.mu.Unlock()

		// 解锁后，可能出现 SnapShot，协程修改 rf.lastApplied
		for _, msg := range buffer {
			rf.mu.Lock()
			if msg.CommandIndex != rf.lastApplied+1 {
				rf.mu.Unlock()
				continue
			}
			currentLastApplied := rf.lastApplied
			rf.mu.Unlock()
			// DPrintf("%v %v CommitCheck UnLock", roleName(rf.role), rf.me)
			rf.applyCh <- msg
			// DPrintf("%v %v CommitCheck try Get Lock", roleName(rf.role), rf.me)
			rf.mu.Lock()
			// DPrintf("%v %v Commited Command %v(Idx: %v) from Buffer", roleName(rf.role), rf.me, msg.Command, msg.CommandIndex)
			if msg.CommandIndex != currentLastApplied+1 {
				rf.mu.Unlock()
				continue
			}
			rf.lastApplied = max(rf.lastApplied, msg.CommandIndex)
			rf.mu.Unlock()
		}
	}
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (3C).
	// Example:
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.votedFor)
	e.Encode(rf.currentTerm)
	e.Encode(rf.log)

	e.Encode(rf.lastIncludedIndex)
	e.Encode(rf.lastIncludedTerm)
	raftstate := w.Bytes()
	// DPrintf("%v %v persist()", roleName(rf.role), rf.me)
	rf.persister.Save(raftstate, rf.snapShot)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (3C).
	// Example:
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var logData []Entry
	var votedFor int
	var currentTerm int
	var lastIncludedIndex int
	var lastIncludedTerm int
	errVotedFor := d.Decode(&votedFor)
	errCurrentTerm := d.Decode(&currentTerm)
	errLog := d.Decode(&logData)
	errLastIncludedIndex := d.Decode(&lastIncludedIndex)
	errLastIncludedTerm := d.Decode(&lastIncludedTerm)
	if errLog != nil || errVotedFor != nil || errCurrentTerm != nil || errLastIncludedIndex != nil || errLastIncludedTerm != nil {
		if errLog != nil {
			DPrintf("Server %v readPersist error: %v", rf.me, errLog)
		}
		if errVotedFor != nil {
			DPrintf("Server %v readPersist error: %v", rf.me, errVotedFor)
		}
		if errCurrentTerm != nil {
			DPrintf("Server %v readPersist error: %v", rf.me, errCurrentTerm)
		}
		if errLastIncludedIndex != nil {
			DPrintf("Server %v readPersist error: %v", rf.me, errLastIncludedIndex)
		}
		if errLastIncludedTerm != nil {
			DPrintf("Server %v readPersist error: %v", rf.me, errLastIncludedTerm)
		}
	} else {
		// rf.mu.Lock()
		DPrintf("%v %v readPersist()", roleName(rf.role), rf.me)
		rf.log = logData
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.lastIncludedIndex = lastIncludedIndex
		rf.lastIncludedTerm = lastIncludedTerm

		rf.commitIndex = lastIncludedIndex
		rf.lastApplied = lastIncludedIndex
		// rf.mu.Unlock()
	}
}

func (rf *Raft) readSnapshot(data []byte) {
	if len(data) < 1 {
		DPrintf("%v readSnapShot Fail", rf.me)
		return
	}
	rf.snapShot = data
	DPrintf("%v readSnapShot Success", rf.me)
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (3D).
	// 1. 判断是否接受 Snapshot
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if rf.commitIndex < index || index <= rf.lastIncludedIndex {
		DPrintf("%v %v(ComitIdx: %v,LastIncludIdx: %v) Reject Snapshot, Idx %v", roleName(rf.role), rf.me, rf.commitIndex, rf.lastIncludedIndex, index)
		return
	}
	// 2. 将 Snapshot 保存，Follower 可能需要 Snapshot，持久化需要 Snapshot 保存
	rf.snapShot = snapshot
	// DPrintf("%v %v(ComitIdx: %v,LastIncludIdx: %v) Receive Snapshot, Idx %v, Local: %v, LogLen: %v", roleName(rf.role), rf.me, rf.commitIndex, rf.lastIncludedIndex, index, rf.GlobalToLocal(index), len(rf.log))
	rf.lastIncludedTerm = rf.log[rf.GlobalToLocal(index)].Term
	// 3. 截断 log
	// DPrintf("%v %v(ComitIdx: %v,LastIncludIdx: %v) Start Snapshot, Idx: %v,BeforeCut: %v", roleName(rf.role), rf.me, rf.commitIndex, rf.lastIncludedIndex, index, len(rf.log))
	rf.log = rf.log[rf.GlobalToLocal(index):]
	rf.lastIncludedIndex = index
	if rf.lastApplied < index {
		rf.lastApplied = index
	}
	// 4. 调用 persist
	// DPrintf("%v %v(ComitIdx: %v,LastIncludIdx: %v) Start Snapshot, Idx: %v,LogCutTo: %v", roleName(rf.role), rf.me, rf.commitIndex, rf.lastIncludedIndex, index, len(rf.log))
	rf.persist()
}

// example RequestVote RPC arguments structure.
// field names must start with capital letters!
type RequestVoteArgs struct {
	// Your data here (3A, 3B).
	Term         int // 候选者的任期
	CandidateId  int
	LastLogIndex int // 候选人最后一个日志的下标
	LastLogTerm  int // 候选人最后一个日志的任期
}

// example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (3A).
	Term        int  // 当前任期，用于候选人更新
	VoteGranted bool // 候选人是否收到投票
}

// example RequestVote RPC handler.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (3A, 3B).
	// DPrintf("%v(V: %v) << %v's requestVote", rf.me, rf.votedFor, args.CandidateId)
	rf.mu.Lock()
	DPrintf("%v %v(T: %v, V: %v) <<< C %v(T: %v), Try Lock", roleName(rf.role), rf.me, rf.currentTerm, rf.votedFor, args.CandidateId, args.Term)
	defer rf.mu.Unlock()
	// DPrintf("%v %v(T: %v, V: %v) <<< C %v(T: %v), Get Lock", roleName(rf.role), rf.me, rf.currentTerm, rf.votedFor, args.CandidateId, args.Term)
	if args.Term < rf.currentTerm {
		// 1. Candidate 的任期小于 Follower 任期
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		// DPrintf("%v %v(T: %v, LastLogT: %v, V: %v)  X  C %v(T: %v,LastLogT: %v) Term\n", roleName(rf.role), rf.me, rf.currentTerm, rf.log[len(rf.log)-1].Term, rf.votedFor, args.CandidateId, args.Term, args.LastLogTerm)
		return
	}

	if args.Term > rf.currentTerm {
		// 新一轮投票，需要取消上一轮的投票
		// DPrintf("请求(T: %v)的Term大于自己的Term(T: %v), %v %v自己降级为Follower", args.Term, rf.currentTerm, roleName(rf.role), rf.me)
		rf.votedFor = -1
		rf.currentTerm = args.Term // 需要将自己的Term更新，以防止再次开启一轮投票
		rf.role = Follower
		rf.persist()
	}

	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		// if votedFor is null or candidateId
		if args.LastLogTerm > rf.log[len(rf.log)-1].Term || (args.LastLogIndex >= rf.LocalToGlobal(len(rf.log)-1) && args.LastLogTerm == rf.log[len(rf.log)-1].Term) {
			// 需要防止有旧log的candidate选举成功，从而覆盖其他log
			// and candidate's log is at least as up-to-date as receiver's log, grant vote
			rf.currentTerm = args.Term
			rf.votedFor = args.CandidateId
			reply.Term = rf.currentTerm
			rf.persist()
			rf.role = Follower
			rf.heartbeatTimeStamp = time.Now()
			// rf.ResetHeartTimer(HeartbeatTimeThreshold())

			reply.VoteGranted = true
			// rf.mu.Unlock()
			DPrintf("%v %v(T: %v, LastLogT: %v, LastLogI: %v, V: %v)  V  C %v(T: %v, LastLogT: %v, LastLogI: %v)\n", roleName(rf.role), rf.me, rf.currentTerm, rf.log[len(rf.log)-1].Term, rf.LocalToGlobal(len(rf.log)-1), rf.votedFor, args.CandidateId, args.Term, args.LastLogTerm, args.LastLogIndex)
			return
		}
	}
	// DPrintf("%v(T: %v, V: %v)  X  %v(T: %v)\n", rf.me, rf.currentTerm, rf.votedFor, args.CandidateId, args.Term)

	reply.Term = rf.currentTerm
	// DPrintf("%v %v(T: %v, LastLogT: %v, LastLogI: %v, V: %v)  X  C %v(T: %v, LastLogT: %v,LastLogI: %v) Already Voted or LastLogTerm\n", roleName(rf.role), rf.me, rf.currentTerm, rf.log[len(rf.log)-1].Term, rf.LocalToGlobal(len(rf.log)-1), rf.votedFor, args.CandidateId, args.Term, args.LastLogTerm, args.LastLogIndex)
	reply.VoteGranted = false
}

type AppendEntriesArgs struct {
	Term         int     // leader's term
	LeaderId     int     //follower can redirect clients
	PrevLogIndex int     // index of log entry immediately preceding new ones
	PrevLogTerm  int     // term of prevLogIndex entry
	Entries      []Entry //log entries to store (empty for heartbeat, more than one for efficiency)
	LeaderCommit int     // leader's commitIndex
}

type AppendEntriesReply struct {
	Term    int  // currentTerm, for leader to update itself
	Success bool // true if follower contained entry matching prevLogIndex and prevLogTerm
	XTerm   int  // term in the conflicting entry (if any) -1 if no log at XIndex
	XIndex  int  // index of first entry with conflicting term (if any)
	XLen    int  // blank log situations, 即Follower差RPC的进度，Entries中间缺失的长度
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		// 1. Reply false if term < currentTerm
		// DPrintf("Illegal%v: L %v(T: %v, PreLogIdx: %v, PreLogTerm: %v) XXX F %v(T: %v, LastLogIdx: %v, LastLogTerm: %v)\n", AppendOrHeartbeat(args.Entries), args.LeaderId, args.Term, args.PrevLogIndex, args.PrevLogTerm, rf.me, rf.currentTerm, rf.LocalToGlobal(len(rf.log)-1), rf.log[len(rf.log)-1].Term)
		reply.Term = rf.currentTerm
		// rf.mu.Unlock()
		reply.Success = false
		return
	}

	// 重置Follower的心跳时间
	rf.heartbeatTimeStamp = time.Now()
	// rf.ResetHeartTimer(HeartbeatTimeThreshold())

	if args.Term > rf.currentTerm {
		// 新Leader的消息
		rf.currentTerm = args.Term // 更新Term
		rf.votedFor = -1           // 新Leader已经产生，消除之前的投票记录
		rf.persist()
		rf.role = Follower // 心跳抑制投票
	}

	reply.Term = rf.currentTerm

	isConflict := false
	// 3. An existing entry conflicts with a new one(same index but different term),
	// delete the existing entry and all that follow it
	if args.PrevLogIndex >= rf.LocalToGlobal(len(rf.log)) {
		// PrevLogIndex 位置不存在日志项
		reply.XTerm = -1
		reply.XLen = rf.LocalToGlobal(len(rf.log)) // Log 长度
		isConflict = true
		DPrintf("F %v(T: %v,LastLog: %v) doesn't contain L %v(T: %v,FirLog: %v)", rf.me, rf.currentTerm, rf.LocalToGlobal(len(rf.log)-1), args.LeaderId, args.Term, args.PrevLogIndex+1)
	} else if args.PrevLogIndex < rf.lastIncludedIndex {
		// PrevLogIndex 位置在 Follower 的快照中，快照部分不动，log部分的Term无法验证，因此重新复制log
		reply.XTerm = -1
		reply.XLen = rf.lastIncludedIndex + 1
		isConflict = true
		DPrintf("F %v(T: %v,LastIncludedLog: %v) Snapshot Contains L %v(T: %v,FirLog: %v)", rf.me, rf.currentTerm, rf.lastIncludedIndex, args.LeaderId, args.Term, args.PrevLogIndex+1)
	} else if rf.commitIndex == args.PrevLogIndex {
		// 排除Follower占位log的Term不匹配导致的错误
	} else if rf.log[rf.GlobalToLocal(args.PrevLogIndex)].Term != args.PrevLogTerm {
		// PrevLogIndex 位置的日志项存在，但term不匹配
		reply.XTerm = rf.log[rf.GlobalToLocal(args.PrevLogIndex)].Term
		i := args.PrevLogIndex
		for i > rf.lastIncludedIndex && rf.log[rf.GlobalToLocal(i)].Term == reply.XTerm {
			i -= 1
		}
		reply.XIndex = i + 1
		reply.XLen = rf.LocalToGlobal(len(rf.log))
		isConflict = true
		DPrintf("F %v(T: %v,LastLogT: %v,LastI: %v,LastIncludIdx: %v) Conflict L %v(T: %v,PreLogT: %v,RreI: %v)", rf.me, rf.currentTerm, rf.log[rf.GlobalToLocal(args.PrevLogIndex)].Term, rf.LocalToGlobal(len(rf.log)-1), rf.lastIncludedIndex, args.LeaderId, args.Term, args.PrevLogTerm, args.PrevLogIndex)
	}

	if isConflict {
		// 2. Reply false log doesn't contain an entry at prevLogIndex whose term mathces prevLogTerm
		reply.Term = rf.currentTerm
		reply.Success = false
		// rf.mu.Unlock()
		return
	}

	// 4. Append any new entries not already in the log
	if args.Entries != nil && rf.role != Leader {

		for idx, log := range args.Entries {
			ridx := rf.GlobalToLocal(args.PrevLogIndex) + 1 + idx
			if ridx < len(rf.log) && rf.log[ridx].Term != log.Term {
				DPrintf("%v: Remove Logs After(T:%v,I:%v,Val:%v)", rf.me, rf.log[ridx].Term, args.PrevLogIndex+1+idx, rf.log[ridx].Command)
				// rf.log = rf.log[:ridx]
				rf.log = append(rf.log[:ridx], args.Entries[idx:]...)
				// DPrintf("%v %v Append New Entries From %v: First(T:%v,I:%v,Val:%v) - Last(T:%v,I:%v,Val:%v)", roleName(rf.role), rf.me, args.LeaderId, rf.log[rf.GlobalToLocal(args.PrevLogIndex+1)].Term, rf.GlobalToLocal(args.PrevLogIndex+1), args.Entries[0].Command, rf.log[len(rf.log)-1].Term, rf.LocalToGlobal(len(rf.log)-1), rf.log[len(rf.log)-1].Command)
				break
			} else if ridx == len(rf.log) {
				rf.log = append(rf.log, args.Entries[idx:]...)
				DPrintf("%v:(I:%v,Cmd:%v) Append Logs From %v First(T:%v,I:%v,Val:%v) - Last(T:%v,I:%v,Val:%v)", rf.me, rf.LocalToGlobal(len(rf.log)-1), rf.log[len(rf.log)-1].Command, args.LeaderId, args.Entries[idx].Term, ridx, args.Entries[idx].Command, args.Entries[len(args.Entries)-1].Term, ridx+len(args.Entries)-idx-1, args.Entries[len(args.Entries)-1].Command)
				break
			}
		}
	}
	rf.persist()
	reply.Success = true
	reply.Term = rf.currentTerm
	reply.XIndex = rf.lastIncludedIndex

	// if args.LeaderCommit > rf.commitIndex && args.PrevLogIndex == rf.LocalToGlobal(len(rf.log)-1) && args.PrevLogTerm == rf.log[len(rf.log)-1].Term {
	if args.LeaderCommit > rf.commitIndex {
		// 5. LeaderCommit > commitIndex, set commitIndex = min(leaderCommit, index of last new entry)
		rf.commitIndex = (int)(math.Min((float64)(args.LeaderCommit), (float64)(rf.LocalToGlobal(len(rf.log)-1))))
		DPrintf("F %v update commitIndex to %v", rf.me, rf.commitIndex)
		rf.condApply.Signal()
	}
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntriesToServer(server int, args *AppendEntriesArgs) {
	// DPrintf("L %v sendAppendEntriesToServer try get Lock %v", rf.me, server)
	rf.mu.Lock()
	reply := &AppendEntriesReply{}
	if rf.role != Leader {
		rf.mu.Unlock()
		return
	}
	rf.mu.Unlock()
	ok := rf.sendAppendEntries(server, args, reply)
	if !ok {
		DPrintf("Connect fail From %v >>> %v", rf.me, server)
		return
	}
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if args.Term != rf.currentTerm {
		// 可能发送心跳期间，任期更改
		return
	}

	if reply.Success {
		DPrintf("%v %v Update %v matchIdx %v > %v, nextIdx %v > %v", roleName(rf.role), rf.me, server, rf.matchIndex[server], max(rf.matchIndex[server], args.PrevLogIndex+len(args.Entries)), rf.nextIndex[server], args.PrevLogIndex+len(args.Entries)+1)
		rf.matchIndex[server] = max(rf.matchIndex[server], args.PrevLogIndex+len(args.Entries))
		// rf.matchIndex[server] = args.PrevLogIndex + len(args.Entries)
		rf.nextIndex[server] = rf.matchIndex[server] + 1
		if rf.lastIncludedIndex > reply.XIndex {
			go rf.sendInstallSnapshotToServer(server)
		}
		// DPrintf("Check log for Commit L %v(ComIdx: %v,LastLogIdx: %v) >>> F %v matchIdx: %v\n", rf.me, rf.commitIndex, rf.LocalToGlobal(len(rf.log)-1), server, rf.matchIndex[server])
		// 需要检查log复制数是否超过半数，判断是否可以提交
		commitLastLog := rf.LocalToGlobal(len(rf.log) - 1)
		for commitLastLog > rf.commitIndex {
			// 找到一个提交超过半数的日志
			count := 1
			for i := 0; i < len(rf.peers); i++ {
				if i == rf.me {
					continue
				}
				// 加了 rf.log[commitLastLog].Term == rf.currentTerm，只有当前任期的日志才能提交
				if rf.matchIndex[i] >= commitLastLog && rf.log[rf.GlobalToLocal(commitLastLog)].Term == rf.currentTerm {
					count++
				}
			}
			// DPrintf("ComiLastLog: %v, matchIdx: %v,count: %v", commitLastLog, rf.matchIndex[server], count)
			if count > len(rf.peers)/2 {
				DPrintf("L %v Commit Log(T:%v, Idx:%v)", rf.me, rf.log[rf.GlobalToLocal(commitLastLog)].Term, commitLastLog)
				// rf.commitIndex = commitLastLog
				break
			}
			commitLastLog--
		}
		rf.commitIndex = commitLastLog
		rf.condApply.Signal()
		return
	}

	if reply.Term > rf.currentTerm {
		// DPrintf("Old Leader %v(T: %v,LastLogI: %v,LastLogT: %v) Received Reply(T: %v), Convert to Follower", rf.me, rf.currentTerm, rf.LocalToGlobal(len(rf.log)-1), rf.log[len(rf.log)-1].Term, reply.Term)
		rf.currentTerm = reply.Term
		rf.votedFor = -1
		rf.persist()
		rf.role = Follower
		rf.heartbeatTimeStamp = time.Now()
		// rf.ResetHeartTimer(HeartbeatTimeThreshold())
	} else if reply.Term == rf.currentTerm && rf.role == Leader {
		DPrintf("%v %v(LastIncludIdx %v) Conflict %v Reply(T %v,XT %v,XL %v,XI %v)", roleName(rf.role), rf.me, rf.lastIncludedIndex, server, reply.Term, reply.XTerm, reply.XLen, reply.XIndex)
		if reply.XTerm == -1 {
			if rf.lastIncludedTerm >= reply.XLen {
				// 缺失的log在Leader的Snapshot中
				rf.nextIndex[server] = rf.lastIncludedIndex
			} else {
				rf.nextIndex[server] = reply.XLen
			}
			DPrintf("%v %v(LastIncludIdx: %v) Reset %v(NextIdx >>> %v, Xlen: %v), ", roleName(rf.role), rf.me, rf.lastIncludedIndex, server, rf.nextIndex[server], reply.XLen)
			return
		}

		i := rf.nextIndex[server] - 1
		if i < rf.lastIncludedIndex {
			i = rf.lastIncludedIndex
		}
		for i > rf.lastIncludedIndex && rf.log[rf.GlobalToLocal(i)].Term > reply.Term {
			i--
		}
		if i == rf.lastIncludedIndex && rf.log[rf.GlobalToLocal(i)].Term > reply.Term {
			rf.nextIndex[server] = rf.lastIncludedIndex
		} else if rf.log[rf.GlobalToLocal(i)].Term == reply.Term {
			rf.nextIndex[server] = i + 1
		} else {
			if reply.XIndex <= rf.lastIncludedIndex {
				rf.nextIndex[server] = rf.lastIncludedIndex
			} else {
				rf.nextIndex[server] = reply.XIndex
			}
		}
		// rf.ResetHeartTimer(ImmidiateTime)
	}
}

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	rf.mu.Lock()
	DPrintf("Snapshot %v %v(LastIncludedIdx: %v) <<< L %v(LastIncludedIdx: %v)", roleName(rf.role), rf.me, rf.lastIncludedIndex, args.LeaderId, args.LastIncludedIndex)
	defer func() {
		rf.heartbeatTimeStamp = time.Now()
		// rf.ResetHeartTimer(HeartbeatTimeThreshold())
		// DPrintf("Snapshot %v %v(LastLogI: %v) <<< L %v(LastIncludedIdx: %v) End Reset HeartbeatTime", roleName(rf.role), rf.me, rf.LocalToGlobal(len(rf.log)-1), args.LeaderId, args.LastIncludedIndex)
		rf.mu.Unlock()
	}()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		return
	}
	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.votedFor = -1
	}
	rf.role = Follower

	if args.LastIncludedIndex < rf.lastIncludedIndex || args.LastIncludedIndex < rf.commitIndex {
		// 收到快照 比当前快照旧，不需要快照 || 快照比当前 commitIndex 旧
		reply.Term = rf.currentTerm
		return
	}

	hasLogInSnapshot := false
	rIdx := 0
	for ; rIdx < len(rf.log); rIdx++ {
		if rf.LocalToGlobal(rIdx) == args.LastIncludedIndex && rf.log[rIdx].Term == args.LastIncludedTerm {
			hasLogInSnapshot = true
			break
		}
	}
	msg := &ApplyMsg{
		SnapshotValid: true,
		Snapshot:      args.Data,
		SnapshotIndex: args.LastIncludedIndex,
		SnapshotTerm:  args.LastIncludedTerm,
	}
	if hasLogInSnapshot {
		// DPrintf("%v %v has Log in Snapshot Idx: %v", roleName(rf.role), rf.me, rIdx)
		rf.log = rf.log[rIdx:]
	} else {
		// DPrintf("%v %v don't has Log in Snapshot, Clear Log", roleName(rf.role), rf.me)
		rf.log = make([]Entry, 0)
		rf.log = append(rf.log, Entry{Term: args.LastIncludedTerm, Command: args.LastIncludedCommand})
	}

	rf.snapShot = args.Data
	rf.lastIncludedIndex = args.LastIncludedIndex
	rf.lastIncludedTerm = args.LastIncludedTerm
	if rf.commitIndex < args.LastIncludedIndex {
		rf.commitIndex = args.LastIncludedIndex
	}

	rf.lastApplied = max(rf.lastApplied, args.LastIncludedIndex)
	reply.Term = rf.currentTerm
	rf.mu.Unlock()
	rf.applyCh <- *msg
	rf.mu.Lock()
	rf.persist()

}

func (rf *Raft) sendInstallSnapshot(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) bool {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	return ok
}

func (rf *Raft) sendInstallSnapshotToServer(server int) {
	// DPrintf("%v %v Preparing SendInstallSnapshotTo %v", roleName(rf.role), rf.me, server)
	rf.mu.Lock()
	if rf.role != Leader {
		rf.mu.Unlock()
		return
	}
	reply := InstallSnapshotReply{}
	args := InstallSnapshotArgs{
		Term:                rf.currentTerm,
		LeaderId:            rf.me,
		LastIncludedIndex:   rf.lastIncludedIndex,
		LastIncludedTerm:    rf.lastIncludedTerm,
		Data:                rf.snapShot,
		LastIncludedCommand: rf.log[0].Command,
		// Done: ,
	}
	DPrintf("%v %v SendInstallSnapshotTo %v", roleName(rf.role), rf.me, server)
	rf.mu.Unlock()
	ok := rf.sendInstallSnapshot(server, &args, &reply)

	if !ok {
		return // 发送失败
	}

	rf.mu.Lock()
	defer func() {
		rf.mu.Unlock()
	}()

	if reply.Term > rf.currentTerm { // 旧 Leader，降级
		rf.currentTerm = reply.Term
		rf.role = Follower
		rf.votedFor = -1
		rf.heartbeatTimeStamp = time.Now()
		// rf.ResetHeartTimer(HeartbeatTimeThreshold())
		rf.persist()
		return
	}
	DPrintf("%v %v Reset Server %v NextIdx %v >>> %v", roleName(rf.role), rf.me, server, rf.nextIndex[server], max(rf.LocalToGlobal(1), rf.nextIndex[server]))
	rf.nextIndex[server] = max(rf.LocalToGlobal(1), rf.nextIndex[server])
}

func (rf *Raft) StartSendAppendEntries() {
	// DPrintf("server_%v tries to send heartbeat\n", rf.me)
	for !rf.killed() {
		// DPrintf("L %v StartSendAppendEntries try get Lock\n", rf.me)
		<-rf.HeartbeatTimer.C
		rf.mu.Lock()
		if rf.role != Leader {
			rf.mu.Unlock()
			return
		}
		// DPrintf("L %v StartSendAppendEntries get the Lock\n", rf.me)

		for i := 0; i < len(rf.peers); i++ {
			if i == rf.me {
				continue
			}
			// DPrintf("Prepare Send AppendEntries, nextIdx: %v, lastIncludedIndex: %v, global :%v", rf.nextIndex[i], rf.lastIncludedIndex, rf.GlobalToLocal(rf.nextIndex[i]-1))
			args := &AppendEntriesArgs{
				Term:         rf.currentTerm,
				LeaderId:     rf.me,
				PrevLogIndex: rf.nextIndex[i] - 1,
				// PrevLogTerm:  rf.log[rf.GlobalToLocal(rf.nextIndex[i]-1)].Term,
				LeaderCommit: rf.commitIndex,
			}
			// 判断发送 InstallSnapshot 还是 Heartbeat/AppendEntries
			installSnapshot := false

			if args.PrevLogIndex < rf.lastIncludedIndex {
				DPrintf("InstallSnapshot L %v(LastIncludedIdx: %v) >>> %v(PrevLogIdx: %v,NextIdx: %v) ", rf.me, rf.lastIncludedIndex, i, args.PrevLogIndex, rf.nextIndex[i])
				installSnapshot = true
			} else if rf.LocalToGlobal(len(rf.log)-1) > args.PrevLogIndex {
				// 测试是否应该在复制 log 时检查leader任期和最后一个日志的任期是否相同，
				args.Entries = append([]Entry{}, rf.log[rf.GlobalToLocal(rf.nextIndex[i]):]...)
				// args.Entries = rf.log[rf.GlobalToLocal(rf.nextIndex[i]):]
				DPrintf("AppendEntries: L %v(T: %v,I: %v) >>> F %v\n", rf.me, rf.currentTerm, rf.nextIndex[i], i)
			}

			if installSnapshot {
				go rf.sendInstallSnapshotToServer(i)
			} else {
				args.PrevLogTerm = rf.log[rf.GlobalToLocal(rf.nextIndex[i]-1)].Term
				go rf.sendAppendEntriesToServer(i, args)
			}

		}
		rf.mu.Unlock()
		// DPrintf("%v %v UnLock in StartSendAppendEntries()", roleName(rf.role), rf.me)
		rf.ResetHeartTimer(HeartbeatTime)
	}
	if rf.killed() {
		DPrintf("%v %v Killed", roleName(rf.role), rf.me)
	} else {
		DPrintf("L %v Becomes %v", rf.me, roleName(rf.role))
	}

}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	// DPrintf("Get %v(Leader: %v) in Start before Lock", rf.me, rf.role == Leader)
	rf.mu.Lock()

	defer func() {
		rf.ResetHeartTimer(ImmidiateTime)
		rf.mu.Unlock()
	}()

	// Your code here (3B).
	if rf.role != Leader {
		// DPrintf("Get %v(Leader: false) in Start after Lock", rf.me)
		return -1, -1, false
	}

	newEntry := Entry{
		Term:    rf.currentTerm,
		Command: command,
	}

	rf.log = append(rf.log, newEntry)

	rf.persist()
	DPrintf("Start %v Cmd %v, LastIdx %v Term: %v", rf.me, command, rf.LocalToGlobal(len(rf.log)-1), rf.currentTerm)
	return rf.LocalToGlobal(len(rf.log) - 1), rf.currentTerm, true
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) procVoteAnswer(server int, args *RequestVoteArgs) bool {
	// sendArgs := &args
	reply := RequestVoteReply{}
	ok := rf.sendRequestVote(server, args, &reply)
	if !ok { // 调用失败，直接返回投票失败
		// DPrintf("cannot connect to server_%v", server)
		return false
	}

	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.role != Candidate || args.Term != rf.currentTerm {
		// 发送投票请求时，有其他Leader产生了，并通过心跳改变了自己的任期，需要放弃投票
		// DPrintf("%v 发送投票Term: %v，Term(T: %v)改变，return false", rf.me, args.Term, rf.currentTerm)
		return false
	}

	if reply.Term > rf.currentTerm {
		// Follower 任期大于 Candidate，需要更新自己记录的当前任期、清除投票、改变角色
		rf.currentTerm = reply.Term
		rf.votedFor = -1
		rf.persist()
		rf.role = Follower
	}

	return reply.VoteGranted
}

func (rf *Raft) collectVote(server int, args *RequestVoteArgs) {
	ok := rf.procVoteAnswer(server, args)
	if !ok {
		// DPrintf("%v Votes Return False", rf.me)
		return
	}
	rf.mu.Lock()
	if rf.voteCount > len(rf.peers)/2 {
		// 如果投票数已经超过了半数，直接返回，因为之前的协程已经处理过了
		// DPrintf("%v skip Check vote", rf.me)
		rf.mu.Unlock()
		return
	}
	rf.voteCount += 1
	// DPrintf("%v(T: %v) Try Check vote", rf.me, rf.currentTerm)
	startElectionTime := time.Since(rf.electionTimeStamp)
	outTime := !(startElectionTime <= time.Duration(rf.electionTimeout)*time.Millisecond)
	if rf.voteCount > len(rf.peers)/2 && rf.role == Candidate && !outTime {
		// 第一次超过半票，并且需要检查自己的身份还是否为Candidate，因为期间可能有其他Leader产生
		// 需要成为leader，并发送心跳
		rf.role = Leader
		// 需要设置 nextIndex 等
		for i := 0; i < len(rf.nextIndex); i++ {
			// rf.nextIndex[i] = len(rf.log)
			// rf.matchIndex[i] = 0
			rf.nextIndex[i] = rf.LocalToGlobal(len(rf.log))
			rf.matchIndex[i] = rf.lastIncludedIndex
		}
		DPrintf("C %v(T: %v,LastLogI: %v,LastLogT: %v,LastLog: %v,CommitI: %v) becomes new Leader", rf.me, rf.currentTerm, len(rf.log)-1, rf.log[len(rf.log)-1].Term, rf.log[len(rf.log)-1].Command, rf.commitIndex)
		// rf.mu.Unlock()
		// 发送心跳消息或复制消息
		rf.ResetHeartTimer(ImmidiateTime)
		go rf.StartSendAppendEntries()
		rf.mu.Unlock()
	} else {
		rf.mu.Unlock()
	}
}

func (rf *Raft) StartElection() {

	rf.mu.Lock()
	defer rf.mu.Unlock()
	rf.currentTerm += 1 // 自增Term
	rf.role = Candidate
	rf.votedFor = rf.me
	rf.persist()
	// DPrintf("%v %v(T: %v) start RequestVote\n", roleName(rf.role), rf.me, rf.currentTerm)
	rf.voteCount = 1
	rf.electionTimeout = RandomElectionTimeout()
	rf.electionTimeStamp = time.Now()  // 更新自己的选举时间戳
	rf.heartbeatTimeStamp = time.Now() // 以免当前选举还未结束，自己又开启一轮选举
	// rf.ResetHeartTimer(HeartbeatTimeThreshold()) // 以免当前选举还未结束，自己又开启一轮选举

	args := &RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: rf.LocalToGlobal(len(rf.log) - 1),
		LastLogTerm:  rf.log[len(rf.log)-1].Term,
	}
	// rf.mu.Unlock()

	for i := 0; i < len(rf.peers); i++ {
		if i == rf.me {
			continue
		}
		go rf.collectVote(i, args)
	}
}

func (rf *Raft) ticker() {
	for !rf.killed() {

		// Your code here (3A)
		// Check if a leader election should be started.
		// DPrintf("%v server_%v checking itself\n", roleName(rf.role), rf.me)
		// <-rf.HeartbeatTimer.C
		rf.mu.Lock()
		sincePrevHeartbeat := time.Since(rf.heartbeatTimeStamp)
		if rf.role != Leader && sincePrevHeartbeat > time.Duration(HeartbeatTimeThreshold())*time.Millisecond {
			go rf.StartElection()
		}

		rf.mu.Unlock()
		// pause for a random amount of time between 50 and 350
		// milliseconds.
		ms := 50 + (rand.Int63() % 300)
		time.Sleep(time.Duration(ms) * time.Millisecond)
	}
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{
		peers:     peers,
		me:        me,
		persister: persister,
		applyCh:   applyCh,
		dead:      0,

		log:         make([]Entry, 0),
		currentTerm: 0,
		votedFor:    -1,
		nextIndex:   make([]int, len(peers)),
		matchIndex:  make([]int, len(peers)),
		commitIndex: 0,
		lastApplied: 0,
		role:        Follower,
		// heartbeatTimeStamp: time.Now(),
		HeartbeatTimer:    time.NewTimer(0),
		electionTimeStamp: time.Now(),
		voteCount:         0,
	}
	rf.condApply = sync.NewCond(&rf.mu)
	rf.log = append(rf.log, Entry{Term: 0})

	// Your initialization code here (3A, 3B, 3C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.readSnapshot(persister.ReadSnapshot())

	// 重置nextIndex
	for i := 0; i < len(rf.nextIndex); i++ {
		rf.nextIndex[i] = rf.LocalToGlobal(len(rf.log))
	}

	// start ticker goroutine to start elections
	go rf.ticker()
	go rf.CommitCheck()

	return rf
}
