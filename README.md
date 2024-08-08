# raft-kv-service

# test single client
```
cd server
./run_tests.sh -cp CPU.out -f TestBasic
```

# CPU usage
```
go tool pprof CPU.out
```