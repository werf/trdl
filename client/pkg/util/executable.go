package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SelfUpdateBackupSuffix is appended to the running trdl binary's path while
// go-update swaps in a new version (see pkg/client.doSelfUpdate). It is our
// own convention rather than go-update's default ".<name>.old", so that the
// resolver below strips a suffix we control instead of matching against
// go-update's internals.
const SelfUpdateBackupSuffix = ".selfupdate-backup"

// ResolveTrdlOnDiskBinaryPath returns an absolute path to a trdl binary that
// is (or is about to be) sitting on disk under a stable, non-transient name,
// suitable for two uses:
//
//   - embedding into the generated source_script, so that later shells re-
//     sourcing the file can exec that path;
//   - forking the "trdl update" background process from "trdl use", so the
//     child exec starts from a filename that will still exist.
//
// On Linux /proc/self/exe (and therefore os.Executable) follows the running
// binary's inode across rename(2). While one trdl process is inside
// inconshreveable/go-update apply.go it renames "trdl" -> the OldSavePath we
// pass in ("trdl"+SelfUpdateBackupSuffix), swaps in the new binary as
// "trdl", and we then unlink the backup. A concurrent trdl process that
// reads os.Executable across that window observes the backup path, and —
// unfixed — embeds it into the source_script written by "trdl use". Later
// shells sourcing that file hit an about-to-be-unlinked path with
// "No such file or directory".
//
// This helper resolves os.Executable freshly (no init-time cache, which
// used to fossilize a bad backup path for the whole life of any process
// that observed it once) and, if the path ends in SelfUpdateBackupSuffix,
// returns the stable sibling without the suffix. By the time a downstream
// shell exec's the embedded path the swap is long since done — go-update
// never lingers in the microsecond window between its two rename(2) calls.
//
// Note the returned path may point at a *different* binary than the
// caller's own inode after self-update. That is intentional: we want the
// path a subsequent shell exec will resolve to, which is the new trdl on
// disk, not our ephemeral old inode.
func ResolveTrdlOnDiskBinaryPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("unable to determine trdl binary path: %w", err)
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return stripSelfUpdateBackupSuffix(exe), nil
}

func stripSelfUpdateBackupSuffix(p string) string {
	if !strings.HasSuffix(p, SelfUpdateBackupSuffix) {
		return p
	}
	return strings.TrimSuffix(p, SelfUpdateBackupSuffix)
}
