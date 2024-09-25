# raft-kv-service

### protobuf
to generate `name_grpc.pb.go` and `name.pb.go`
```
protoc --go_out=. --go-grpc_out=. name.proto
```

# test single client


```
cd server
./run_tests.sh -cp CPU.out -f TestBasic
```

# CPU Utilization
```
go tool pprof CPU.out
```