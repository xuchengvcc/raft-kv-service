package main

import (
	"fmt"
	"net"
	"raft-kv-service/conf"
	"raft-kv-service/mylog"
	"raft-kv-service/persister"
	"raft-kv-service/proxy"
	"raft-kv-service/server"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCorrect(t *testing.T) {
	globalObj := conf.GetGlobalObj()

	wl, err := net.Listen("tcp", globalObj.ProxyAddrs[server.PUT])
	if err != nil {
		mylog.DPrintln("listening error: ", err)
	}
	defer wl.Close()
	rl, err := net.Listen("tcp", globalObj.ProxyAddrs[server.GET])
	if err != nil {
		mylog.DPrintln("listening error: ", err)
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
		mylog.DPrintln("convert ip port addr error: ", err)
		return
	}
	p.Endpoints = append(p.Endpoints, &net.SRV{
		Target: serverAddr[0],
		Port:   uint16(port),
	})

	// make cluster manager
	clusterMg := server.NewCluster(globalObj.ClusterConf.ServAddr, globalObj.ClusterConf.SyncAddr)
	clusterMg.Start()
	defer clusterMg.Stop()

	// make servers
	serverNum := len(globalObj.ServerAddrs)
	nodeAddrs := make(map[int32]string)
	for i := 0; i < serverNum; i++ {
		nodeAddrs[int32(i)] = globalObj.ServerAddrs[i]["raft"]
	}
	servers := make([]*server.KVServer, 0, serverNum)
	for i := 0; i < serverNum; i++ {
		persister := persister.MakePersister(i)

		server := server.StartKVServer(nodeAddrs, int32(i), persister, globalObj.ServerAddrs[i], 1000)

		servers = append(servers, server)
	}
	for i := 0; i < serverNum; i++ {
		<-servers[i].Serverprepare
	}

	go p.Run()
	defer func() {
		for _, server := range servers {
			server.Stop()
		}
		p.Stop()
	}()

	var isEnd string
	if isEnd != "end" {
		fmt.Scan(&isEnd)
	}
}

func TestClientUnpack(t *testing.T) {
	globalObj := conf.GetGlobalObj()

	client := server.MakeClerk(globalObj.ProxyAddrs)
	defer client.Stop()

	timeout := 3 * time.Second
	var Op int32 = 0
	var Key string
	var Value string
	var Got string
	var err error
	for Op != -1 {
		fmt.Println("please enter Op: [1: GET, 2: PUT, 3: APPEND, -1: Quit]")
		fmt.Scan(&Op)
		if Op == -1 {
			break
		}
		fmt.Println("please enter Key: [any string]")
		fmt.Scan(&Key)
		switch Op {
		case 1:
			Got, err = client.Get(Key, timeout)
		case 2:
			fmt.Println("please enter Value: [any string]")
			fmt.Scan(&Value)
			err = client.Put(Key, Value, timeout)
		case 3:
			fmt.Println("please enter Value: [any string]")
			fmt.Scan(&Value)
			err = client.Append(Key, Value, timeout)
		}
		if err != nil {
			fmt.Println("Process error: ", err)
			continue
		}
		if Op == 1 {
			fmt.Println("Process OK, you get: ", Got)
		} else {
			fmt.Println("Process OK")
		}
	}
}
