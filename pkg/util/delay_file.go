package util

import (
	"os"
	"path/filepath"
	"time"

	"github.com/werf/lockgate"
)

type DelayFile struct {
	Locker   lockgate.Locker
	FilePath string
	Delay    time.Duration
}

func NewDelayFile(locker lockgate.Locker, filePath string, delay time.Duration) DelayFile {
	return DelayFile{
		Locker:   locker,
		FilePath: filePath,
		Delay:    delay,
	}
}

func (u DelayFile) IsDelayPassed() (passed bool, err error) {
	err = lockgate.WithAcquire(u.Locker, u.FilePath, lockgate.AcquireOptions{Shared: true, Timeout: 30 * time.Second}, func(_ bool) error {
		passed, err = u.isDelayPassed()
		return err
	})

	return
}

func (u DelayFile) isDelayPassed() (bool, error) {
	info, err := os.Stat(u.FilePath)
	if err != nil {
		if isNotExistErr(err) {
			return true, nil
		}

		return false, nil
	}

	fTime := info.ModTime()
	if fTime.Add(u.Delay).Before(time.Now()) {
		return true, nil
	}

	return false, nil
}

func (u DelayFile) UpdateTimestamp() error {
	return lockgate.WithAcquire(u.Locker, u.FilePath, lockgate.AcquireOptions{Shared: false, Timeout: 30 * time.Second}, func(_ bool) error {
		return u.updateTimestamp()
	})
}

func (u DelayFile) updateTimestamp() error {
	exist, err := IsRegularFileExist(u.FilePath)
	if err != nil {
		return err
	}

	if exist {
		if err := os.Remove(u.FilePath); err != nil {
			return err
		}
	}

	if err := os.MkdirAll(filepath.Dir(u.FilePath), os.ModePerm); err != nil {
		return err
	}

	f, err := os.Create(u.FilePath)
	if err != nil {
		return err
	}

	if err := f.Close(); err != nil {
		return err
	}

	return nil
}
