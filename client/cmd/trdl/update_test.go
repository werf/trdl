package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestStartUpdateInBackgroundDisablesBackgroundMode(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	for _, envValue := range []string{"true", "1", "false", ""} {
		t.Run("TRDL_IN_BACKGROUND="+envValue, func(t *testing.T) {
			t.Setenv("TRDL_IN_BACKGROUND", envValue)
			homeDir := t.TempDir()
			t.Setenv("TRDL_HOME_DIR", homeDir)
			resultPath := filepath.Join(homeDir, "result.json")
			args := []string{"-test.run=^TestBackgroundUpdateHelper$", "--", resultPath}

			if err := StartUpdateInBackground(executable, args, "", ""); err != nil {
				t.Fatal(err)
			}

			timeout := time.NewTimer(10 * time.Second)
			defer timeout.Stop()
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()

			var result []string
		waitForChild:
			for {
				data, err := os.ReadFile(resultPath)
				if err == nil {
					if err := json.Unmarshal(data, &result); err != nil {
						t.Fatal(err)
					}
					break waitForChild
				}
				if !os.IsNotExist(err) {
					t.Fatal(err)
				}
				select {
				case <-timeout.C:
					t.Fatal("background child did not report its environment")
				case <-ticker.C:
				}
			}

			if want := []string{"false", homeDir}; !slices.Equal(result, want) {
				t.Errorf("child [in-background, TRDL_HOME_DIR] = %q, want %q", result, want)
			}
			if got := os.Getenv("TRDL_IN_BACKGROUND"); got != envValue {
				t.Errorf("parent TRDL_IN_BACKGROUND = %q, want %q", got, envValue)
			}
		})
	}
}
