package persister

//
// support for Raft and kvraft to save persistent
// Raft state (log &c) and k/v server snapshots.
//
// we will use the original persister.go to test your code for grading.
// so, while you can modify this code to help you debug, please
// test with the original before submitting.
//

import (
	"hash/crc32"
	"log"
	"os"
	"path"
	rrpc "raft-kv-service/rpc"
	"raft-kv-service/wal"
	"strconv"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"
)

const DBNAME string = "my.db"
const SNAPNAME string = "snap"
const STATNAME string = "./stat"
const STATSUFFIX string = ".state"

const PersistTime = 10 * time.Second

type Persister struct {
	mu sync.Mutex
	// snapName string
	statName string
	id       int
	state    *IndexState

	snapshotter Snapshotter
	snapshot    *Snapshot
	log         *wal.Log
	entryCh     chan *logEntry
	// firstIndex      int64
	// lastIndex       int64
	// lastCommitIndex int64
	checkSum []uint32
	// checkSum []crcentry
	SnapTimer *time.Timer
}

func MakePersister(id int) *Persister {
	log, err := wal.Open("./log/"+strconv.Itoa(id), wal.DefaultOptions)
	if err != nil {
		panic("wal open failed")
	}
	state := &State{VotedFor: -1, CurrentTerm: 0}
	persister := &Persister{
		// snapName:    SNAPNAME,
		id:          id,
		statName:    path.Join(STATNAME, strconv.Itoa(id)+STATSUFFIX),
		snapshotter: *NewSnapshotter(nil, "./snap"),
		state:       &IndexState{RaftSate: state, LastIndex: -1},
		log:         log,
		// entryCh:     make(chan *logEntry, 100),
		SnapTimer: time.NewTimer(PersistTime),
	}
	// persister.initFromDisk()
	persister.initFromDisk()
	go persister.saveLogToDisk()
	go persister.saveSnapClock()
	return persister
}

func (ps *Persister) ResetSnapTimer() {
	ps.SnapTimer.Reset(PersistTime)
}

func (ps *Persister) initFromDisk() {
	var err error
	ps.snapshot, err = ps.snapshotter.Load()
	if err != nil {
		// log.Fatal("init err, failed to load snapshot: ", err)
		log.Print("init err: ", err)
	}
	if ps.snapshot == nil {
		ps.snapshot = &Snapshot{RowData: make([]byte, 0, 1024)}
	}
	// for
	ps.loadState()
	// ps.checkSum = make([]crcentry, ps.state.LastIndex-ps.state.FirstIndex+1)
	ps.checkSum = make([]uint32, max(16, ps.state.LastIndex-ps.state.FirstIndex+1))
	ps.entryCh = make(chan *logEntry, max(100, ps.state.LastIndex-ps.state.FirstIndex+1))
	if ps.state.LastIndex == -1 {
		b, err := proto.Marshal(&rrpc.Entry{Term: 0})
		if err != nil {
			log.Fatal("failed to marshal empty log: ", err)
		}
		ps.checkSum[0] = crc32.ChecksumIEEE(b)
		ps.state.LastIndex = 0
		// ps.checkSum[0] = crcentry{Crc: crc32.ChecksumIEEE(b), Index: 0}
	} else {
		for i := ps.state.FirstIndex; i <= ps.state.LastIndex; i++ {
			b, err := ps.log.Read(uint64(i))
			if err != nil {
				log.Fatal("failed to read log: ", err)
			}
			ps.checkSum[i-ps.state.FirstIndex] = crc32.ChecksumIEEE(b)
			// ps.checkSum[i-ps.state.FirstIndex] = crcentry{Crc: crc32.ChecksumIEEE(b), Index: i}
		}
	}
}

func (ps *Persister) saveState() {
	b, err := proto.Marshal(ps.state)
	if err != nil {
		log.Fatal("save state error: ", err)
	}
	err = WriteAndSyncFile(ps.statName, b, 0666)
	if err != nil {
		log.Fatal("failed to write file, err: ", err)
	}
}

func (ps *Persister) loadState() {
	b, err := os.ReadFile(ps.statName)
	if err != nil {
		// log.Fatal("failed to load state, err: ", err)
		log.Print("failed to load state, err: ", err)
	}
	if !Exist(ps.statName) {
		_, err := os.Create(ps.statName)
		if err != nil {
			log.Print("failed to create file: ", err)
		}
	}
	if len(b) > 0 {
		err = proto.Unmarshal(b, ps.state)
		if err != nil {
			log.Print("failed to unmarshal state, err: ", err)
		} else {
			log.Printf("%v load state first: %v, last: %v", ps.id, ps.state.FirstIndex, ps.state.LastIndex)
		}
	}

}

func clone(orig []byte) []byte {
	x := make([]byte, len(orig))
	copy(x, orig)
	return x
}

func (ps *Persister) Copy() *Persister {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	np := MakePersister(ps.id)
	np.snapshot = ps.snapshot
	return np
}

func (ps *Persister) ReadRaftState() ([]*rrpc.Entry, int32, int64, int64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	start := ps.state.FirstIndex
	end := ps.state.LastIndex
	entries := make([]*rrpc.Entry, 0, end-start+1)
	// if ps.state.LastIndex == 0 {
	entries = append(entries, &rrpc.Entry{Term: max(ps.state.FirstIndex-1, 0)})
	// }
	for ; end > 0 && start <= end; start++ {
		b, err := ps.log.Read(uint64(start))
		if err != nil {
			log.Fatal("failed to read log, err: ", err)
		}
		entry := &rrpc.Entry{}
		err = proto.Unmarshal(b, entry)
		if err != nil {
			log.Fatal("failed to unmarshal log, err: ", err)
		}
		entries = append(entries, entry)
	}

	if entries == nil {
		log.Print("entries is nil")
	}
	if ps.state == nil {
		log.Print("ps.state is nil")
	}
	if ps.state.RaftSate == nil {
		log.Print("ps.state.RaftSate is nil")
	}

	// b, err := ps.log.Read(ps.log.GetFirstIndex())
	// if err != nil {
	// 	log.Fatal("failed to read log, err: ", err)
	// }
	// firstEntry := &rrpc.Entry{}
	// err = proto.Unmarshal(b, firstEntry)
	// if err != nil {
	// 	log.Fatal("failed to unmarshal log, err: ", err)
	// }
	// log.Printf("Log first saved entry: %v", firstEntry)

	log.Printf("%v persister saved state FirstIdx: %v,LastIdx: %v, commitIndex: %v", ps.id, ps.state.FirstIndex, ps.state.LastIndex, ps.state.LastCommitIndex)
	log.Printf("%v val saved state FirstIdx: %v,LastIdx: %v", ps.id, ps.log.GetFirstIndex(), ps.log.GetLastIndex())
	// log.Printf("Load logs: %v", entries)
	// log.Fatal("Stop")
	return entries, ps.state.RaftSate.VotedFor, ps.state.RaftSate.CurrentTerm, ps.state.LastCommitIndex
}

func (ps *Persister) RaftStateSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return int(ps.state.LastIndex - ps.state.FirstIndex + 1)
}

// Save both Raft state and K/V snapshot as a single atomic action,
// to help avoid them getting out of sync.
func (ps *Persister) Save(snapshot []byte, state *State, logs []*rrpc.Entry, lastIncludeIndex int64, lastTerm int64, commitIndex int64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.state.RaftSate = state
	ps.state.LastCommitIndex = commitIndex
	ps.saveState()
	// ps.snapshot = &Snapshot{RowData: clone(snapshot), LastIndex: lastIncludeIndex, LastTerm: lastTerm}
	ps.snapshot.RowData = clone(snapshot)
	ps.snapshot.LastIndex = lastIncludeIndex
	ps.snapshot.LastTerm = lastTerm
	// TODO: 找到冲突开始的日志

	// log.Printf("将存储logs: %v", logs)

	// localLogs := make([]*rrpc.Entry, 0)
	// for i:= 0; i< len(logs); i++ {
	// 	localLogs[i] = &rrpc.Entry{

	// 	}
	// }
	// go ps.saveLog(lastIncludeIndex, commitIndex, logs)
	ps.filter(lastIncludeIndex, logs)
}

func (ps *Persister) filter(lastIndex int64, logs []*rrpc.Entry) {
	log.Printf("persister lastIdx: %v, log lastIdx: %v", ps.state.LastIndex, lastIndex)
	if lastIndex >= ps.state.LastIndex {
		for i, entry := range logs {
			b, err := proto.Marshal(entry)
			if err != nil {
				log.Fatal("Marshal log error: ", err)
			}
			log.Printf("%v (1) add log channel idx: %v", ps.id, lastIndex+int64(i+1))
			ps.entryCh <- &logEntry{Index: lastIndex + int64(i+1), Entry: b}
		}
		// if lastIndex > ps.state.LastIndex {
		// 	// TODO: 将快照持久化，并截断日志
		// }
		return
	}
	i := max(0, ps.state.LastCommitIndex-lastIndex)
	for ; i < int64(len(logs)); i++ {
		b, err := proto.Marshal(logs[i])
		if err != nil {
			log.Fatal("Marshal log error: ", err)
		}
		if i+lastIndex+1 > ps.state.LastIndex || crc32.ChecksumIEEE(b) != ps.checkSum[i+lastIndex+1-ps.state.FirstIndex] {
			log.Printf("%v (2) add log channel idx: %v", ps.id, i+lastIndex+1)
			ps.entryCh <- &logEntry{Index: i + lastIndex + 1, Entry: b}
		}
	}
}

func (ps *Persister) saveLogToDisk() {
	for {
		_log := <-ps.entryCh
		ps.mu.Lock()

		if _log.Index > ps.state.LastIndex+1 {
			// TODO: 将快照持久化，并截断日志
			ps.saveSnapToDisk()
		} else if _log.Index < ps.state.LastIndex {
			err := ps.log.TruncateBack(uint64(_log.Index - 1))
			if err != nil {
				log.Fatalf("%v truncateback failed idx %v err: %v", ps.id, _log.Index-1, err)
			}
			ps.checkSum = ps.checkSum[:_log.Index-ps.state.FirstIndex]
		}

		ps.checkSum = append(ps.checkSum, crc32.ChecksumIEEE(_log.Entry))
		err := ps.log.Write(uint64(_log.Index), _log.Entry)
		if err != nil {
			log.Printf("%v write log to disk failed: %v", ps.id, err)
		}
		ps.state.LastIndex = int64(ps.log.GetLastIndex())
		ps.saveState()
		ps.mu.Unlock()
	}
}

func (ps *Persister) saveSnapToDisk() {
	if ps.snapshot.LastIndex != ps.snapshotter.LastIdx {
		err := ps.snapshotter.SaveSnap(ps.snapshot)
		if err != nil {
			log.Fatal("snapshot save err: ", err)
		} else {
			log.Print("snapshot save to disk successfully")
			ps.snapshotter.LastIdx = ps.snapshot.LastIndex
		}
		lastWalIndex := ps.log.GetLastIndex()

		// 此处暂时会多留一个 log，但不影响
		err = ps.log.TruncateFront(min(uint64(ps.snapshot.LastIndex), lastWalIndex))
		// err = ps.log.TruncateFront(uint64(ps.snapshot.LastIndex + 1))
		if err != nil {
			log.Printf("log firstIdx: %v,lastIdx: %v, need truncate: %v", ps.log.GetFirstIndex(), lastWalIndex, min(uint64(ps.snapshot.LastIndex), lastWalIndex))
			log.Fatal("truncate front log err: ", err)
		}
		log.Printf("log in disk firstIdx: %v, lastIdx: %v", ps.log.GetFirstIndex(), ps.log.GetFirstIndex())
		ps.checkSum = ps.checkSum[ps.snapshot.LastIndex-ps.state.FirstIndex+1:]
		ps.state.FirstIndex = ps.snapshot.LastIndex + 1
		ps.saveState()
	} else {
		log.Printf("will not save snapshot lastIdx: %v", ps.snapshot.LastIndex)
	}
	ps.ResetSnapTimer()
}

func (ps *Persister) saveSnapClock() {
	for {
		<-ps.SnapTimer.C
		ps.mu.Lock()
		ps.saveSnapToDisk()
		ps.mu.Unlock()
	}

}

func (ps *Persister) ReadSnapshot() ([]byte, int64, int64) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	// return clone(ps.snapshot)
	snapshot := make([]byte, len(ps.snapshot.RowData))
	copy(snapshot, ps.snapshot.RowData)
	return snapshot, ps.snapshot.LastIndex, ps.snapshot.LastTerm
}

func (ps *Persister) SnapshotSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.snapshot.RowData)
}
