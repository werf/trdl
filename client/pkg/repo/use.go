package repo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
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
	logPathBackgroundUpdateStdout := filepath.Join(c.logsDir, basename+"_background_update_stdout.log")
	logPathBackgroundUpdateStderr := filepath.Join(c.logsDir, basename+"_background_update_stderr.log")

	commonArgs := []string{c.repoName, group, channel}
	foregroundUpdateArgs := commonArgs[0:]
	backgroundUpdateArgs := append(
		append([]string{}, commonArgs[0:]...),
		"--in-background",
		fmt.Sprintf("--background-stdout-file=%q", logPathBackgroundUpdateStdout),
		fmt.Sprintf("--background-stderr-file=%q", logPathBackgroundUpdateStderr),
	)

	if opts.NoSelfUpdate {
		foregroundUpdateArgs = append(foregroundUpdateArgs, "--no-self-update")
		backgroundUpdateArgs = append(backgroundUpdateArgs, "--no-self-update")
	}

	commonArgsString := strings.Join(commonArgs, " ")
	foregroundUpdateArgsString := strings.Join(foregroundUpdateArgs, " ")
	backgroundUpdateArgsString := strings.Join(backgroundUpdateArgs, " ")
	_ = logPathBackgroundUpdateStderr
	trdlBinaryPath := os.Args[0]
	trdlUseRepoGroupChannelEnvName := fmt.Sprintf("TRDL_USE_%s_GROUP_CHANNEL", formatRepoName(c.repoName))
	trdlUseRepoGroupChannelEnvValue := fmt.Sprintf("%s %s", group, channel)

	var tmpl string
	var ext string
	switch shell {
	case "pwsh":
		ext = "ps1"
		tmpl = `
if (Test-Path %[4]q -PathType Leaf) {
  $trdlStderrLog = Get-Content %[4]q
  if (!([String]::IsNullOrWhiteSpace($trdlStderrLog))) {
    'Previous run of "trdl update" in background generated following errors:'
    $trdlStderrLog
  }
}

if ((Invoke-Expression -Command "%[5]s bin-path %[1]s" 2> $null | Out-String -OutVariable trdlRepoBinPath) -and ($LastExitCode -eq 0)) {
   %[5]s update %[3]s
} else {
   %[5]s update %[2]s
   $trdlRepoBinPath = %[5]s bin-path %[1]s
}

[System.Environment]::SetEnvironmentVariable('%[6]s','%[7]s',[System.EnvironmentVariableTarget]::Process);

$trdlRepoBinPath = $trdlRepoBinPath.Trim()
$oldPath = [System.Environment]::GetEnvironmentVariable('PATH',[System.EnvironmentVariableTarget]::Process)
$newPath = "$trdlRepoBinPath;$oldPath"
[System.Environment]::SetEnvironmentVariable('Path',$newPath,[System.EnvironmentVariableTarget]::Process);
`
	default: // unix shell
		ext = ""
		tmpl = `
if [ -s %[4]q ]; then
   echo Previous run of "trdl update" in background generated following errors:
   cat %[4]q
fi

if trdl_repo_bin_path="$(%[5]q bin-path %[1]s 2>/dev/null)"; then
   %[5]q update %[3]s
else
   %[5]q update %[2]s
   trdl_repo_bin_path="$(%[5]q bin-path %[1]s)"
fi

export %[6]s="%[7]s"

export PATH="$trdl_repo_bin_path${PATH:+:${PATH}}"
`
	}

	script := fmt.Sprintf(tmpl,
		commonArgsString,                // %[1]s: REPO GROUP CHANNEL            (common args string)
		foregroundUpdateArgsString,      // %[2]s: REPO GROUP CHANNEL [flag ...] (foreground update args string)
		backgroundUpdateArgsString,      // %[3]s: REPO GROUP CHANNEL [flag ...] (background update args string)
		logPathBackgroundUpdateStderr,   // %[4]s: <path>                        (background update error file path)
		trdlBinaryPath,                  // %[5]s: <path>                        (trdl binary path)
		trdlUseRepoGroupChannelEnvName,  // %[6]s: <env name>                    (TRDL_USE_<REPO>_GROUP_CHANNEL)
		trdlUseRepoGroupChannelEnvValue, // %[7]s: <env value>                   (TRDL_USE_<REPO>_GROUP_CHANNEL value)
	)

	name := "source_script"
	if ext != "" {
		name = strings.Join([]string{name, ext}, ".")
	}

	data := []byte(fmt.Sprintln(strings.TrimSpace(script)))

	return name, data
}

func (c Client) prepareSourceScriptBasename(group, channel, shell string, opts UseSourceOptions) string {
	basename := fmt.Sprintf("use_%s_%s_%s", group, channel, shell)

	if opts.NoSelfUpdate {
		basename += "_" + util.MurmurHash(fmt.Sprintf("%+v", opts))
	}

	return basename
}

func (c Client) syncSourceScriptFile(group, channel, name string, data []byte) (string, error) {
	scriptPath := filepath.Join(c.channelScriptsDir(group, channel), name)
	scriptTmpPath := filepath.Join(c.channelScriptsTmpDir(group, channel), name)

	exist, err := util.IsRegularFileExist(scriptPath)
	if err != nil {
		return "", fmt.Errorf("unable to check existence of file %q: %w", scriptPath, err)
	}

	if exist {
		currentData, err := ioutil.ReadFile(scriptPath)
		if err != nil {
			return "", fmt.Errorf("unable to read file %q: %w", scriptPath, err)
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

// formatRepoName returns a formatted repository name.
// It replaces all non-alphanumeric characters with underscores and converts the result to uppercase.
func formatRepoName(repoName string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	formattedName := re.ReplaceAllString(repoName, "_")
	return strings.ToUpper(formattedName)
}
