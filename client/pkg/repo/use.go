package repo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/werf/trdl/client/pkg/util"
)

func (c Client) UseChannelReleaseBinDir(group, channel, shell string, opts UseSourceOptions) (string, error) {
	name, data := c.prepareSourceScriptFileNameAndData(group, channel, shell, opts)
	sourceScriptPath, err := c.syncSourceScriptFile(group, channel, name, data)
	if err != nil {
		return "", err
	}

	return sourceScriptPath, nil
}

type UseSourceOptions struct {
	NoSelfUpdate bool
}

func (c Client) prepareSourceScriptFileNameAndData(group, channel, shell string, opts UseSourceOptions) (string, []byte) {
	basename := c.prepareSourceScriptBasename(group, channel, shell, opts)
	logPathFirstBinPath := filepath.Join(c.logsDir, basename+"_first_bin_path.log")
	logPathBackgroundUpdateStdout := filepath.Join(c.logsDir, basename+"_background_update_stdout.log")
	logPathBackgroundUpdateStderr := filepath.Join(c.logsDir, basename+"_background_update_stderr.log")

	commonArgs := []string{c.repoName, group, channel}
	foregroundUpdateArgs := commonArgs[0:]
	backgroundUpdateArgs := append(
		commonArgs[0:],
		"--in-background",
		fmt.Sprintf("--background-stdout-file=%q", logPathBackgroundUpdateStdout),
		fmt.Sprintf("--background-stderr-file=%q", logPathBackgroundUpdateStderr),
	)

	if opts.NoSelfUpdate {
		foregroundUpdateArgs = append(foregroundUpdateArgs, "--no-self-update")
		backgroundUpdateArgs = append(backgroundUpdateArgs, "--no-self-update")
	}

	common := strings.Join(commonArgs, " ")               // %[1]s: REPO GROUP CHANNEL
	foreground := strings.Join(foregroundUpdateArgs, " ") // %[2]s: REPO GROUP CHANNEL [flag ...]
	background := strings.Join(backgroundUpdateArgs, " ") // %[3]s: REPO GROUP CHANNEL [flag ...]
	_ = logPathFirstBinPath                               // %[4]s: "*_first_bin_path.log"
	_ = logPathBackgroundUpdateStderr                     // %[5]s: "*_background_update_stderr.log"
	trdlBinaryPath := os.Args[0]                          // %[6]s: trdl binary path

	var content string
	var ext string
	switch shell {
	case "pwsh":
		ext, content = pwshSourceScript(common, foreground, background, logPathFirstBinPath, logPathBackgroundUpdateStderr, trdlBinaryPath)
	default: // unix shell
		ext, content = unixSourceScript(common, foreground, background, logPathFirstBinPath, logPathBackgroundUpdateStderr, trdlBinaryPath)
	}

	name := "source_script"
	if ext != "" {
		name = strings.Join([]string{name, ext}, ".")
	}

	data := []byte(fmt.Sprintln(strings.TrimSpace(content)))

	return name, data
}

func (c Client) prepareSourceScriptBasename(group, channel, shell string, opts UseSourceOptions) string {
	basename := fmt.Sprintf("use_%s_%s_%s", group, channel, shell)

	if opts.NoSelfUpdate {
		basename += "_" + util.MurmurHash(fmt.Sprintf("%+v", opts))
	}

	return basename
}

func pwshSourceScript(common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, logPathBackgroundUpdateStderr, trdlBinaryPath string) (string, string) {
	filenameExt := "ps1"
	fileContent := fmt.Sprintf(`
$ErrorActionPreference = "Stop"

if (Test-Path %[5]q -PathType Leaf) {
  $trdlStderrLog = Get-Content %[5]q
  if (!([String]::IsNullOrWhiteSpace($trdlStderrLog))) {
    'Previous run of "trdl update" in background generated following errors:'
    $trdlStderrLog
  }
}

if ((Invoke-Expression -Command "%[6]s bin-path %[1]s" 2> $null | Out-String -OutVariable TRDL_REPO_BIN_PATH) -and ($LastExitCode -eq 0)) {
   %[6]s update %[3]s
} else {
   %[6]s update %[2]s
   $TRDL_REPO_BIN_PATH = %[6]s bin-path %[1]s
}

$TRDL_REPO_BIN_PATH = $TRDL_REPO_BIN_PATH.Trim()
$OLDPATH = [System.Environment]::GetEnvironmentVariable('PATH',[System.EnvironmentVariableTarget]::Process)
$NEWPATH = "$TRDL_REPO_BIN_PATH;$OLDPATH"
[System.Environment]::SetEnvironmentVariable('Path',$NEWPATH,[System.EnvironmentVariableTarget]::Process);
`, common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, logPathBackgroundUpdateStderr, trdlBinaryPath)

	return filenameExt, fileContent
}

func unixSourceScript(common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, logPathBackgroundUpdateStderr, trdlBinaryPath string) (string, string) {
	fileContent := fmt.Sprintf(`
if [ -s %[5]q ]; then
   echo Previous run of "trdl update" in background generated following errors:
   cat %[5]q
fi

if %[6]q bin-path %[1]s >%[4]q 2>&1; then
   %[6]q update %[3]s
else
   %[6]q update %[2]s
fi

TRDL_REPO_BIN_PATH=$(%[6]q bin-path %[1]s)
export PATH="$TRDL_REPO_BIN_PATH${PATH:+:${PATH}}"
`, common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, logPathBackgroundUpdateStderr, trdlBinaryPath)

	return "", fileContent
}

func (c Client) syncSourceScriptFile(group string, channel string, name string, data []byte) (string, error) {
	scriptPath := filepath.Join(c.channelScriptsDir(group, channel), name)
	scriptTmpPath := filepath.Join(c.channelScriptsTmpDir(group, channel), name)

	exist, err := util.IsRegularFileExist(scriptPath)
	if err != nil {
		return "", fmt.Errorf("unable to check existence of file %q: %s", scriptPath, err)
	}

	if exist {
		currentData, err := ioutil.ReadFile(scriptPath)
		if err != nil {
			return "", fmt.Errorf("unable to read file %q: %s", scriptPath, err)
		}

		if bytes.Equal(currentData, data) {
			return scriptPath, nil
		}
	}

	// create tmp file
	{
		if err := os.MkdirAll(filepath.Dir(scriptTmpPath), 0o755); err != nil {
			return "", err
		}

		if err := ioutil.WriteFile(scriptTmpPath, data, os.ModePerm); err != nil {
			return "", err
		}
	}

	// rename file
	{
		if err := os.MkdirAll(filepath.Dir(scriptPath), 0o755); err != nil {
			return "", err
		}

		if err := os.Rename(scriptTmpPath, scriptPath); err != nil {
			return "", err
		}
	}

	return scriptPath, nil
}
