package persister

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
)

const (
	PrivateFileMode = 0600
	PrivateDirMode  = 0700
)

var (
	ErrLocked = errors.New("fileutil: file already locked")
	// errBadWALName = errors.New("bad wal name")
	linuxLockFile = flockLockFile
)

type LockedFile struct{ *os.File }

func WriteAndSyncFile(filename string, data []byte, perm os.FileMode) error {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}

	if err == nil {
		err = Fsync(f)
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return err
}

func Fsync(f *os.File) error {
	return f.Sync()
}

func Assert(condition bool, msg string, v ...any) {
	if !condition {
		panic(fmt.Sprintf("assertion failed: "+msg, v...))
	}
}

func MustMarshal(m protoreflect.ProtoMessage) ([]byte, error) {
	b, err := proto.Marshal(m)
	if err != nil {
		panic(fmt.Sprintf("marshal should never fail (%v)", err))
	}
	return b, err
}

func Fdatasync(f *os.File) error {
	return syscall.Fdatasync(int(f.Fd()))
}

// func walName(seq, index uint64) string {
// 	return fmt.Sprintf("%016x-%016x.wal", seq, index)
// }

// func parseWALName(str string) (seq, index uint64, err error) {
// 	if !strings.HasSuffix(str, ".wal") {
// 		return 0, 0, errBadWALName
// 	}
// 	_, err = fmt.Sscanf(str, "%016x-%016x.wal", &seq, &index)
// 	return seq, index, err
// }

func LockFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	return linuxLockFile(path, flag, perm)
}

func Preallocate(f *os.File, sizeInBytes int64, extendFile bool) error {
	if sizeInBytes == 0 {
		// fallocate will return EINVAL if length is 0; skip
		return nil
	}
	if extendFile {
		return preallocExtend(f, sizeInBytes)
	}
	return preallocFixed(f, sizeInBytes)
}

func preallocFixed(f *os.File, sizeInBytes int64) error {
	// use mode = 1 to keep size; see FALLOC_FL_KEEP_SIZE
	err := syscall.Fallocate(int(f.Fd()), 1, 0, sizeInBytes)
	if err != nil {
		errno, ok := err.(syscall.Errno)
		// treat not supported as nil error
		if ok && errno == syscall.ENOTSUP {
			return nil
		}
	}
	return err
}

func preallocExtend(f *os.File, sizeInBytes int64) error {
	// use mode = 0 to change size
	err := syscall.Fallocate(int(f.Fd()), 0, 0, sizeInBytes)
	if err != nil {
		errno, ok := err.(syscall.Errno)
		// not supported; fallback
		// fallocate EINTRs frequently in some environments; fallback
		if ok && (errno == syscall.ENOTSUP || errno == syscall.EINTR) {
			return preallocExtendTrunc(f, sizeInBytes)
		}
	}
	return err
}

func preallocExtendTrunc(f *os.File, sizeInBytes int64) error {
	curOff, err := f.Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	size, err := f.Seek(sizeInBytes, io.SeekEnd)
	if err != nil {
		return err
	}
	if _, err = f.Seek(curOff, io.SeekStart); err != nil {
		return err
	}
	if sizeInBytes > size {
		return nil
	}
	return f.Truncate(sizeInBytes)
}

func flockLockFile(path string, flag int, perm os.FileMode) (*LockedFile, error) {
	f, err := os.OpenFile(path, flag, perm)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, err
	}
	return &LockedFile{f}, err
}

func Exist(name string) bool {
	_, err := os.Stat(name)
	return err == nil
}

func ExistWal(dir string) bool {
	names, err := ReadDir(dir, WithExt(".wal"))
	if err != nil {
		return false
	}
	return len(names) != 0
}

type ReadDirOp struct {
	ext string
}

// ReadDirOption configures archiver operations.
type ReadDirOption func(*ReadDirOp)

func WithExt(ext string) ReadDirOption {
	return func(op *ReadDirOp) { op.ext = ext }
}

func (op *ReadDirOp) applyOpts(opts []ReadDirOption) {
	for _, opt := range opts {
		opt(op)
	}
}

func ReadDir(d string, opts ...ReadDirOption) ([]string, error) {
	op := &ReadDirOp{}
	op.applyOpts(opts)

	dir, err := os.Open(d)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	sort.Strings(names)

	if op.ext != "" {
		tss := make([]string, 0)
		for _, v := range names {
			if filepath.Ext(v) == op.ext {
				tss = append(tss, v)
			}
		}
		names = tss
	}
	return names, nil
}

func CreateDirAll(lg *zap.Logger, dir string) error {
	err := TouchDirAll(lg, dir)
	if err == nil {
		var ns []string
		ns, err = ReadDir(dir)
		if err != nil {
			return err
		}
		if len(ns) != 0 {
			err = fmt.Errorf("expected %q to be empty, got %q", dir, ns)
		}
	}
	return err
}

func TouchDirAll(lg *zap.Logger, dir string) error {
	Assert(lg != nil, "nil log isn't allowed")
	// If path is already a directory, MkdirAll does nothing and returns nil, so,
	// first check if dir exists with an expected permission mode.
	if Exist(dir) {
		err := CheckDirPermission(dir, PrivateDirMode)
		if err != nil {
			lg.Warn("check file permission", zap.Error(err))
		}
	} else {
		err := os.MkdirAll(dir, PrivateDirMode)
		if err != nil {
			// if mkdirAll("a/text") and "text" is not
			// a directory, this will return syscall.ENOTDIR
			return err
		}
	}

	return IsDirWriteable(dir)
}

func CheckDirPermission(dir string, perm os.FileMode) error {
	if !Exist(dir) {
		return fmt.Errorf("directory %q empty, cannot check permission", dir)
	}
	//check the existing permission on the directory
	dirInfo, err := os.Stat(dir)
	if err != nil {
		return err
	}
	dirMode := dirInfo.Mode().Perm()
	if dirMode != perm {
		err = fmt.Errorf("directory %q exist, but the permission is %q. The recommended permission is %q to prevent possible unprivileged access to the data", dir, dirInfo.Mode(), os.FileMode(PrivateDirMode))
		return err
	}
	return nil
}

func IsDirWriteable(dir string) error {
	f, err := filepath.Abs(filepath.Join(dir, ".touch"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(f, []byte(""), PrivateFileMode); err != nil {
		return err
	}
	return os.Remove(f)
}
