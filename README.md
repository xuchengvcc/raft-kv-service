# raft-kv-service

# test single client
```
cd server
./run_tests.sh -cp CPU.out -f TestBasic
```

# check CPU usage
```
go tool pprof CPU.out
```