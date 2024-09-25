package server

import (
	"fmt"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"raft-kv-service/conf"
	"raft-kv-service/mylog"
	"raft-kv-service/persister"
	"raft-kv-service/proxy"
	rrpc "raft-kv-service/rpc"
	"raft-kv-service/wal"
	"runtime/pprof"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
)

type FailedCase struct {
	ClientId int32
	LoopIdx  int
}

var logs_files = []string{"./log", "./stat"}

func deleteFilesInDir(dir string) error {
	// 遍历目录下的所有文件
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 如果是文件则删除
		if !info.IsDir() {
			if err := os.Remove(path); err != nil {
				return err
			}
		}
		return nil
	})
}

func deleteLogsFile() {
	for _, log_file := range logs_files {
		deleteFilesInDir(log_file)
	}
}

func setupTestCase(t *testing.T) ([]*KVServer, []*Clerk) {

	globalObj := conf.GetGlobalObj()
	serverNum := len(globalObj.ServerAddrs)

	nodeAddrs := make(map[int32]string)
	for i := 0; i < serverNum; i++ {
		nodeAddrs[int32(i)] = globalObj.ServerAddrs[i]["raft"]
	}

	servers := make([]*KVServer, 0, serverNum)
	// 启动服务端
	for i := 0; i < serverNum; i++ {

		persister := persister.MakePersister(i)

		server := StartKVServer(nodeAddrs, int32(i), persister, globalObj.ServerAddrs[i], 1000)

		servers = append(servers, server)
	}

	// time.Sleep(3 * time.Second)
	for i := 0; i < serverNum; i++ {
		<-servers[i].serverprepare
	}

	// TODO: 启动代理
	err := StartOpenResty()
	if err != nil {
		mylog.DPrintln("start nginx error: ", err)
		t.Fatal("failed to start nginx")
	}
	mylog.DPrintln("start nginx succ")

	// 启动客户端
	clerks := make([]*Clerk, 0, globalObj.ClientNum)
	for i := 0; i < globalObj.ClientNum; i++ {
		clerks = append(clerks, MakeClerk(globalObj.ProxyAddrs))
	}
	mylog.DPrintln("start clients succ")

	return servers, clerks

}

func TestStartPersister(t *testing.T) {
	_ = persister.MakePersister(1)
	mylog.DPrintln("starrt persister")
}

func TestStartClusterManager(t *testing.T) {
	globalObj := conf.GetGlobalObj()

	clusterMg := NewCluster(globalObj.ProxyAddrs[0], globalObj.ProxyAddrs[1])

	clusterMg.Start()

	time.Sleep(5 * time.Second)

	clusterMg.Stop()
}

func TestStartClusterManagerServers(t *testing.T) {
	globalObj := conf.GetGlobalObj()

	clusterMg := NewCluster(globalObj.ClusterConf.ServAddr, globalObj.ClusterConf.SyncAddr)

	clusterMg.Start()

	serverNum := len(globalObj.ServerAddrs)
	nodeAddrs := make(map[int32]string)
	for i := 0; i < serverNum; i++ {
		nodeAddrs[int32(i)] = globalObj.ServerAddrs[i]["raft"]
	}
	servers := make([]*KVServer, 0, serverNum)
	for i := 0; i < serverNum; i++ {
		persister := persister.MakePersister(i)

		server := StartKVServer(nodeAddrs, int32(i), persister, globalObj.ServerAddrs[i], 1000)

		servers = append(servers, server)
	}
	for i := 0; i < serverNum; i++ {
		<-servers[i].serverprepare
	}

	time.Sleep(5 * time.Second)

	clusterMg.Stop()
}

// 暂时不用了
func TestStartNginx(t *testing.T) {
	err := StartOpenResty()
	if err != nil {
		mylog.DPrintln("start nginx error: ", err)
		t.Fail()
	}
	defer func() {
		err := StopOpenResty()
		mylog.DPrintln("stop nginx error: ", err)
	}()
}

func TestStartClient(t *testing.T) {

	globalObj := conf.GetGlobalObj()
	clients := make([]*Clerk, 0, 3)
	for i := 0; i < 3; i++ {
		clients = append(clients, MakeClerk(globalObj.ProxyAddrs))
	}

	defer func() {
		for _, client := range clients {
			client.Stop()
		}
	}()
}

func TestStartServer(t *testing.T) {
	globalObj := conf.GetGlobalObj()
	serverNum := len(globalObj.ServerAddrs)

	nodeAddrs := make(map[int32]string)
	for i := 0; i < serverNum; i++ {
		nodeAddrs[int32(i)] = globalObj.ServerAddrs[i]["raft"]
	}

	servers := make([]*KVServer, 0, serverNum)
	for i := 0; i < serverNum; i++ {
		persister := persister.MakePersister(i)

		server := StartKVServer(nodeAddrs, int32(i), persister, globalObj.ServerAddrs[i], 1000)

		servers = append(servers, server)
	}

	for i := 0; i < serverNum; i++ {
		<-servers[i].serverprepare
	}

	time.Sleep(5 * time.Second)
}

// 测试Servers启动、ClusterManager启动、Proxy启动、Client启动并请求
func TestUserspaceProxy(t *testing.T) {
	deleteLogsFile()
	globalObj := conf.GetGlobalObj()

	wl, err := net.Listen("tcp", globalObj.ProxyAddrs[PUT])
	if err != nil {
		t.Fatal(err)
	}
	defer wl.Close()
	rl, err := net.Listen("tcp", globalObj.ProxyAddrs[GET])
	if err != nil {
		t.Fatal(err)
	}
	defer rl.Close()

	// make proxy
	p := proxy.TCPProxy{
		WListener: wl,
		RListener: rl,
		Endpoints: []*net.SRV{},
	}

	serverAddr := strings.Split(globalObj.ClusterConf.ServAddr, ":")
	port, err := strconv.Atoi(serverAddr[1])
	if err != nil {
		t.Fatal("解析port失败")
	}
	p.Endpoints = append(p.Endpoints, &net.SRV{
		Target: serverAddr[0],
		Port:   uint16(port),
	})

	// make cluster manager
	clusterMg := NewCluster(globalObj.ClusterConf.ServAddr, globalObj.ClusterConf.SyncAddr)
	clusterMg.Start()
	defer clusterMg.Stop()

	// make servers
	serverNum := len(globalObj.ServerAddrs)
	nodeAddrs := make(map[int32]string)
	for i := 0; i < serverNum; i++ {
		nodeAddrs[int32(i)] = globalObj.ServerAddrs[i]["raft"]
	}
	servers := make([]*KVServer, 0, serverNum)
	for i := 0; i < serverNum; i++ {
		persister := persister.MakePersister(i)

		server := StartKVServer(nodeAddrs, int32(i), persister, globalObj.ServerAddrs[i], 1000)

		servers = append(servers, server)
	}
	for i := 0; i < serverNum; i++ {
		<-servers[i].serverprepare
	}

	go p.Run()
	defer func() {
		for _, server := range servers {
			server.Stop()
		}
		p.Stop()
	}()

	// make client
	client := MakeClerk(globalObj.ProxyAddrs)
	err = client.Put("key1", "value1", 3*time.Second)
	if err != nil {
		t.Fatal("[Client] Put op failed: ", err)
	}
	mylog.DPrintln("[Client] put value succ")
	got, err := client.Get("key1", 3*time.Second)
	if err != nil {
		t.Fatal("[Client] Get op failed: ", err)
	}
	mylog.DPrintln("[Client] get value: ", got)
	defer client.Stop()
}

func TestStart(t *testing.T) {
	deleteLogsFile()
	globalObj := conf.GetGlobalObj()

	wl, err := net.Listen("tcp", globalObj.ProxyAddrs[PUT])
	if err != nil {
		t.Fatal(err)
	}
	defer wl.Close()
	rl, err := net.Listen("tcp", globalObj.ProxyAddrs[GET])
	if err != nil {
		t.Fatal(err)
	}
	defer rl.Close()

	// make proxy
	p := proxy.TCPProxy{
		WListener: wl,
		RListener: rl,
		Endpoints: []*net.SRV{},
	}

	serverAddr := strings.Split(globalObj.ClusterConf.ServAddr, ":")
	port, err := strconv.Atoi(serverAddr[1])
	if err != nil {
		t.Fatal("解析port失败")
	}
	p.Endpoints = append(p.Endpoints, &net.SRV{
		Target: serverAddr[0],
		Port:   uint16(port),
	})

	// make cluster manager
	clusterMg := NewCluster(globalObj.ClusterConf.ServAddr, globalObj.ClusterConf.SyncAddr)
	clusterMg.Start()
	defer clusterMg.Stop()

	// make servers
	serverNum := len(globalObj.ServerAddrs)
	nodeAddrs := make(map[int32]string)
	for i := 0; i < serverNum; i++ {
		nodeAddrs[int32(i)] = globalObj.ServerAddrs[i]["raft"]
	}
	servers := make([]*KVServer, 0, serverNum)
	for i := 0; i < serverNum; i++ {
		persister := persister.MakePersister(i)

		server := StartKVServer(nodeAddrs, int32(i), persister, globalObj.ServerAddrs[i], 1000)

		servers = append(servers, server)
	}
	for i := 0; i < serverNum; i++ {
		<-servers[i].serverprepare
	}

	go p.Run()
	defer func() {
		for _, server := range servers {
			server.Stop()
		}
		p.Stop()
	}()
}

func TestBasic(t *testing.T) {
	f, _ := os.Create("CPU.out")
	defer f.Close()
	pprof.StartCPUProfile(f)
	defer pprof.StopCPUProfile()
	fmt.Println("write setup code here...") // 测试之前的做一些设置
	// 如果 TestMain 使用了 flags，这里应该加上flag.Parse()
	servers, clients := setupTestCase(t)

	time.Sleep(2 * time.Second)
	result := make(map[string]string, 1000)
	n := 10000
	errorCount := 0
	errorKey := make([]string, 0)
	failedCount := 0
	failedLoop := make([]FailedCase, 0, n)

	var mu sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(clients))
	timeout := 3 * time.Second

	for i := 0; i < 1000; i++ {
		key := strconv.Itoa(i)
		randIdx := rand.Intn(len(clients))
		getValue, err := clients[randIdx].Get(key, timeout)
		mu.Lock()
		if err == nil {
			result[key] = getValue
		} else {
			failedCount++
			failedLoop = append(failedLoop, FailedCase{ClientId: clients[randIdx].clerkId, LoopIdx: i})
		}
		mu.Unlock()
	}

	startTime := time.Now()
	for i := 0; i < len(clients); i++ {
		go func(k int) {
			defer wg.Done()
			for j := 0; j < n; j++ {
				randnum := rand.Float64()
				key := strconv.Itoa(j % 1000)
				value := strconv.Itoa(int(rand.Int31() / 100000))
				fmt.Printf("Client %d Testing Loop: %d\n", k, j)
				if randnum < 0.33 {
					err := clients[k].Put(key, value, timeout)
					if err == nil {
						result[key] = value
					} else {
						failedCount++
						failedLoop = append(failedLoop, FailedCase{ClientId: clients[k].clerkId, LoopIdx: j})
					}
				} else if randnum > 0.67 {
					getValue, err := clients[k].Get(key, timeout)
					mu.Lock()
					if err == nil {
						if !strings.EqualFold(getValue, result[key]) {
							fmt.Printf("Get Value Err: K:%v want:(V:%v) but Get(V:%v)\n", key, result[key], getValue)
							errorCount++
							errorKey = append(errorKey, key)
						}
					} else {
						failedCount++
						failedLoop = append(failedLoop, FailedCase{ClientId: clients[k].clerkId, LoopIdx: j})
					}
					mu.Unlock()
				} else {
					err := clients[k].Append(key, value, timeout)
					mu.Lock()
					if err == nil {
						result[key] = result[key] + value
					} else {
						failedCount++
						failedLoop = append(failedLoop, FailedCase{ClientId: clients[k].clerkId, LoopIdx: j})
					}
					mu.Unlock()
				}
			}
		}(i)
	}

	wg.Wait()

	for key, want := range result {
		get, err := clients[0].Get(key, timeout)
		mu.Lock()
		if err == nil {
			if !strings.EqualFold(get, want) {
				fmt.Printf("Get Value Err: K:%v want:(V:%v) but Get(V:%v)\n", key, want, get)
				errorCount++
				errorKey = append(errorKey, key)
			}
		} else {
			failedCount++
		}
		mu.Unlock()
	}

	fmt.Printf("Spend Time: %v\n", time.Since(startTime))
	fmt.Printf("Operation Failed: %v\n", failedCount)
	fmt.Printf("Operation Failed Loop: %v\n", failedLoop)
	if errorCount > 0 {
		fmt.Printf("Error Get: %v Times\n", errorCount)
		fmt.Printf("Error Get Keys: %v\n", errorKey)
		t.Fatalf("Test Failed")
	}
	defer func() {
		for i := 0; i < len(clients); i++ {
			clients[i].Stop()
		}

		for i := 0; i < len(servers); i++ {
			servers[i].Stop()
		}
		StopOpenResty()

	}()
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
