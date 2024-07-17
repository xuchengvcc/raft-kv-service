package server

import (
	"fmt"
	"math/rand"
	"net"
	"raft-kv-service/raft"
	"strconv"
	"strings"
	"testing"
	"time"
)

func setupTestCase(t *testing.T) ([]*KVServer, []string) {
	nodeAddrs := []string{"localhost:50050", "localhost:50051", "localhost:50052", "localhost:50053", "localhost:50054"}
	// ips := []string{"localhost", "localhost", "localhost", "localhost", "localhost"}
	serverAddrs := []string{"localhost:50040", "localhost:50041", "localhost:50042", "localhost:50043", "localhost:50044"}
	servers := make([]*KVServer, 0, len(nodeAddrs))

	for i := 0; i < len(nodeAddrs); i++ {

		persister := raft.MakePersister()
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
		server := StartKVServer(nodeAddrs, int32(i), persister, nodeAddrs[i], lis, slis, 10000)

		servers = append(servers, server)
	}

	time.Sleep(3 * time.Second)

	return servers, serverAddrs

}

func TestBasic(t *testing.T) {

	fmt.Println("write setup code here...") // 测试之前的做一些设置
	// 如果 TestMain 使用了 flags，这里应该加上flag.Parse()
	_, addrs := setupTestCase(t)

	client := MakeClerk(addrs)
	time.Sleep(2 * time.Second)
	result := make(map[string]string, 1000)
	n := 10000
	errorCount := 0
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
		value := strconv.Itoa(rand.Int())
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
			}
		} else {
			failedCount++
		}
	}

	if errorCount > 0 {
		t.Fatalf("Test Failed")
	}
	fmt.Printf("Spend Time: %v\n", time.Since(startTime))
	fmt.Printf("Operation Failed: %v\n", failedCount)
	fmt.Printf("Operation Failed Loop: %v\n", failedLoop)
}
