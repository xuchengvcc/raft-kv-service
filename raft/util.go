package raft

import "log"

const DBNAME string = "my.db"
const SNAPNAME string = "snap"
const STATNAME string = "stat"

// Debugging
const Debug = true

func DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(format, a...)
	}
}
