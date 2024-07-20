package persister

import (
	"io"
	"os"
	"path/filepath"
	"raft-kv-service/raft"
	"sync"
	"time"

	"go.uber.org/zap"
)

var SegmentSizeBytes int64 = 64 * 1000 * 1000 // 64MB
const (
	warnSyncDuration = time.Second
)

type Wal struct {
	lg            *zap.Logger
	mu            sync.Mutex
	dir           string // the living directory of the underlay files
	dirFile       *os.File
	state         *State
	encoder       *encoder
	enti          uint64
	walWriteBytes int64
	locks         []*LockedFile // the locked files the WAL holds (the name is increasing)
	unsafeNoSync  bool          // if set, do not fsync
	fp            *filePipeline
}

func Create(lg *zap.Logger, dirpath string, metadata []byte) (*WAL, error) {
	if ExistWal(dirpath) {
		return nil, os.ErrExist
	}

	if lg == nil {
		lg = zap.NewNop()
	}

	// keep temporary wal directory so WAL initialization appears atomic
	tmpdirpath := filepath.Clean(dirpath) + ".tmp"
	if Exist(tmpdirpath) {
		if err := os.RemoveAll(tmpdirpath); err != nil {
			return nil, err
		}
	}
	defer os.RemoveAll(tmpdirpath)

	if err := CreateDirAll(lg, tmpdirpath); err != nil {
		lg.Warn(
			"failed to create a temporary WAL directory",
			zap.String("tmp-dir-path", tmpdirpath),
			zap.String("dir-path", dirpath),
			zap.Error(err),
		)
		return nil, err
	}

	p := filepath.Join(tmpdirpath, walName(0, 0))
	f, err := LockFile(p, os.O_WRONLY|os.O_CREATE, PrivateFileMode)
	if err != nil {
		lg.Warn(
			"failed to flock an initial WAL file",
			zap.String("path", p),
			zap.Error(err),
		)
		return nil, err
	}
	if _, err = f.Seek(0, io.SeekEnd); err != nil {
		lg.Warn(
			"failed to seek an initial WAL file",
			zap.String("path", p),
			zap.Error(err),
		)
		return nil, err
	}
	if err = Preallocate(f.File, SegmentSizeBytes, true); err != nil {
		lg.Warn(
			"failed to preallocate an initial WAL file",
			zap.String("path", p),
			zap.Int64("segment-bytes", SegmentSizeBytes),
			zap.Error(err),
		)
		return nil, err
	}

	w := &Wal{
		lg:  lg,
		dir: dirpath,
	}
	w.encoder, err = newFileEncoder(f.File, 0)
	if err != nil {
		return nil, err
	}
	w.locks = append(w.locks, f)

	logDirPath := w.dir
	if w, err = w.renameWAL(tmpdirpath); err != nil {
		lg.Warn(
			"failed to rename the temporary WAL directory",
			zap.String("tmp-dir-path", tmpdirpath),
			zap.String("dir-path", logDirPath),
			zap.Error(err),
		)
		return nil, err
	}

	var perr error
	defer func() {
		if perr != nil {
			w.cleanupWAL(lg)
		}
	}()

	// directory was renamed; sync parent dir to persist rename
	pdir, perr := fileutil.OpenDir(filepath.Dir(w.dir))
	if perr != nil {
		lg.Warn(
			"failed to open the parent data directory",
			zap.String("parent-dir-path", filepath.Dir(w.dir)),
			zap.String("dir-path", w.dir),
			zap.Error(perr),
		)
		return nil, perr
	}
	dirCloser := func() error {
		if perr = pdir.Close(); perr != nil {
			lg.Warn(
				"failed to close the parent data directory file",
				zap.String("parent-dir-path", filepath.Dir(w.dir)),
				zap.String("dir-path", w.dir),
				zap.Error(perr),
			)
			return perr
		}
		return nil
	}
	start := time.Now()
	if perr = Fsync(pdir); perr != nil {
		dirCloser()
		lg.Warn(
			"failed to fsync the parent data directory file",
			zap.String("parent-dir-path", filepath.Dir(w.dir)),
			zap.String("dir-path", w.dir),
			zap.Error(perr),
		)
		return nil, perr
	}
	walFsyncSec.Observe(time.Since(start).Seconds())
	if err = dirCloser(); err != nil {
		return nil, err
	}

	return w, nil
}

func (w *Wal) renameWAL(tmpdirpath string) (*WAL, error) {
	if err := os.RemoveAll(w.dir); err != nil {
		return nil, err
	}
	// On non-Windows platforms, hold the lock while renaming. Releasing
	// the lock and trying to reacquire it quickly can be flaky because
	// it's possible the process will fork to spawn a process while this is
	// happening. The fds are set up as close-on-exec by the Go runtime,
	// but there is a window between the fork and the exec where another
	// process holds the lock.
	if err := os.Rename(tmpdirpath, w.dir); err != nil {
		if _, ok := err.(*os.LinkError); ok {
			return w.renameWALUnlock(tmpdirpath)
		}
		return nil, err
	}
	w.fp = newFilePipeline(w.lg, w.dir, SegmentSizeBytes)
	df, err := fileutil.OpenDir(w.dir)
	w.dirFile = df
	return w, err
}

func (w *WAL) renameWALUnlock(tmpdirpath string) (*WAL, error) {
	// rename of directory with locked files doesn't work on windows/cifs;
	// close the WAL to release the locks so the directory can be renamed.
	w.lg.Info(
		"closing WAL to release flock and retry directory renaming",
		zap.String("from", tmpdirpath),
		zap.String("to", w.dir),
	)
	w.Close()

	if err := os.Rename(tmpdirpath, w.dir); err != nil {
		return nil, err
	}

	// reopen and relock
	newWAL, oerr := Open(w.lg, w.dir, walpb.Snapshot{})
	if oerr != nil {
		return nil, oerr
	}
	if _, _, _, err := newWAL.ReadAll(); err != nil {
		newWAL.Close()
		return nil, err
	}
	return newWAL, nil
}

func (w *Wal) Save(st *State, ents []raft.Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if IsEmptyState(st) && len(ents) == 0 {
		return nil
	}

	mustSync := MustSync(st, w.state, len(ents))

	for i := range ents {
		if err := w.saveEntry(&ents[i]); err != nil {
			return err
		}
	}

	if err := w.saveState(st); err != nil {
		return err
	}

	curOff, err := w.tail().Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}
	if curOff < SegmentSizeBytes {
		if mustSync {
			err = w.sync()
			return err
		}
		return nil
	}
	return w.cut()
}

func (w *Wal) saveEntry(ent *raft.Entry) error {
	b, _ := MustMarshal(ent)
	if err := w.encoder.encode(b); err != nil {
		return err
	}
	w.enti = ent.Command.IncrId
	return nil
}

func (w *Wal) saveState(ent *State) error {
	if IsEmptyState(ent) {
		return nil
	}
	w.state = ent
	b, _ := MustMarshal(ent)
	return w.encoder.encode(b)
}

func (w *Wal) tail() *LockedFile {
	if len(w.locks) > 0 {
		return w.locks[len(w.locks)-1]
	}
	return nil
}

func (w *Wal) sync() error {
	if w.encoder != nil {
		if err := w.encoder.flush(); err != nil {
			return err
		}
	}

	if w.unsafeNoSync {
		return nil
	}

	start := time.Now()
	err := Fdatasync(w.tail().File)

	took := time.Since(start)
	if took > warnSyncDuration {
		w.lg.Warn(
			"slow fdatasync",
			zap.Duration("took", took),
			zap.Duration("expected-duration", warnSyncDuration),
		)
	}
	walFsyncSec.Observe(took.Seconds())

	return err
}

func (w *Wal) cut() error {
	// close old wal file; truncate to avoid wasting space if an early cut
	off, serr := w.tail().Seek(0, io.SeekCurrent)
	if serr != nil {
		return serr
	}

	if err := w.tail().Truncate(off); err != nil {
		return err
	}

	if err := w.sync(); err != nil {
		return err
	}

	fpath := filepath.Join(w.dir, walName(w.seq()+1, w.enti+1))

	// create a temp wal file with name sequence + 1, or truncate the existing one
	newTail, err := w.fp.Open()
	if err != nil {
		return err
	}

	// update writer and save the previous crc
	w.locks = append(w.locks, newTail)

	if err = w.saveState(w.state); err != nil {
		return err
	}

	// atomically move temp wal file to wal file
	if err = w.sync(); err != nil {
		return err
	}

	off, err = w.tail().Seek(0, io.SeekCurrent)
	if err != nil {
		return err
	}

	if err = os.Rename(newTail.Name(), fpath); err != nil {
		return err
	}
	start := time.Now()
	if err = Fsync(w.dirFile); err != nil {
		return err
	}
	walFsyncSec.Observe(time.Since(start).Seconds())

	// reopen newTail with its new path so calls to Name() match the wal filename format
	newTail.Close()

	if newTail, err = LockFile(fpath, os.O_WRONLY, PrivateFileMode); err != nil {
		return err
	}
	if _, err = newTail.Seek(off, io.SeekStart); err != nil {
		return err
	}

	w.locks[len(w.locks)-1] = newTail

	w.lg.Info("created a new WAL segment", zap.String("path", fpath))
	return nil
}

func (w *Wal) seq() uint64 {
	t := w.tail()
	if t == nil {
		return 0
	}
	seq, _, err := parseWALName(filepath.Base(t.Name()))
	if err != nil {
		w.lg.Fatal("failed to parse WAL name", zap.String("name", t.Name()), zap.Error(err))
	}
	return seq
}
