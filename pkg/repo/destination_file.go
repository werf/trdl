package repo

import "os"

type destinationFile struct {
	*os.File
}

func (t *destinationFile) Delete() error {
	_ = t.Close()
	return os.Remove(t.Name())
}
