package conf

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"raft-kv-service/mylog"
	"strings"
)

type clientConfig struct {
	ClientNum int `json:"ClientNum"`
}

var cliConfig *clientConfig

type Node struct {
	Name     string `json:"Name"`
	SerAddr  string `json:"SerAddr"`
	NodeAddr string `json:"NodeAddr"`
	SyncAddr string `json:"SyncAddr"`
}

type serversConfig struct {
	NodeNum  int    `json:"nodeNum"`
	SyncAddr string `json:"SyncAddr"`
	ServAddr string `json:"ServAddr"`
	Servers  []Node `json:"servers"`
}

var sersConfig *serversConfig

type proxyConfg struct {
	GetAddr  string `json:"GetAddr"`
	PutAddr  string `json:"PutAddr"`
	ResAddr  string `json:"ResAddr"`
	SyncAddr string `json:"SyncAddr"`
}

var proxyAddr *proxyConfg

type GlobalObj struct {
	ProxyAddrs  map[uint32]string
	ClusterConf serversConfig
	ServerAddrs []map[string]string
	ClientNum   int
}

var GlobalObject *GlobalObj

func (g *GlobalObj) Reload() {
	cwd, err := os.Getwd()
	if err != nil {
		mylog.DPrintln("get cwd error: ", err)
		return
	}
	proxyFile, err := os.ReadFile(filepath.Join(cwd, "..", "conf", "proxy.json"))
	if err != nil {
		mylog.DPrintln("config reload failed, use default proxy config, err: ", err)
		return
	}
	err = json.Unmarshal(proxyFile, &proxyAddr)
	if err != nil {
		panic(err)
	}

	serversFile, err := os.ReadFile(filepath.Join(cwd, "..", "conf", "server.json"))
	if err != nil {
		mylog.DPrintln("config reload failed, use default server config, err: ", err)
		return
	}
	err = json.Unmarshal(serversFile, &sersConfig)
	if err != nil {
		panic(err)
	}

	cliNum := g.ClientNum
	clientFile, err := os.ReadFile(filepath.Join(cwd, "..", "conf", "client.json"))
	if err != nil {
		mylog.DPrintln("config reload failed, use default clientNum, err: ", err)
	} else {
		err = json.Unmarshal(clientFile, &cliConfig)
		if err != nil {
			mylog.DPrintln("config unmarshal failed, use default clientNum, err: ", err)
		} else {
			cliNum = cliConfig.ClientNum
		}
	}

	g.ProxyAddrs[0] = proxyAddr.GetAddr
	g.ProxyAddrs[1] = proxyAddr.PutAddr
	g.ServerAddrs = make([]map[string]string, sersConfig.NodeNum)
	for i := 0; i < sersConfig.NodeNum; i++ {
		GlobalObject.ServerAddrs[i] = make(map[string]string)
		g.ServerAddrs[i]["cluster_sync"] = sersConfig.SyncAddr
		g.ServerAddrs[i]["cluster_serv"] = sersConfig.ServAddr
		g.ServerAddrs[i]["raft"] = sersConfig.Servers[i].NodeAddr
		g.ServerAddrs[i]["server"] = sersConfig.Servers[i].SerAddr
		g.ServerAddrs[i]["sync"] = sersConfig.Servers[i].SyncAddr
	}
	g.ClientNum = cliNum
	mylog.DPrintln("config reload succ.")
	mylog.DPrintln(g)

}

func init() {
	defaultSerNum := 5
	GlobalObject = &GlobalObj{
		ProxyAddrs:  make(map[uint32]string),
		ServerAddrs: make([]map[string]string, 5),
		ClusterConf: serversConfig{},
		ClientNum:   1,
	}
	GlobalObject.ProxyAddrs[0] = "127.0.0.1:9000"
	GlobalObject.ProxyAddrs[1] = "127.0.0.1:9001"
	GlobalObject.ClusterConf.NodeNum = defaultSerNum
	GlobalObject.ClusterConf.ServAddr = "127.0.0.1:9998"
	GlobalObject.ClusterConf.SyncAddr = "127.0.0.1:9999"
	GlobalObject.ClusterConf.Servers = make([]Node, 0, defaultSerNum)
	for i := 0; i < defaultSerNum; i++ {
		GlobalObject.ClusterConf.Servers = append(GlobalObject.ClusterConf.Servers, Node{
			Name:     fmt.Sprintf("server_%d", i),
			SerAddr:  fmt.Sprintf("127.0.0.1:5000%d", i),
			NodeAddr: fmt.Sprintf("127.0.0.1:5003%d", i),
			SyncAddr: fmt.Sprintf("127.0.0.1:5005%d", i),
		})
		GlobalObject.ServerAddrs[i] = make(map[string]string)
		GlobalObject.ServerAddrs[i]["cluster_sync"] = GlobalObject.ClusterConf.SyncAddr
		GlobalObject.ServerAddrs[i]["cluster_serv"] = GlobalObject.ClusterConf.ServAddr
		GlobalObject.ServerAddrs[i]["raft"] = GlobalObject.ClusterConf.Servers[i].NodeAddr
		GlobalObject.ServerAddrs[i]["server"] = GlobalObject.ClusterConf.Servers[i].SerAddr
		GlobalObject.ServerAddrs[i]["sync"] = GlobalObject.ClusterConf.Servers[i].SyncAddr
	}

	GlobalObject.Reload()
}

func (g *GlobalObj) String() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ClientNum: %d\n", g.ClientNum))
	sb.WriteString("ProxyAddrs:\n")
	for id, addr := range g.ProxyAddrs {
		sb.WriteString(fmt.Sprintf("  [%d]: %s\n", id, addr))
	}

	sb.WriteString("ServerAddrs:\n")
	for i, server := range g.ServerAddrs {
		sb.WriteString(fmt.Sprintf("  Server %d:\n", i))
		for key, addr := range server {
			sb.WriteString(fmt.Sprintf("    %s: %s\n", key, addr))
		}
	}

	return sb.String()
}

func GetGlobalObj() *GlobalObj {
	return GlobalObject
}
