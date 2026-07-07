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
	commonArgs := []string{c.repoName, group, channel}
	basename := c.prepareSourceScriptBasename(fmt.Sprintf("%s_%s", group, channel), shell, opts)
	envName := FormatRepoChannelGroupEnvName(c.repoName)
	envValue := fmt.Sprintf("%s %s", group, channel)

	name, data, err := c.prepareSourceScriptFileNameAndData(commonArgs, basename, shell, envName, envValue, opts)
	if err != nil {
		return "", err
	}
	sourceScriptPath, err := c.syncSourceScriptFile(c.channelScriptsDir(group, channel), c.channelScriptsTmpDir(group, channel), name, data)
	if err != nil {
		return "", err
	}

	return sourceScriptPath, nil
}

func (c Client) UseReleaseBinDir(version, shell string, opts UseSourceOptions) (string, error) {
	commonArgs := []string{c.repoName, fmt.Sprintf("'%s'", version)}
	basename := c.prepareSourceScriptBasename(slugifyConstraint(version), shell, opts)
	envName := FormatRepoVersionEnvName(c.repoName)

	envValue, err := c.prepareVersionExtractor(shell, version)
	if err != nil {
		return "", fmt.Errorf("prepare version extractor: %w", err)
	}

	name, data, err := c.prepareSourceScriptFileNameAndData(commonArgs, basename, shell, envName, envValue, opts)
	if err != nil {
		return "", err
	}
	sourceScriptPath, err := c.syncSourceScriptFile(c.versionScriptsDir(version), c.versionScriptsTmpDir(version), name, data)
	if err != nil {
		return "", err
	}

	return sourceScriptPath, nil
}

type UseSourceOptions struct {
	NoSelfUpdate bool
}

func (c Client) prepareSourceScriptFileNameAndData(commonArgs []string, basename, shell, trdlUseRepoEnvName, trdlUseRepoEnvValue string, opts UseSourceOptions) (string, []byte, error) {
	logPathBackgroundUpdateStdout := filepath.Join(c.logsDir, basename+"_background_update_stdout.log")
	logPathBackgroundUpdateStderr := filepath.Join(c.logsDir, basename+"_background_update_stderr.log")

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

	trdlBinaryPath, err := trdl.GetTrdlBinaryPath()
	if err != nil {
		return "", nil, err
	}

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

[System.Environment]::SetEnvironmentVariable('%[6]s',"%[7]s",[System.EnvironmentVariableTarget]::Process);

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
		commonArgsString,              // %[1]s: REPO GROUP CHANNEL            (common args string)
		foregroundUpdateArgsString,    // %[2]s: REPO GROUP CHANNEL [flag ...] (foreground update args string)
		backgroundUpdateArgsString,    // %[3]s: REPO GROUP CHANNEL [flag ...] (background update args string)
		logPathBackgroundUpdateStderr, // %[4]s: <path>                        (background update error file path)
		trdlBinaryPath,                // %[5]s: <path>                        (trdl binary path)
		trdlUseRepoEnvName,            // %[6]s: <env name>                    (TRDL_USE_<REPO>_GROUP_CHANNEL)
		trdlUseRepoEnvValue,           // %[7]s: <env value>                   (TRDL_USE_<REPO>_GROUP_CHANNEL value)
	)

	name := "source_script"
	if ext != "" {
		name = strings.Join([]string{name, ext}, ".")
	}

	data := []byte(fmt.Sprintln(strings.TrimSpace(script)))

	return name, data, nil
}

func (c Client) prepareSourceScriptBasename(selector, shell string, opts UseSourceOptions) string {
	basename := fmt.Sprintf("use_%s_%s", selector, shell)

	if opts.NoSelfUpdate {
		basename += "_" + util.MurmurHash(fmt.Sprintf("%+v", opts))
	}

	return basename
}

func (c Client) syncSourceScriptFile(scriptsDir, scriptsTmpDir, name string, data []byte) (string, error) {
	scriptPath := filepath.Join(scriptsDir, name)

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

	if err := util.AtomicWriteFile(scriptPath, data, os.ModePerm, scriptsTmpDir); err != nil {
		return "", fmt.Errorf("write source script %q: %w", scriptPath, err)
	}

	return scriptPath, nil
}

func (c Client) prepareVersionExtractor(shell, version string) (string, error) {
	trdlBinaryPath, err := trdl.GetTrdlBinaryPath()
	if err != nil {
		return "", err
	}

	switch shell {
	case "pwsh":
		return fmt.Sprintf("$(((%s dir-path %s '%s') -split '[\\\\/]')[-2])", trdlBinaryPath, c.repoName, version), nil
	default:
		return fmt.Sprintf("$(%q dir-path %s '%s' | awk -F'/' '{print $(NF-1)}')", trdlBinaryPath, c.repoName, version), nil
	}
}

// FormatRepoChannelGroupEnvName returns a formatted repo channel group env name
func FormatRepoChannelGroupEnvName(repoName string) string {
	return fmt.Sprintf("TRDL_USE_%s_GROUP_CHANNEL", formatRepoName(repoName))
}

// FormatRepoVersionEnvName returns a formatted repo version env name
func FormatRepoVersionEnvName(repoName string) string {
	return fmt.Sprintf("TRDL_USE_%s_VERSION", formatRepoName(repoName))
}

// formatRepoName returns a formatted repository name.
// It replaces all non-alphanumeric characters with underscores and converts the result to uppercase.
func formatRepoName(repoName string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	formattedName := re.ReplaceAllString(repoName, "_")
	return strings.ToUpper(formattedName)
}

// slugifyConstraint replaces semver constraint symbols by strings for file paths
func slugifyConstraint(constraint string) string {
	replacer := strings.NewReplacer(
		">=", "gte_",
		"<=", "lte_",
		">", "gt_",
		"<", "lt_",
		"^", "caret_",
		"~", "tilde_",
		"=", "eq_",
		" ", "",
	)

	return replacer.Replace(constraint)
}
