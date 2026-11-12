package main

import (
	"encoding/json"
	"os"
	"slices"
	"strconv"
	"testing"
	"time"
)

func TestBackgroundUpdateHelper(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "--" {
		return
	}

	inBackground, err := updateCmd().Flags().GetBool("in-background")
	if err != nil {
		t.Fatal(err)
	}
	writeBackgroundUpdateResult(t, os.Args[len(os.Args)-1], []string{strconv.FormatBool(inBackground), os.Getenv("TRDL_HOME_DIR")})
}

func TestUpdateCommandHelper(t *testing.T) {
	separator := slices.Index(os.Args, "--")
	if separator < 0 {
		return
	}

	generation, err := strconv.Atoi(os.Getenv("TRDL_TEST_UPDATE_GENERATION"))
	if err != nil {
		t.Fatal(err)
	}
	resultPath := os.Getenv("TRDL_TEST_UPDATE_RESULT")
	if generation > 1 {
		writeBackgroundUpdateResult(t, resultPath, []string{strconv.Itoa(generation), "unexpected repeated background launch"})
		return
	}
	if err := os.Setenv("TRDL_TEST_UPDATE_GENERATION", strconv.Itoa(generation+1)); err != nil {
		t.Fatal(err)
	}

	cmd := rootCmd()
	cmd.SetArgs(os.Args[separator+1:])
	if err := cmd.Execute(); err != nil {
		writeBackgroundUpdateResult(t, resultPath, []string{strconv.Itoa(generation), err.Error()})
	}
}

func writeBackgroundUpdateResult(t *testing.T, resultPath string, result []string) {
	t.Helper()
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(resultPath+".tmp", data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(resultPath+".tmp", resultPath); err != nil {
		t.Fatal(err)
	}
}

func readBackgroundUpdateResult(t *testing.T, resultPath string) []string {
	t.Helper()
	timeout := time.NewTimer(10 * time.Second)
	defer timeout.Stop()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		data, err := os.ReadFile(resultPath)
		if err == nil {
			var result []string
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatal(err)
			}
			return result
		}
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		select {
		case <-timeout.C:
			t.Fatal("update process did not report its result")
		case <-ticker.C:
		}
	}
}
