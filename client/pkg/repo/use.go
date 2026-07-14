package repo

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/werf/trdl/client/pkg/trdl"
	"github.com/werf/trdl/client/pkg/util"
)

func (c Client) UseChannelReleaseBinDir(group, channel, shell string, opts UseSourceOptions) (string, error) {
	name, data, err := c.prepareSourceScriptFileNameAndData(group, channel, shell, opts)
	if err != nil {
		return "", err
	}
	sourceScriptPath, err := c.syncSourceScriptFile(group, channel, name, data)
	if err != nil {
		return "", err
	}

	return sourceScriptPath, nil
}

type UseSourceOptions struct {
	NoSelfUpdate bool
}

func (c Client) prepareSourceScriptFileNameAndData(group, channel, shell string, opts UseSourceOptions) (string, []byte, error) {
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
	trdlBinaryPath, err := trdl.GetTrdlBinaryPath()
	if err != nil {
		return "", nil, err
	}

	name, data := renderSourceScript(shell, sourceScriptParams{
		commonArgs:           commonArgsString,
		foregroundUpdateArgs: foregroundUpdateArgsString,
		backgroundUpdateArgs: backgroundUpdateArgsString,
		stderrLogPath:        logPathBackgroundUpdateStderr,
		trdlBinaryPath:       trdlBinaryPath,
		envName:              FormatRepoChannelGroupEnvName(c.repoName),
		envValue:             fmt.Sprintf("%s %s", group, channel),
	})

	return name, data, nil
}

type sourceScriptParams struct {
	commonArgs           string
	foregroundUpdateArgs string
	backgroundUpdateArgs string
	stderrLogPath        string
	trdlBinaryPath       string
	envName              string
	envValue             string
}

func renderSourceScript(shell string, p sourceScriptParams) (string, []byte) {
	var script string
	var name string
	switch shell {
	case "pwsh":
		script = renderPwshSourceScript(p)
		name = "source_script.ps1"
	default:
		script = renderUnixSourceScript(p)
		name = "source_script"
	}

	return name, []byte(fmt.Sprintln(strings.TrimSpace(script)))
}

func renderUnixSourceScript(p sourceScriptParams) string {
	return fmt.Sprintf(`
__trdl_retry() {
   local attempt=0
   while [ "$attempt" -lt 5 ]; do
      if %[5]q "$@" ; then
         return 0
      fi
      if [ -x %[5]q ]; then
         return 1
      fi
      attempt=$((attempt + 1))
      sleep 1
   done
   %[5]q "$@"
}

if [ -s %[4]q ]; then
   echo Previous run of "trdl update" in background generated following errors:
   cat %[4]q
fi

if trdl_repo_bin_path="$(__trdl_retry bin-path %[1]s 2>/dev/null)"; then
   __trdl_retry update %[3]s
else
   __trdl_retry update %[2]s
   trdl_repo_bin_path="$(__trdl_retry bin-path %[1]s)"
fi

export %[6]s="%[7]s"

export PATH="$trdl_repo_bin_path${PATH:+:${PATH}}"
`,
		p.commonArgs,           // %[1]s
		p.foregroundUpdateArgs, // %[2]s
		p.backgroundUpdateArgs, // %[3]s
		p.stderrLogPath,        // %[4]s
		p.trdlBinaryPath,       // %[5]s
		p.envName,              // %[6]s
		p.envValue,             // %[7]s
	)
}

func renderPwshSourceScript(p sourceScriptParams) string {
	return fmt.Sprintf(`
function __trdl_retry {
  $trdlArgs = $args
  for ($attempt = 0; $attempt -lt 5; $attempt++) {
    $output = & %[5]s @trdlArgs 2>&1
    if ($LASTEXITCODE -eq 0) {
      return $output
    }
    if (Test-Path %[5]q -PathType Leaf) {
      throw "trdl failed with exit code $LASTEXITCODE"
    }
    Start-Sleep -Seconds 1
  }
  $output = & %[5]s @trdlArgs
  if ($LASTEXITCODE -ne 0) {
    throw "trdl failed with exit code $LASTEXITCODE"
  }
  return $output
}

if (Test-Path %[4]q -PathType Leaf) {
  $trdlStderrLog = Get-Content %[4]q
  if (!([String]::IsNullOrWhiteSpace($trdlStderrLog))) {
    'Previous run of "trdl update" in background generated following errors:'
    $trdlStderrLog
  }
}

if (($trdlRepoBinPath = (__trdl_retry bin-path %[1]s) 2> $null) -and ($LastExitCode -eq 0)) {
   __trdl_retry update %[3]s
} else {
   __trdl_retry update %[2]s
   $trdlRepoBinPath = __trdl_retry bin-path %[1]s
}

[System.Environment]::SetEnvironmentVariable('%[6]s','%[7]s',[System.EnvironmentVariableTarget]::Process);

$trdlRepoBinPath = $trdlRepoBinPath.Trim()
$oldPath = [System.Environment]::GetEnvironmentVariable('PATH',[System.EnvironmentVariableTarget]::Process)
$newPath = "$trdlRepoBinPath;$oldPath"
[System.Environment]::SetEnvironmentVariable('Path',$newPath,[System.EnvironmentVariableTarget]::Process);
`,
		p.commonArgs,           // %[1]s
		p.foregroundUpdateArgs, // %[2]s
		p.backgroundUpdateArgs, // %[3]s
		p.stderrLogPath,        // %[4]s
		p.trdlBinaryPath,       // %[5]s
		p.envName,              // %[6]s
		p.envValue,             // %[7]s
	)
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

	if err := util.AtomicWriteFile(scriptPath, data, os.ModePerm, c.channelScriptsTmpDir(group, channel)); err != nil {
		return "", fmt.Errorf("write source script %q: %w", scriptPath, err)
	}

	return scriptPath, nil
}

// FormatRepoChannelGroupEnvName returns a formatted repo channel group env name
func FormatRepoChannelGroupEnvName(repoName string) string {
	return fmt.Sprintf("TRDL_USE_%s_GROUP_CHANNEL", formatRepoName(repoName))
}

// formatRepoName returns a formatted repository name.
// It replaces all non-alphanumeric characters with underscores and converts the result to uppercase.
func formatRepoName(repoName string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	formattedName := re.ReplaceAllString(repoName, "_")
	return strings.ToUpper(formattedName)
}
