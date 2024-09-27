package main

import (
	"fmt"
	"raft-kv-service/conf"
	"raft-kv-service/server"
	"time"
)

func main() {
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
