package tuf

import (
	tufClient "github.com/theupdateframework/go-tuf/client"
	"github.com/theupdateframework/go-tuf/data"
)

type Client interface {
	Init(rootKeys []*data.Key, threshold int) error
	Targets() (data.TargetFiles, error)
	Download(string, tufClient.Destination) error
	DownloadMetaUnsafe(basename string, maxSize int64) ([]byte, error)
	Update() error
}
