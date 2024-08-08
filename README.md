# raft-kv-service

# test single client
```
cd server
./run_tests.sh -cp CPU.out -f TestBasic
```

# CPU Utilization
```
go tool pprof CPU.out
```