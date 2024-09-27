package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"net"
	cm "raft-kv-service/common"
	"raft-kv-service/mylog"
	"raft-kv-service/proxy"
	"sync"
	"sync/atomic"
	"time"
)

// 最大调整权重次数，调整次数超过该值时，将权重的最大差距缩小一些，以免节点间选择偏好差距过大
const MaxAdjustTime int32 = 100000
const MeanWeight int32 = 10000000
const (
	WEIGHT = iota
	RANDOM
	LOG
)

type AdjustOp struct {
	Id    int32
	Value int32
}

type Node struct {
	CommitIndex int64
	IsLeader    bool
	Port        int32
	Ip          string
	// SyncPort    int32
}

type Cluster struct {
	mu         sync.Mutex
	Nodes      map[int32]*Node
	Weights    map[int32]int32 // 依据权重影响读请求的节点分配：(1) 处理中的请求会导致权重降低 (2) 复制进度快可以相对增加权重 (3) 处理请求失败将降低选取权重
	AdjustCh   chan AdjustOp
	PickMode   uint32
	LeaderId   int32
	MsgProc    *MsgProcess
	ServAddr   string
	SyncAddr   string
	dead       int32
	Conns      map[int32]*net.Conn
	ProxyConns map[int32]*net.Conn
	SendQue    chan *proxy.ClientMessage
	RespQue    chan *proxy.ServerResponse
	FailQue    chan *proxy.ServerResponse
}

func (cu *Cluster) kill() {
	atomic.StoreInt32(&cu.dead, 1)
}

func (cu *Cluster) killed() bool {
	z := atomic.LoadInt32(&cu.dead)
	return z == 1
}

func (cu *Cluster) SetPickMod(mod uint32) {
	if mod == RANDOM || mod == LOG || mod == WEIGHT {
		cu.PickMode = mod
	} else {
		cu.PickMode = WEIGHT
	}
}

func NewCluster(servAddr, syncAddr string) *Cluster {
	return &Cluster{
		Nodes:      make(map[int32]*Node),
		Weights:    make(map[int32]int32),
		AdjustCh:   make(chan AdjustOp, 20),
		PickMode:   WEIGHT,
		MsgProc:    NewMsgProcess(),
		ServAddr:   servAddr,
		SyncAddr:   syncAddr,
		dead:       0,
		Conns:      make(map[int32]*net.Conn),
		ProxyConns: make(map[int32]*net.Conn),
		SendQue:    make(chan *proxy.ClientMessage, 100),
		RespQue:    make(chan *proxy.ServerResponse, 100),
		FailQue:    make(chan *proxy.ServerResponse, 100),
	}
}

func (cu *Cluster) Start() {
	mylog.DPrintln("[Cluster Manager] Start")
	go cu.Moniter()
	go cu.Serve()
}

func (cu *Cluster) procSendQue() {
	for !cu.killed() {
		msg := <-cu.SendQue
		var conn *net.Conn = nil
		for conn == nil {

			selectId := cu.pick(msg.OpId)
			cu.mu.Lock()
			conn = cu.Conns[selectId]
			cu.mu.Unlock()
			if conn == nil {
				err := cu.connectToNode(selectId)
				if err != nil {
					mylog.DPrintf("[Cluster Manager] connect to server %d failed error:%s \n", selectId, err)
					// TODO：无法与响应节点建立连接，将该节点设置为不可用，以免重复pick该节点
					time.Sleep(100 * time.Millisecond)
					continue
				}
				conn = cu.Conns[selectId]
			}
			bitData, err := cu.MsgProc.ClientMsgToBit(msg)
			if err != nil {
				mylog.DPrintf("[Cluster Manager] req message(Clent: %d) marshal error: %s", msg.ClerkId, err)
				continue
			}
			_, err = (*conn).Write(bitData)
			cu.AdjustCh <- AdjustOp{Id: selectId, Value: -1}
			if err != nil {
				mylog.DPrintln("[Cluster Manager] send req message error: ", err)
			}
			mylog.DPrintln("[Cluster Manager] send req message succ")
		}
	}
}

func (cu *Cluster) procRespQue() {
	for !cu.killed() {
		resp := <-cu.RespQue
		cu.mu.Lock()
		conn, exist := cu.ProxyConns[resp.ClerkId]
		cu.mu.Unlock()
		if !exist {
			mylog.DPrintln("[Cluster Manager] proxy conn dont exist ")
			continue
		}
		bitData, err := cu.MsgProc.RespToBinary(resp)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] marshal resp msg error: ", err)
			continue
		}
		_, err = (*conn).Write(bitData)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] send resp msg to proxy error: ", err)
		}
	}
}

func (cu *Cluster) procFailedReq() {
	for !cu.killed() {
		resp := <-cu.FailQue
		cu.AdjustCh <- AdjustOp{Id: resp.ServerId, Value: -10}
		switch resp.Err {
		case cm.ErrMap[cm.ErrLeaderChanged]:
		case cm.ErrMap[cm.ErrWrongLeader]:
			go func(res *proxy.ServerResponse) {
				// 如果leader已更新，则直接重新请求
				cu.mu.Lock()
				if cu.LeaderId != res.ServerId {
					cu.SendQue <- cu.MsgProc.RespToReq(res)
					cu.mu.Unlock()
					return
				}
				cu.mu.Unlock()
				// 等待同步leader消息
				time.Sleep(800 * time.Millisecond)
				cu.SendQue <- cu.MsgProc.RespToReq(res)
			}(resp)
		case cm.ErrMap[cm.ErrChanClosed]:
		case cm.ErrMap[cm.ErrHandleTimeout]:
			// 暂时直接重试，让均衡器重新分配
			go func(res *proxy.ServerResponse) {
				cu.SendQue <- cu.MsgProc.RespToReq(res)
			}(resp)
		}
	}
}

func (cu *Cluster) serveServ(conn net.Conn) {
	defer conn.Close()
	for !cu.killed() {
		headBuffer := make([]byte, cu.MsgProc.serRespPack.GetHeadLen())
		_, err := io.ReadFull(conn, headBuffer)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] failed to read resp msg head, error: ", err)
			return
		}
		msg, err := cu.MsgProc.serRespPack.Unpack(headBuffer)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] failed to unmarshal resp msg head, error: ", err)
			return
		}
		resp := msg.(*proxy.ServerResponse)
		bodyBuff := make([]byte, resp.DataLen)
		_, err = io.ReadFull(conn, bodyBuff)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] failed to read resp body, error: ", err)
			return
		}
		resp.SetData(bodyBuff)
		if resp.Err == cm.ErrMap[cm.OK] || resp.Err == cm.ErrMap[cm.ErrNoKey] {
			cu.RespQue <- resp
			cu.AdjustCh <- AdjustOp{Id: resp.ServerId, Value: 1}
		} else {
			cu.FailQue <- resp
		}
	}
}

func (cu *Cluster) connectToNode(id int32) error {
	cu.mu.Lock()
	if _, exist := cu.Nodes[id]; !exist {
		cu.mu.Unlock()
		return fmt.Errorf("[Cluster Manager] target node info is empty")
	}
	addr := fmt.Sprintf("%s:%d", cu.Nodes[id].Ip, cu.Nodes[id].Port)
	cu.mu.Unlock()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("[Cluster Manager] connect to target node error: %s", err)
	}
	cu.mu.Lock()
	cu.Conns[id] = &conn
	cu.mu.Unlock()
	go cu.serveServ(conn)
	return nil
}

func (cu *Cluster) handleSync(conn net.Conn) {
	defer conn.Close()
	for !cu.killed() {
		headBuffer := make([]byte, cu.MsgProc.syncMsgPack.GetHeadLen())
		_, err := io.ReadFull(conn, headBuffer)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] failed to read sync msg head error: ", err)
			return
		}
		msg, err := cu.MsgProc.SyncHeadUnpack(headBuffer)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] failed to unmarsh sync msg head, error: ", err)
			return
		}
		// mylog.DPrintf("[Cluster Manager] receive sync msg Id: %d, CmtIdx: %d, IpLen: %d", msg.ServerId, msg.CommitIndex, msg.IpLen)
		if msg.IpLen > 0 {
			bodyBuffer := make([]byte, msg.IpLen+4)
			_, err := io.ReadFull(conn, bodyBuffer)
			if err != nil {
				mylog.DPrintln("[Cluster Manager] failed to read sync msg body, error: ", err)
				return
			}
			if err := cu.MsgProc.SyncBodyUnpack(bodyBuffer, msg); err != nil {
				mylog.DPrintln("[Cluster Manager] failed to unmarsh sync msg body, error :", err)
				return
			}
			mylog.DPrintf("[Cluster Manager] receive sync msg Id: %d, CmtIdx: %d, Ip: %s, Port: %d", msg.ServerId, msg.CommitIndex, msg.Ip, msg.Port)
			cu.mu.Lock()
			if node, exist := cu.Nodes[msg.ServerId]; !exist {
				cu.Nodes[msg.ServerId] = &Node{
					CommitIndex: msg.CommitIndex,
					IsLeader:    msg.IsLeader,
					Ip:          msg.Ip,
					Port:        msg.Port,
				}
			} else {
				node.CommitIndex = msg.CommitIndex
				node.IsLeader = msg.IsLeader
				node.Ip = msg.Ip
				node.Port = msg.Port
			}
			if msg.IsLeader {
				cu.LeaderId = msg.ServerId
			}
			cu.mu.Unlock()
			respBuff := bytes.NewBuffer(make([]byte, 4))
			binary.Write(respBuff, binary.LittleEndian, cm.ErrMap[cm.OK])
			if _, err := conn.Write(respBuff.Bytes()); err != nil {
				mylog.DPrintln("[Cluster Manager] confirm sync msg error: ", err)
			}
			mylog.DPrintln("[Cluster Manager] confirm sync send succ")
			cu.mu.Lock()
			_, exist := cu.Conns[msg.ServerId]
			cu.mu.Unlock()
			if !exist {
				cu.connectToNode(msg.ServerId)
			}
			continue
		}
		var diff int32
		cu.mu.Lock()
		mylog.DPrintf("[Cluster Manager] receive sync msg Id: %d, CmtIdx: %d", msg.ServerId, msg.CommitIndex)
		if msg.IsLeader {
			cu.LeaderId = msg.ServerId
		}
		if node, exist := cu.Nodes[msg.ServerId]; !exist {
			cu.Nodes[msg.ServerId] = &Node{
				CommitIndex: msg.CommitIndex,
				IsLeader:    msg.IsLeader,
			}
		} else {
			node.CommitIndex = msg.CommitIndex
			node.IsLeader = msg.IsLeader
		}
		if leader, exist := cu.Nodes[cu.LeaderId]; !exist {
			diff = 0
		} else {
			diff = int32(cu.Nodes[msg.ServerId].CommitIndex - leader.CommitIndex)
		}
		cu.mu.Unlock()
		cu.AdjustCh <- AdjustOp{Id: msg.ServerId, Value: diff}
	}
}

func (cu *Cluster) Moniter() error {
	mylog.DPrintln("[Cluster Manager] start moniter")
	lis, err := net.Listen("tcp", cu.SyncAddr)
	defer func() {
		err := lis.Close()
		if err != nil {
			mylog.DPrintln("[Cluster Manager] close moniter listenner error: ", err)
		}
	}()
	if err != nil {
		return err
	}
	go cu.adjustWeight()
	for !cu.killed() {
		conn, err := lis.Accept()
		if err != nil {
			mylog.DPrintln("[Cluster Manager] accept sync conn error: ", err)
			return err
		}
		mylog.DPrintf("[Cluster Manager] accept sync conn succ: %s", conn.RemoteAddr())
		go cu.handleSync(conn)
	}
	return nil
}

func (cu *Cluster) balanceWeight() {
	var totalWeight int32
	for _, v := range cu.Weights {
		totalWeight += v
	}
	avgWeight := totalWeight / int32(len(cu.Weights))
	for id, v := range cu.Weights {
		v += MeanWeight - avgWeight
		v += (v - MeanWeight) / 2
		cu.Weights[id] = v
	}
}

func (cu *Cluster) adjustWeight() {
	var op AdjustOp
	var count int32 = 0
	for !cu.killed() {
		op = <-cu.AdjustCh
		count++
		cu.mu.Lock()
		if count > MaxAdjustTime {
			cu.balanceWeight()
			count = 0
		}
		if weight, exist := cu.Weights[op.Id]; !exist {
			cu.Weights[op.Id] = MeanWeight + op.Value
		} else {
			cu.Weights[op.Id] = weight + op.Value
		}
		cu.mu.Unlock()
	}

}

func (cu *Cluster) pick(op uint32) int32 {
	cu.mu.Lock()
	defer cu.mu.Unlock()
	if op == GET {
		var selectId int32 = cu.LeaderId
		switch cu.PickMode {
		case WEIGHT:
			var totalWeight int32 = 0
			for _, v := range cu.Weights {
				totalWeight += v
			}
			pickedWeight := rand.Int31n(totalWeight)
			for id, v := range cu.Weights {
				pickedWeight -= v
				if pickedWeight <= 0 {
					selectId = id
					break
				}
			}
		case RANDOM:
			randidx := rand.Intn(len(cu.Weights))
			for randidx > 0 {
				for id := range cu.Weights {
					if id == cu.LeaderId {
						continue
					}
					selectId = id
					randidx--
					if randidx < 1 {
						break
					}
				}
			}
		case LOG:
			var bestCommit int64 = 0
			cu.mu.Lock()
			for id, node := range cu.Nodes {
				if id == cu.LeaderId {
					continue
				}
				if bestCommit < node.CommitIndex {
					bestCommit = node.CommitIndex
					selectId = id
				}
			}
		}
		return selectId
	} else {
		return cu.LeaderId
	}
}

func (cu *Cluster) serve(conn net.Conn) {
	defer conn.Close()
	for !cu.killed() {
		headerBuffer := make([]byte, cu.MsgProc.clientDataPack.GetHeadLen())
		_, err := io.ReadFull(conn, headerBuffer)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] failed to read request from proxy, error: ", err)
			return
		}
		msg, err := cu.MsgProc.clientDataPack.Unpack(headerBuffer)

		cmsg := msg.(*proxy.ClientMessage)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] failed to unmarshal request from proxy, error: ", err)
			return
		}
		body := make([]byte, cmsg.GetDataLen())
		_, err = io.ReadFull(conn, body)
		if err != nil {
			mylog.DPrintln("[Cluster Manager] failed to read request body from proxy, error: ", err)
			return
		}

		cmsg.SetData(body)
		cu.SendQue <- cmsg

		cu.mu.Lock()
		if oldConn, exist := cu.ProxyConns[cmsg.ClerkId]; !exist || *oldConn != conn {
			cu.ProxyConns[cmsg.ClerkId] = &conn
		}
		cu.mu.Unlock()
	}
}

func (cu *Cluster) Serve() error {
	mylog.DPrintln("[Cluster Manager] start serve")
	lis, err := net.Listen("tcp", cu.ServAddr)
	defer func() {
		err := lis.Close()
		if err != nil {
			mylog.DPrintln("[Cluster Manager] close serverSide listenner error: ", err)
		}
	}()
	if err != nil {
		return err
	}
	go cu.procSendQue()
	go cu.procRespQue()
	go cu.procFailedReq()
	for !cu.killed() {
		conn, err := lis.Accept()
		if err != nil {
			mylog.DPrintln("[Cluster Manager] cluster manager accept proxy conn error: ", err)
			return err
		}
		go cu.serve(conn)
	}
	return nil
}

func (cu *Cluster) Stop() {
	cu.kill()
	close(cu.SendQue)
	close(cu.RespQue)
	close(cu.FailQue)
	mylog.DPrintln("[Cluster Manager] Stop")
}
