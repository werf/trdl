package repo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/werf/trdl/pkg/util"
)

func (c Client) UseChannelReleaseBinDir(group, channel, shell string, asFile bool) error {
	name, data := c.prepareSourceScriptFileNameAndData(group, channel, shell)

	if !asFile {
		fmt.Print(string(data))
		return nil
	}

	sourceScriptPath, err := c.syncSourceScriptFile(group, channel, name, data)
	if err != nil {
		return err
	}

	fmt.Println(sourceScriptPath)
	return nil
}

func (c Client) prepareSourceScriptFileNameAndData(group, channel, shell string) (string, []byte) {
	basename := fmt.Sprintf("use_%s_%s_%s", group, channel, shell)
	logPathFirstBinPath := filepath.Join(c.logsDir, basename+"_first_bin_path.log")
	logPathBackgroundUpdate := filepath.Join(c.logsDir, basename+"_background_update.log")

	commonArgs := []string{c.repoName, group, channel}
	foregroundUpdateArgs := commonArgs[0:]
	backgroundUpdateArgs := append(
		commonArgs[0:],
		"--in-background",
		fmt.Sprintf("--background-output-file=%q", logPathBackgroundUpdate),
	)

	common := strings.Join(commonArgs, " ")               // %[1]s: REPO GROUP CHANNEL
	foreground := strings.Join(foregroundUpdateArgs, " ") // %[2]s: REPO GROUP CHANNEL [flag ...]
	background := strings.Join(backgroundUpdateArgs, " ") // %[3]s: REPO GROUP CHANNEL [flag ...]
	_ = logPathFirstBinPath                               // %[4]s: "*_first_bin_path.log"
	trdlBinaryPath := os.Args[0]                          // %[5]s: trdl binary path

	var content string
	var ext string
	switch shell {
	case "cmd":
		ext, content = cmdSourceScript(common, foreground, background, logPathFirstBinPath, trdlBinaryPath)
	case "pwsh":
		ext, content = pwshSourceScript(common, foreground, background, logPathFirstBinPath, trdlBinaryPath)
	default: // unix shell
		ext, content = unixSourceScript(common, foreground, background, logPathFirstBinPath, trdlBinaryPath)
	}

	name := "source_script"
	if ext != "" {
		name = strings.Join([]string{name, ext}, ".")
	}

	data := []byte(fmt.Sprintln(strings.TrimSpace(content)))

	return name, data
}

func cmdSourceScript(common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, trdlBinaryPath string) (string, string) {
	ext := "bat"
	content := fmt.Sprintf(`
@echo off

%[5]s bin-path %[1]s 1>nul 2>&1
IF %%ERRORLEVEL%% NEQ 0 (
    %[5]s update %[2]s
) ELSE (
    %[5]s update %[3]s
)

FOR /F "tokens=*" %%%%g IN ('%[5]s bin-path %[1]s') do (SET TRDL_REPO_BIN_PATH=%%%%g)
SET PATH=%%PATH%%;%%TRDL_REPO_BIN_PATH%%
`, common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, trdlBinaryPath)

	return ext, content
}

func pwshSourceScript(common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, trdlBinaryPath string) (string, string) {
	filenameExt := "ps1"
	fileContent := fmt.Sprintf(`
if ((Invoke-Expression -Command "%[5]s bin-path %[1]s" 2> $null | Out-String -OutVariable TRDL_REPO_BIN_PATH) -and ($LastExitCode -eq 0)) {
   %[5]s update %[3]s
} else {
   %[5]s update %[2]s
   $TRDL_REPO_BIN_PATH = %[5]s bin-path %[1]s
}

$TRDL_REPO_BIN_PATH = $TRDL_REPO_BIN_PATH.Trim()
$OLDPATH = [System.Environment]::GetEnvironmentVariable('PATH',[System.EnvironmentVariableTarget]::Process)
$NEWPATH = "$OLDPATH;$TRDL_REPO_BIN_PATH"
[System.Environment]::SetEnvironmentVariable('Path',$NEWPATH,[System.EnvironmentVariableTarget]::Process);
`, common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, trdlBinaryPath)

	return filenameExt, fileContent
}

func unixSourceScript(common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, trdlBinaryPath string) (string, string) {
	fileContent := fmt.Sprintf(`
if %[5]q bin-path %[1]s >%[4]q 2>&1; then
   %[5]q update %[3]s
else
   %[5]q update %[2]s
fi

TRDL_REPO_BIN_PATH=$(%[5]q bin-path %[1]s)
export PATH="${PATH:+${PATH}:}$TRDL_REPO_BIN_PATH"
`, common, foregroundUpdate, backgroundUpdate, logPathFirstBinPath, trdlBinaryPath)

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
