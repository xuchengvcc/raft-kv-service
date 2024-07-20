package persister

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

const snapSuffix = ".snap"

var (
	crcTable = crc32.MakeTable(crc32.Castagnoli)

	validFiles = map[string]bool{
		"db": true,
	}
)

var (
	ErrNoSnapshot    = errors.New("no snapshot exists")
	ErrEmptySnapshot = errors.New("snapshot file is empty")
)

type Snapshotter struct {
	lg  *zap.Logger
	dir string
}

func NewSnapshotter(lg *zap.Logger, dir string) *Snapshotter {
	if lg == nil {
		lg = zap.NewNop()
	}
	return &Snapshotter{
		lg:  lg,
		dir: dir,
	}
}

func (s *Snapshotter) SaveSnap(snapshot *Snapshot) error {
	if snapshot.IsEmptySnap() {
		return nil
	}
	return s.save(snapshot)
}

func (s *Snapshotter) save(snapshot *Snapshot) error {
	fname := fmt.Sprintf("%016x-%016x%s", snapshot.LastTerm, snapshot.LastTerm, snapSuffix)
	b, err := proto.Marshal(snapshot)
	if err != nil {
		panic(fmt.Sprintf("Marshalling error: %s", err))
	}

	spath := filepath.Join(s.dir, fname)
	err = WriteAndSyncFile(spath, b, 0666)
	if err != nil {
		s.lg.Warn("failed to write a snap file", zap.String("path", spath), zap.Error(err))
		rerr := os.Remove(spath)
		if rerr != nil {
			s.lg.Warn("failed to remove a broken snap file", zap.String("path", spath), zap.Error(rerr))
		}
		return err
	}

	return nil
}

func (s *Snapshotter) Load() (*Snapshot, error) {
	return s.loadMatching(func(*Snapshot) bool { return true })
}

func (s *Snapshotter) loadMatching(matchFn func(*Snapshot) bool) (*Snapshot, error) {
	names, err := s.snapNames()
	if err != nil {
		return nil, err
	}
	var snap *Snapshot
	for _, name := range names {
		if snap, err = s.loadSnap(name); err == nil && matchFn(snap) {
			return snap, nil
		}
	}
	return nil, ErrNoSnapshot
}

func (s *Snapshotter) loadSnap(name string) (*Snapshot, error) {
	fpath := filepath.Join(s.dir, name)
	snap, err := Read(s.lg, fpath)
	if err != nil {
		brokenPath := fpath + ".broken"
		s.lg.Warn("failed to read a snap file", zap.String("path", fpath), zap.Error(err))
		if rerr := os.Rename(fpath, brokenPath); rerr != nil {
			s.lg.Warn("failed to rename a broken snap file", zap.String("path", fpath), zap.String("broken-path", brokenPath), zap.Error(rerr))
		} else {
			s.lg.Warn("renamed to a broken snap file", zap.String("path", fpath), zap.String("broken-path", brokenPath))
		}
	}
	return snap, err
}

func Read(lg *zap.Logger, snapname string) (*Snapshot, error) {
	Assert(lg != nil, "the logger should not be nil")
	b, err := os.ReadFile(snapname)
	if err != nil {
		lg.Warn("failed to read a snap file", zap.String("path", snapname), zap.Error(err))
		return nil, err
	}

	if len(b) == 0 {
		lg.Warn("failed to read empty snapshot file", zap.String("path", snapname))
		return nil, ErrEmptySnapshot
	}

	snap := &Snapshot{}
	if err = proto.Unmarshal(b, snap); err != nil {
		lg.Warn("failed to unmarshal snappb.Snapshot", zap.String("path", snapname), zap.Error(err))
		return nil, err
	}

	return snap, nil
}

func (s *Snapshotter) snapNames() ([]string, error) {
	dir, err := os.Open(s.dir)
	if err != nil {
		return nil, err
	}
	defer dir.Close()
	names, err := dir.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	filenames, err := s.cleanupSnapdir(names)
	if err != nil {
		return nil, err
	}
	snaps := s.checkSuffix(filenames)
	if len(snaps) == 0 {
		return nil, ErrNoSnapshot
	}
	sort.Sort(sort.Reverse(sort.StringSlice(snaps)))
	return snaps, nil
}

func (s *Snapshotter) checkSuffix(names []string) []string {
	var snaps []string
	for i := range names {
		if strings.HasSuffix(names[i], snapSuffix) {
			snaps = append(snaps, names[i])
		} else {
			// If we find a file which is not a snapshot then check if it's
			// a valid file. If not throw out a warning.
			if _, ok := validFiles[names[i]]; !ok {
				s.lg.Warn("found unexpected non-snap file; skipping", zap.String("path", names[i]))
			}
		}
	}
	return snaps
}

func (s *Snapshotter) cleanupSnapdir(filenames []string) (names []string, err error) {
	names = make([]string, 0, len(filenames))
	for _, filename := range filenames {
		if strings.HasPrefix(filename, "db.tmp") {
			s.lg.Info("found orphaned defragmentation file; deleting", zap.String("path", filename))
			if rmErr := os.Remove(filepath.Join(s.dir, filename)); rmErr != nil && !os.IsNotExist(rmErr) {
				return names, fmt.Errorf("failed to remove orphaned .snap.db file %s: %v", filename, rmErr)
			}
		} else {
			names = append(names, filename)
		}
	}
	return names, nil
}
