package repo

import (
	"os"

	"github.com/theupdateframework/go-tuf/data"
)

type TufInterface interface {
	Setup(rootVersion int64, rootSha512 string) error
	Update() error
	DownloadFile(targetName, dest string, destMode os.FileMode) error
	GetTargets() (data.TargetFiles, error)
}
