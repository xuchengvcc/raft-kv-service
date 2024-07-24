package persister

// type crcentry struct {
// 	Crc   uint32
// 	Index int64
// }

type logEntry struct {
	Index int64
	Entry []byte
}
