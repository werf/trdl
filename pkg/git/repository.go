package git

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
)

type CloneOptions struct {
	TagName           string
	RecurseSubmodules git.SubmoduleRescursivity
}

func CloneInMemory(url string, opts CloneOptions) (*git.Repository, error) {
	storage := memory.NewStorage()
	fs := memfs.New()

	cloneOptions := &git.CloneOptions{}
	{
		cloneOptions.URL = url

		if opts.TagName != "" {
			cloneOptions.ReferenceName = plumbing.ReferenceName(fmt.Sprintf("refs/tags/%s", opts.TagName))
		}

		if opts.RecurseSubmodules != 0 {
			cloneOptions.RecurseSubmodules = opts.RecurseSubmodules
		}
	}

	return git.Clone(storage, fs, cloneOptions)
}

func ForEachWorktreeFile(gitRepo *git.Repository, fileFunc func(path string, link string, fileReader io.Reader, info os.FileInfo) error) error {
	w, err := gitRepo.Worktree()
	if err != nil {
		return fmt.Errorf("unable to get git repository worktree: %s", err)
	}

	fs := w.Filesystem

	var processFilesFunc func(directory string, files []os.FileInfo) error
	processFilesFunc = func(directory string, fileInfoList []os.FileInfo) error {
		for _, fileInfo := range fileInfoList {
			absPath := path.Join(directory, fileInfo.Name())
			if fileInfo.IsDir() {
				fFileInfoList, err := fs.ReadDir(absPath)
				if err != nil {
					return fmt.Errorf("unable to read dir %q: %s", absPath, err)
				}

				if err := processFilesFunc(absPath, fFileInfoList); err != nil {
					return err
				}

				continue
			}

			if fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink {
				// TODO: support for symlinks
				//link, err := fs.Readlink(absPath)
				//if err != nil {
				//	return fmt.Errorf("unable to read link %q: %s", absPath, err)
				//}
				//
				//if err := fileFunc(absPath, link, nil, fileInfo); err != nil {
				//	return err
				//}

				continue
			} else {
				billyFile, err := fs.Open(absPath)
				if err != nil {
					return fmt.Errorf("unable to open file %q: %s", absPath, err)
				}

				if err := fileFunc(absPath, "", billyFile, fileInfo); err != nil {
					return err
				}

				if err := billyFile.Close(); err != nil {
					return err
				}
			}
		}

		return nil
	}

	rootDirectory := ""
	files, err := fs.ReadDir(rootDirectory)
	if err != nil {
		return fmt.Errorf("unable to read root directory: %s", err)
	}

	return processFilesFunc(rootDirectory, files)
}
