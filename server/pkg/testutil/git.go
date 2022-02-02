package testutil

import "strings"

func GetHeadCommit(workTreeDir string) string {
	out := SucceedCommandOutputString(
		workTreeDir,
		"git",
		"rev-parse", "HEAD",
	)

	return strings.TrimSpace(out)
}
