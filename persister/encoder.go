package persister

import (
	"encoding/binary"
	"io"
	"os"
	"sync"
)

const minSectorSize = 512
const walPageBytes = 8 * minSectorSize

type encoder struct {
	mu        sync.Mutex
	bw        *PageWriter
	buf       []byte
	uint64buf []byte
}

func (e *encoder) encode(b []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, lenField := prepareDataWithPadding(b)
	return write(e.bw, e.uint64buf, data, lenField)

}

func prepareDataWithPadding(data []byte) ([]byte, uint64) {
	lenField, padBytes := encodeFrameSize(len(data))
	if padBytes != 0 {
		data = append(data, make([]byte, padBytes)...)
	}
	return data, lenField
}

func encodeFrameSize(dataBytes int) (lenField uint64, padBytes int) {
	lenField = uint64(dataBytes)
	padBytes = (8 - (dataBytes % 8)) % 8
	if padBytes != 0 {
		lenField |= uint64(0x80|padBytes) << 56
	}
	return lenField, padBytes
}

func write(w io.Writer, uint64buf, data []byte, lenField uint64) error {
	binary.LittleEndian.PutUint64(uint64buf, lenField)
	nv, err := w.Write(uint64buf)
	walWriteBytes.Add(float64(nv))
	if err != nil {
		return err
	}
	n, err := w.Write(data)
	walWriteBytes.Add(float64(n))
	return err
}

func (e *encoder) flush() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.bw.Flush()
}

func newFileEncoder(f *os.File, prevCrc uint32) (*encoder, error) {
	offset, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	return newEncoder(f, prevCrc, int(offset)), nil
}

func newEncoder(w io.Writer, prevCrc uint32, pageOffset int) *encoder {
	return &encoder{
		bw: NewPageWriter(w, walPageBytes, pageOffset),
		// 1MB buffer
		buf:       make([]byte, 1024*1024),
		uint64buf: make([]byte, 8),
	}
}
