package util

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv("TRDL_TEST_EXECUTABLE_HELPER"); mode != "" {
		runExecutableHelper(mode)
		return
	}
	os.Exit(m.Run())
}

func runExecutableHelper(mode string) {
	deadline := time.After(2500 * time.Millisecond)
	ticker := time.NewTicker(20 * time.Millisecond)
	done := false
	for !done {
		select {
		case <-deadline:
			done = true
		case <-ticker.C:
			switch mode {
			case "raw":
				exe, _ := os.Executable()
				real, symErr := filepath.EvalSymlinks(exe)
				if symErr != nil {
					fmt.Println("raw:", exe)
				} else {
					fmt.Println("real:", real)
				}
			case "resolve":
				p, err := ResolveTrdlOnDiskBinaryPath()
				if err != nil {
					fmt.Println("err:", err)
				} else {
					fmt.Println("resolved:", p)
				}
			}
		}
	}
}

func TestStripSelfUpdateBackupSuffix(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/bin/trdl" + SelfUpdateBackupSuffix, "/bin/trdl"},
		{"/opt/werf/trdl" + SelfUpdateBackupSuffix, "/opt/werf/trdl"},
		{"/bin/trdl", "/bin/trdl"},
		{"/bin/.trdl.old", "/bin/.trdl.old"},
		{"/bin/trdl-something", "/bin/trdl-something"},
	}
	for _, c := range cases {
		if got := stripSelfUpdateBackupSuffix(c.in); got != c.want {
			t.Errorf("stripSelfUpdateBackupSuffix(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestReproduce_OsExecutableFollowsSelfUpdateRename(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("os.Executable() follows inode rename only on Linux (/proc/self/exe); macOS/Windows cache the exec path")
	}
	lines := runHelperRaceScenario(t, "raw")
	var seenBackup string
	for _, l := range lines {
		if strings.Contains(l, SelfUpdateBackupSuffix) {
			seenBackup = l
			break
		}
	}
	if seenBackup == "" {
		t.Fatalf("expected os.Executable() to observe the renamed %s path; got:\n%s", SelfUpdateBackupSuffix, strings.Join(lines, "\n"))
	}
	t.Logf("REPRO: os.Executable() returned the backup path during self-update rename: %s", seenBackup)
}

func TestResolveTrdlOnDiskBinaryPath_NeverLeaksBackupSuffix(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("needs Linux /proc/self/exe rename semantics to trigger the race")
	}
	lines := runHelperRaceScenario(t, "resolve")
	for _, l := range lines {
		if !strings.HasPrefix(l, "resolved:") {
			continue
		}
		if strings.Contains(l, SelfUpdateBackupSuffix) {
			t.Fatalf("ResolveTrdlOnDiskBinaryPath leaked the backup path; got: %s\nall lines:\n%s", l, strings.Join(lines, "\n"))
		}
	}
	t.Logf("ResolveTrdlOnDiskBinaryPath consistently returned the stable path across the rename window (%d samples)", len(lines))
}

func runHelperRaceScenario(t *testing.T, mode string) []string {
	t.Helper()

	dir := t.TempDir()
	dst := filepath.Join(dir, "trdl")

	src, err := os.Open(os.Args[0])
	if err != nil {
		t.Fatalf("open self: %v", err)
	}
	defer src.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatalf("create dst: %v", err)
	}
	if _, err := io.Copy(out, src); err != nil {
		t.Fatalf("copy: %v", err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close dst: %v", err)
	}

	cmd := exec.Command(dst)
	cmd.Env = append(os.Environ(), "TRDL_TEST_EXECUTABLE_HELPER="+mode)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}

	var mu sync.Mutex
	var lines []string
	done := make(chan struct{})
	go func() {
		defer close(done)
		s := bufio.NewScanner(stdout)
		for s.Scan() {
			mu.Lock()
			lines = append(lines, s.Text())
			mu.Unlock()
		}
	}()

	// Give the helper time to sample the pre-rename state, then run the
	// same rename dance our doSelfUpdate asks go-update to perform:
	// TargetPath -> TargetPath+SelfUpdateBackupSuffix, swap in a new binary,
	// then unlink the backup ourselves. No external synchronization: this
	// is the actual race we ship into production.
	time.Sleep(200 * time.Millisecond)
	backupPath := dst + SelfUpdateBackupSuffix
	if err := os.Rename(dst, backupPath); err != nil {
		t.Fatalf("step 5 rename: %v", err)
	}
	// Tight gap between the two renames — the exact microsecond window
	// where a naive resolver returns the backup path with no stable sibling.
	time.Sleep(5 * time.Millisecond)
	if err := os.WriteFile(dst, []byte("new binary placeholder"), 0o755); err != nil {
		t.Fatalf("step 6 write: %v", err)
	}
	time.Sleep(50 * time.Millisecond)
	if err := os.Remove(backupPath); err != nil {
		t.Fatalf("step 7 remove: %v", err)
	}

	_ = cmd.Wait()
	<-done

	mu.Lock()
	defer mu.Unlock()
	return append([]string(nil), lines...)
}
