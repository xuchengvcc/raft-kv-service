# raft-kv-service
## start
### protobuf
to generate `name_grpc.pb.go` and `name.pb.go`
```
protoc --go_out=. --go-grpc_out=. name.proto
```
### configuration

Please first go to the `./conf` folder to set the address information: `server.json` and `proxy.json`

cluster info:
```
# sever.json

{
    "nodeNum":5,
    "SyncAddr":"127.0.0.1:9999", // cluster manager moniter port
    "ServAddr":"127.0.0.1:9998", // cluster manager serve port
    "servers":[
        {
            // node info
            "Name":"node1",
            "SerAddr":"127.0.0.1:50000",
            "NodeAddr":"127.0.0.1:50030",
            "SyncAddr":"127.0.0.1:50050",
        },
        ......
    ]
}
```

proxy info:

```
// proxy.json
{
    "GetAddr":"127.0.0.1:9000", // read request port
    "PutAddr":"127.0.0.1:9001", // write request port
}
```


### start a server cluster

```
cd ./main
go run main.go -s server
```

### start a client
```
go run main.go -s client
```