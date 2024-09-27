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
)

func start() {

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
