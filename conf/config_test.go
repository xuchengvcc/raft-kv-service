package conf

import (
	"raft-kv-service/mylog"
	"testing"
)

func TestReload(t *testing.T) {
	globalObj := GetGlobalObj()
	mylog.DPrintln(globalObj)
}
