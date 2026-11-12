package main

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

func TestBackgroundUpdateHelper(t *testing.T) {
	if len(os.Args) < 3 || os.Args[len(os.Args)-2] != "--" {
		return
	}

	inBackground, err := updateCmd().Flags().GetBool("in-background")
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal([]string{strconv.FormatBool(inBackground), os.Getenv("TRDL_HOME_DIR")})
	if err != nil {
		t.Fatal(err)
	}
	resultPath := os.Args[len(os.Args)-1]
	if err := os.WriteFile(resultPath+".tmp", data, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(resultPath+".tmp", resultPath); err != nil {
		t.Fatal(err)
	}
}
