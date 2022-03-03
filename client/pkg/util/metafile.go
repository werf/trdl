package util

import (
	"os"
	"path/filepath"
	"time"

	"github.com/werf/lockgate"
)

var metafileLockTimeout = 30 * time.Second

type Metafile struct {
	filePath string
}

func NewMetafile(filePath string) Metafile {
	return Metafile{filePath: filePath}
}

func (f Metafile) HasBeenModifiedWithinPeriod(locker lockgate.Locker, period time.Duration) (s bool, err error) {
	err = lockgate.WithAcquire(locker, f.filePath, lockgate.AcquireOptions{Shared: true, Timeout: metafileLockTimeout}, func(_ bool) error {
		s, err = f.hasBeenModifiedWithinPeriod(period)
		return err
	})

	return
}

func (f Metafile) hasBeenModifiedWithinPeriod(period time.Duration) (bool, error) {
	info, err := os.Stat(f.filePath)
	if err != nil {
		return false, nil
	}

	fTime := info.ModTime()
	if fTime.Add(period).After(time.Now()) {
		return true, nil
	}

	return false, nil
}

func (f Metafile) Reset(locker lockgate.Locker) error {
	return lockgate.WithAcquire(locker, f.filePath, lockgate.AcquireOptions{Shared: false, Timeout: metafileLockTimeout}, func(_ bool) error {
		return f.reset()
	})
}

func (f Metafile) reset() error {
	if err := f.delete(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(f.filePath), os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(f.filePath)
	if err != nil {
		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return nil
}

func (f Metafile) Delete(locker lockgate.Locker) error {
	return lockgate.WithAcquire(locker, f.filePath, lockgate.AcquireOptions{Shared: false, Timeout: metafileLockTimeout}, func(_ bool) error {
		return f.delete()
	})
}

func (f Metafile) delete() error {
	exist, err := IsRegularFileExist(f.filePath)
	if err != nil {
		return err
	}

	if exist {
		if err := os.Remove(f.filePath); err != nil {
			return err
		}
	}

	return nil
}
