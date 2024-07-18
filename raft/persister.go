package raft

//
// support for Raft and kvraft to save persistent
// Raft state (log &c) and k/v server snapshots.
//
// we will use the original persister.go to test your code for grading.
// so, while you can modify this code to help you debug, please
// test with the original before submitting.
//

import (
	"log"
	"sync"
	"time"

	"github.com/boltdb/bolt"
)

const PersistTime = 10 * time.Second

type Persister struct {
	mu        sync.Mutex
	dbName    string
	snapName  string
	statName  string
	raftstate []byte
	snapshot  []byte
}

func MakePersister() *Persister {
	persister := &Persister{
		dbName:   DBNAME,
		snapName: SNAPNAME,
		statName: STATNAME,
	}
	persister.initFromDisk()
	return persister
}

func clone(orig []byte) []byte {
	x := make([]byte, len(orig))
	copy(x, orig)
	return x
}

func (ps *Persister) initFromDisk() {
	db, err := bolt.Open(ps.dbName, 0600, nil)
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	err = db.View(func(tx *bolt.Tx) error {
		bsnap := tx.Bucket([]byte(ps.snapName))
		bstat := tx.Bucket([]byte(ps.statName))
		if bsnap == nil {
			DPrintf("snapshot is nil in disk")
		} else if value := bsnap.Get([]byte(ps.snapName)); value != nil {
			DPrintf("init snapshot from disk")
			ps.mu.Lock()
			ps.snapshot = clone(value)
			ps.mu.Unlock()
		}

		if bstat == nil {
			DPrintf("state is nil in disk")
		} else if value := bstat.Get([]byte(ps.statName)); value != nil {
			DPrintf("init state from disk")
			ps.mu.Lock()
			ps.raftstate = clone(value)
			ps.mu.Unlock()
		}
		return nil
	})

	if err != nil {
		log.Fatal(err)
	}
	DPrintf("Init from disk successfully")

}

// func (ps *Persister) SaveSchedule() {
// 	for {
// 		time.Sleep(PersistTime)
// 		ps.saveToDisk()
// 	}
// }

// func (ps *Persister) saveToDisk() {
// 	db, err := bolt.Open(ps.dbName, 0600, nil)
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	defer db.Close()

// 	// 创建一个桶（bucket）并保存数据
// 	kS := []byte("snap")
// 	kL := []byte("stat")

// 	err = db.Update(func(tx *bolt.Tx) error {
// 		// 创建一个桶（bucket）
<<<<<<< HEAD
=======
// 		ps.mu.Lock()
>>>>>>> 240717-read-index
// 		b, err := tx.CreateBucketIfNotExists([]byte("snap"))
// 		if err != nil {
// 			return fmt.Errorf("create bucket err: %s", err)
// 		}
<<<<<<< HEAD
// 		ps.mu.Lock()
// 		value := clone(ps.ReadSnapshot())
// 		ps.mu.Unlock()
=======
// 		value := clone(ps.ReadSnapshot())
>>>>>>> 240717-read-index
// 		// 存储数据
// 		if err := b.Put(kS, value); err != nil {
// 			return fmt.Errorf("put value err: %s", err)
// 		}
<<<<<<< HEAD
=======
// 		ps.mu.Unlock()
>>>>>>> 240717-read-index
// 		return nil
// 	})
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	err = db.Update(func(tx *bolt.Tx) error {
// 		// 创建一个桶（bucket）
// 		b, err := tx.CreateBucketIfNotExists([]byte("stat"))
// 		if err != nil {
// 			return fmt.Errorf("create bucket err: %s", err)
// 		}
// 		ps.mu.Lock()
// 		value := clone(ps.ReadRaftState())
// 		ps.mu.Unlock()
// 		// 存储数据
// 		if err := b.Put(kL, value); err != nil {
// 			return fmt.Errorf("put value err: %s", err)
// 		}
// 		return nil
// 	})
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// }

func (ps *Persister) Copy() *Persister {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	np := MakePersister()
	np.raftstate = ps.raftstate
	np.snapshot = ps.snapshot
	return np
}

func (ps *Persister) ReadRaftState() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return clone(ps.raftstate)
}

func (ps *Persister) RaftStateSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.raftstate)
}

// Save both Raft state and K/V snapshot as a single atomic action,
// to help avoid them getting out of sync.
func (ps *Persister) Save(raftstate []byte, snapshot []byte) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.raftstate = clone(raftstate)
	ps.snapshot = clone(snapshot)
	// go ps.saveToDisk()
}

func (ps *Persister) ReadSnapshot() []byte {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return clone(ps.snapshot)
}

func (ps *Persister) SnapshotSize() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.snapshot)
}
