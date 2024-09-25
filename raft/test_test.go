package raft

import (
	"fmt"
	"raft-kv-service/persister"
	rrpc "raft-kv-service/rpc"
	"strconv"
	"sync"
	"testing"
	"time"
)

func setupTestCase() ([]*Raftserver, []chan ApplyMsg) {
	nodeAddrs := map[int32]string{0: "localhost:50050", 1: "localhost:50051", 2: "localhost:50052", 3: "localhost:50053", 4: "localhost:50054"}
	// ips := []string{"localhost", "localhost", "localhost", "localhost", "localhost"}
	servers := make([]*Raftserver, 0, len(nodeAddrs))
	channels := make([]chan ApplyMsg, 0, len(nodeAddrs))

	for i := 0; i < len(nodeAddrs); i++ {

		persister := persister.MakePersister(i)
		applyCh := make(chan ApplyMsg, 1)
		channels = append(channels, applyCh)

		ch := make(chan struct{}, 1)
		server := Make(nodeAddrs, int32(i), persister, applyCh, nodeAddrs[int32(i)], ch)

		servers = append(servers, server)
	}

	return servers, channels

}

func TestConnect(t *testing.T) {
	fmt.Println("write setup code here...") // 测试之前的做一些设置
	// 如果 TestMain 使用了 flags，这里应该加上flag.Parse()
	servers, channels := setupTestCase()
	fmt.Printf("装配完毕\n")
	time.Sleep(4 * time.Second)
	var mu sync.Mutex
	var wg sync.WaitGroup
	done := false
	wg.Add(len(servers))
	for i := 0; i < len(servers); i++ {
		go func(idx int, done *bool, mu *sync.Mutex) {
			for {
				op := <-channels[idx]
				fmt.Printf("Apply K:%v, V:%v\n", op.Command.Key, op.Command.Value)
				if op.Command.Key > "998" || *done {
					mu.Lock()
					*done = true
					mu.Unlock()
					wg.Done()
					break
				}
			}
		}(i, &done, &mu)
	}
	fmt.Printf("监听信道\n")

	startTime := time.Now()
	leader := 0
	for i := 0; i < 1000; {
		for ; leader < len(servers); leader++ {
			if _, is := servers[leader].GetState(); is {
				fmt.Printf("t:%v try Start\n", leader)
				_, _, isLeader := servers[leader].Start(&rrpc.Op{Key: strconv.Itoa(i), Value: strconv.Itoa(i + 10000)})
				if isLeader {
					i++
					break
				} else {
					leader = 0
				}
			} else {
				fmt.Printf("%v is not Leader\n", leader)
			}
			time.Sleep(50 * time.Microsecond)
		}

		// time.Sleep(1 * time.Millisecond)
	}
	wg.Wait()

	fmt.Printf("Spend Time: %v\n", time.Since(startTime))
}
