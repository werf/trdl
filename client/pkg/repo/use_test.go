package repo

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("source_script retry logic", func() {
	var (
		tmpDir    string
		stderrLog string
	)

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		stderrLog = filepath.Join(tmpDir, "stderr.log")
		Expect(os.WriteFile(stderrLog, nil, 0o644)).To(Succeed())
	})

	renderScript := func(binPath string) []byte {
		_, data := renderSourceScript("", sourceScriptParams{
			commonArgs:           "werf 2 stable",
			foregroundUpdateArgs: "werf 2 stable",
			backgroundUpdateArgs: "werf 2 stable --in-background",
			stderrLogPath:        stderrLog,
			trdlBinaryPath:       binPath,
			envName:              "TRDL_USE_WERF_GROUP_CHANNEL",
			envValue:             "2 stable",
		})
		return data
	}

	writeScript := func(data []byte) string {
		p := filepath.Join(tmpDir, "source_script")
		Expect(os.WriteFile(p, data, 0o755)).To(Succeed())
		return p
	}

	readCallCount := func(counterFile string) int {
		data, err := os.ReadFile(counterFile)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		n, err := strconv.Atoi(strings.TrimSpace(string(data)))
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		return n
	}

	writeRestorerWrapper := func(scriptPath, hiddenPath, realPath string) string {
		wrapper := fmt.Sprintf(`#!/bin/sh
sleep() {
  if [ -f %q ]; then
    mv %q %q
  fi
  command sleep "$@"
}
. %q
`, hiddenPath, hiddenPath, realPath, scriptPath)
		wrapperPath := scriptPath + ".wrapper"
		Expect(os.WriteFile(wrapperPath, []byte(wrapper), 0o755)).To(Succeed())
		return wrapperPath
	}

	DescribeTable("retry behavior",
		func(binaryScript string, useWrapper, expectSuccess bool, minCalls, maxCalls int) {
			binPath := filepath.Join(tmpDir, "fake-trdl")
			Expect(os.WriteFile(binPath, []byte(binaryScript), 0o755)).To(Succeed())

			scriptPath := writeScript(renderScript(binPath))

			runPath := scriptPath
			if useWrapper {
				runPath = writeRestorerWrapper(scriptPath, binPath+".hidden", binPath)
			}

			cmd := exec.Command("sh", runPath)
			output, err := cmd.CombinedOutput()
			if expectSuccess {
				Expect(err).NotTo(HaveOccurred(), "script failed: %s", string(output))
			}

			counterFile := binPath + ".count"
			if _, statErr := os.Stat(counterFile); statErr != nil {
				counterFile = filepath.Join(tmpDir, "call-count")
			}
			callCount := readCallCount(counterFile)

			if minCalls > 0 {
				Expect(callCount).To(BeNumerically(">=", minCalls))
			}
			if maxCalls > 0 {
				Expect(callCount).To(BeNumerically("<=", maxCalls))
			}
		},

		Entry("retries when binary disappears during self-update",
			`#!/bin/sh
counter="$0.count"
n=0
if [ -f "$counter" ]; then n=$(cat "$counter"); fi
n=$((n + 1))
echo "$n" > "$counter"
if [ "$n" -le 2 ]; then
  mv "$0" "$0.hidden"
  exit 1
fi
case "$1" in
  bin-path) echo "/dummy/bin" ;;
esac
exit 0
`,
			true, // useWrapper
			true, // expectSuccess
			3,    // minCalls
			0,    // maxCalls (no limit)
		),

		Entry("no retry loop when binary exists but returns error",
			`#!/bin/sh
counter="$0.count"
n=0
if [ -f "$counter" ]; then n=$(cat "$counter"); fi
n=$((n + 1))
echo "$n" > "$counter"
exit 1
`,
			false, // useWrapper
			false, // expectSuccess
			0,     // minCalls (no limit)
			3,     // maxCalls
		),
	)
})
