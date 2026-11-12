//go:build !test_coverage

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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

			result := readBackgroundUpdateResult(t, resultPath)
			if want := []string{"false", homeDir}; !slices.Equal(result, want) {
				t.Errorf("child [in-background, TRDL_HOME_DIR] = %q, want %q", result, want)
			}
			if got := os.Getenv("TRDL_IN_BACKGROUND"); got != envValue {
				t.Errorf("parent TRDL_IN_BACKGROUND = %q, want %q", got, envValue)
			}
		})
	}
}

func TestUpdateCommandDoesNotRespawnRepeatedly(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		envValue   string
		flags      []string
		generation int
	}{
		{name: "environment true", envValue: "true", generation: 1},
		{name: "environment 1", envValue: "1", generation: 1},
		{name: "bare flag", flags: []string{"--in-background"}, generation: 1},
		{name: "explicit true flag", envValue: "true", flags: []string{"--in-background=true"}, generation: 1},
		{name: "explicit false flag", envValue: "true", flags: []string{"--in-background=false"}, generation: 0},
	}
	for _, entry := range tests {
		t.Run(entry.name, func(t *testing.T) {
			homeDir := t.TempDir()
			resultPath := filepath.Join(homeDir, "result.json")
			args := append([]string{"-test.run=^TestUpdateCommandHelper$", "--", "update", "missing-background-test", "3", "alpha", "--no-self-update"}, entry.flags...)
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, args...)
			cmd.Env = append(os.Environ(),
				"TRDL_HOME_DIR="+homeDir,
				"TRDL_IN_BACKGROUND="+entry.envValue,
				"TRDL_BACKGROUND_STDOUT_FILE=",
				"TRDL_BACKGROUND_STDERR_FILE=",
				"TRDL_TEST_UPDATE_GENERATION=0",
				"TRDL_TEST_UPDATE_RESULT="+resultPath,
			)
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("run update command: %v\n%s", err, output)
			}

			result := readBackgroundUpdateResult(t, resultPath)
			want := []string{strconv.Itoa(entry.generation), `repository "missing-background-test" not initialized: configure it with "trdl add" command`}
			if !slices.Equal(result, want) {
				t.Errorf("update [generation, error] = %q, want %q", result, want)
			}
		})
	}
}
