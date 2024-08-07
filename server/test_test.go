package server

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"raft-kv-service/persister"
	rrpc "raft-kv-service/rpc"
	"raft-kv-service/wal"
	"runtime/pprof"
	"strconv"
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

func setupTestCase(t *testing.T) ([]*KVServer, []string) {
	nodeAddrs := []string{"localhost:50050", "localhost:50051", "localhost:50052", "localhost:50053", "localhost:50054"}
	// ips := []string{"localhost", "localhost", "localhost", "localhost", "localhost"}
	serverAddrs := []string{"localhost:50040", "localhost:50041", "localhost:50042", "localhost:50043", "localhost:50044"}
	servers := make([]*KVServer, 0, len(nodeAddrs))

	for i := 0; i < len(nodeAddrs); i++ {

		persister := persister.MakePersister(i)
		ip_port := strings.Split(nodeAddrs[i], ":")
		lis, err := net.Listen("tcp", fmt.Sprintf(":%v", ip_port[1]))
		if err != nil {
			t.Fatalf("node listen Failed at %v, Err:%v", ip_port[1], err)
		}
		server_port := strings.Split(serverAddrs[i], ":")
		slis, serr := net.Listen("tcp", fmt.Sprintf(":%v", server_port[1]))
		if serr != nil {
			t.Fatalf("server listen Failed at %v, Err:%v", ip_port[1], err)
		}
		server := StartKVServer(nodeAddrs, int32(i), persister, nodeAddrs[i], lis, slis, 1000)

		servers = append(servers, server)
	}

	// time.Sleep(3 * time.Second)
	for i := 0; i < len(nodeAddrs); i++ {
		<-servers[i].serverprepare
	}

	return servers, serverAddrs

}

func TestBasic(t *testing.T) {
	f, _ := os.Create("CPU.out")
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	fmt.Println("write setup code here...") // 测试之前的做一些设置
	// 如果 TestMain 使用了 flags，这里应该加上flag.Parse()
	_, addrs := setupTestCase(t)

	client := MakeClerk(addrs)
	time.Sleep(2 * time.Second)
	result := make(map[string]string, 1000)
	n := 10000
	errorCount := 0
	errorKey := make([]string, 0)
	failedCount := 0
	failedLoop := make([]int, 0)
	for i := 0; i < 1000; i++ {
		key := strconv.Itoa(i)
		getValue, err := client.Get(key)
		if err == nil {
			result[key] = getValue
		}
	}

	startTime := time.Now()
	for i := 0; i < n; i++ {
		randnum := rand.Float64()
		key := strconv.Itoa(i % 1000)
		value := strconv.Itoa(int(rand.Int31() / 100000))
		fmt.Printf("Testing Loop: %v\n", i)
		if randnum < 0.33 {
			err := client.Put(key, value)
			if err == nil {
				result[key] = value
			} else {
				failedCount++
				failedLoop = append(failedLoop, i)
			}

		} else if randnum > 0.67 {
			getValue, err := client.Get(key)
			if err == nil {
				if !strings.EqualFold(getValue, result[key]) {
					fmt.Printf("Get Value Err: K:%v want:(V:%v) but Get(V:%v)\n", key, result[key], getValue)
					errorCount++
					errorKey = append(errorKey, key)
				}
			} else {
				failedCount++
				failedLoop = append(failedLoop, i)
			}

		} else {
			err := client.Append(key, value)
			if err == nil {
				result[key] = result[key] + value
			} else {
				failedCount++
				failedLoop = append(failedLoop, i)
			}
		}
	}

	for key, want := range result {
		get, err := client.Get(key)
		if err == nil {
			if !strings.EqualFold(get, want) {
				fmt.Printf("Get Value Err: K:%v want:(V:%v) but Get(V:%v)\n", key, want, get)
				errorCount++
				errorKey = append(errorKey, key)
			}
		} else {
			failedCount++
		}
	}

	fmt.Printf("Spend Time: %v\n", time.Since(startTime))
	fmt.Printf("Operation Failed: %v\n", failedCount)
	fmt.Printf("Operation Failed Loop: %v\n", failedLoop)
	if errorCount > 0 {
		fmt.Printf("Error Get: %v Times\n", errorCount)
		fmt.Printf("Error Get Keys: %v\n", errorKey)
		t.Fatalf("Test Failed")
	}
}

func TestWal(t *testing.T) {
	logs, err := wal.Open("./log/"+strconv.Itoa(0), wal.DefaultOptions)
	if err != nil {
		panic("wal open failed")
	}
	entries := make([]*rrpc.Op, 0, 100)
	for i := 0; i < 100; i++ {
		entries = append(entries, &rrpc.Op{Key: strconv.Itoa(i), Value: strconv.Itoa(i)})
	}
	for i := range entries {
		fmt.Println("marshal ", i)
		b, err := proto.Marshal(entries[i])
		if err != nil {
			fmt.Print("err: ", err)
		}
		logs.Write(uint64(i+1), b)
	}

	fmt.Printf("firstIdx: %v, lastIdx: %v\n", logs.GetFirstIndex(), logs.GetLastIndex())

	tb := 99
	logs.TruncateBack(uint64(tb))
	fmt.Printf("TruncateBack: %v, now firstIdx: %v, lastIdx: %v\n", tb, logs.GetFirstIndex(), logs.GetLastIndex())

	// tf := 99
	// logs.TruncateFront(uint64(tf))
	// fmt.Printf("TruncateFront: %v, now firstIdx: %v, lastIdx: %v\n", tf, logs.GetFirstIndex(), logs.GetLastIndex())

	b, err := logs.Read(uint64(0))
	if err != nil {
		fmt.Println("Read failed: ", err)
	}
	entry := &rrpc.Op{}
	err = proto.Unmarshal(b, entry)
	if err != nil {
		fmt.Println("unmarshal failed")
	}
	fmt.Println(0, " get ", entry)
}
