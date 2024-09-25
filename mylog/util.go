package mylog

import "log"

// Debugging
const Debug = true

func DPrintf(format string, a ...interface{}) {
	if Debug {
		log.Printf(format, a...)
	}
}

func DPrintln(a ...interface{}) {
	if Debug {
		log.Println(a...)
	}
}
