package repo

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/werf/trdl/client/pkg/util"
)

func (c Client) UseChannelReleaseBinDir(group, channel, shell string, opts UseSourceOptions) (string, error) {
	commonArgs := []string{c.repoName, group, channel}
	basename := c.prepareSourceScriptBasename(fmt.Sprintf("%s_%s", group, channel), shell, opts)
	envs := []sourceScriptEnv{
		{Name: FormatRepoChannelGroupEnvName(c.repoName), Value: fmt.Sprintf("%s %s", group, channel)},
		{Name: FormatRepoVersionEnvName(c.repoName), Unset: true},
		{Name: FormatRepoVersionConstraintEnvName(c.repoName), Unset: true},
	}

	name, data, err := c.prepareSourceScriptFileNameAndData(commonArgs, basename, shell, envs, opts)
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

	resolvedVersion, err := c.prepareVersionExtractor(shell, version)
	if err != nil {
		return "", fmt.Errorf("prepare version extractor: %w", err)
	}

	envs := []sourceScriptEnv{
		{Name: FormatRepoVersionEnvName(c.repoName), Value: resolvedVersion, Expression: true},
		{Name: FormatRepoVersionConstraintEnvName(c.repoName), Value: version},
		{Name: FormatRepoChannelGroupEnvName(c.repoName), Unset: true},
	}

	name, data, err := c.prepareSourceScriptFileNameAndData(commonArgs, basename, shell, envs, opts)
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

type sourceScriptEnv struct {
	Name  string
	Value string
	// Expression marks Value as a shell expression (e.g. a command substitution)
	Expression bool
	// Unset removes the variable instead of setting it, clearing stale values
	// left by a previous use of the opposite selection mode.
	Unset bool
}

func (c Client) prepareSourceScriptFileNameAndData(commonArgs []string, basename, shell string, envs []sourceScriptEnv, opts UseSourceOptions) (string, []byte, error) {
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

	trdlBinaryPath, err := util.ResolveTrdlOnDiskBinaryPath()
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

%[6]s
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

%[6]s
export PATH="$trdl_repo_bin_path${PATH:+:${PATH}}"
`
	}

	script := fmt.Sprintf(tmpl,
		commonArgsString,                          // %[1]s: REPO GROUP CHANNEL            (common args string)
		foregroundUpdateArgsString,                // %[2]s: REPO GROUP CHANNEL [flag ...] (foreground update args string)
		backgroundUpdateArgsString,                // %[3]s: REPO GROUP CHANNEL [flag ...] (background update args string)
		logPathBackgroundUpdateStderr,             // %[4]s: <path>                        (background update error file path)
		trdlBinaryPath,                            // %[5]s: <path>                        (trdl binary path)
		formatSourceScriptEnvExports(shell, envs), // %[6]s: <env exports block>
	)

	name := "source_script"
	if ext != "" {
		name = strings.Join([]string{name, ext}, ".")
	}

	data := []byte(fmt.Sprintln(strings.TrimSpace(script)))

	return name, data, nil
}

func formatSourceScriptEnvExports(shell string, envs []sourceScriptEnv) string {
	lines := make([]string, 0, len(envs))
	for _, env := range envs {
		switch shell {
		case "pwsh":
			if env.Unset {
				lines = append(lines, fmt.Sprintf("[System.Environment]::SetEnvironmentVariable('%s',$null,[System.EnvironmentVariableTarget]::Process);", env.Name))
				continue
			}
			var value string
			if env.Expression {
				value = fmt.Sprintf(`"%s"`, env.Value)
			} else {
				value = fmt.Sprintf(`'%s'`, strings.ReplaceAll(env.Value, "'", "''"))
			}
			lines = append(lines, fmt.Sprintf("[System.Environment]::SetEnvironmentVariable('%s',%s,[System.EnvironmentVariableTarget]::Process);", env.Name, value))
		default:
			if env.Unset {
				lines = append(lines, fmt.Sprintf("unset %s", env.Name))
				continue
			}
			var value string
			if env.Expression {
				value = fmt.Sprintf(`"%s"`, env.Value)
			} else {
				value = fmt.Sprintf(`'%s'`, strings.ReplaceAll(env.Value, "'", `'\''`))
			}
			lines = append(lines, fmt.Sprintf("export %s=%s", env.Name, value))
		}
	}

	return strings.Join(lines, "\n")
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
	trdlBinaryPath, err := util.ResolveTrdlOnDiskBinaryPath()
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
// It carries the resolved release name (e.g. "v0.0.2").
func FormatRepoVersionEnvName(repoName string) string {
	return fmt.Sprintf("TRDL_USE_%s_VERSION", formatRepoName(repoName))
}

// FormatRepoVersionConstraintEnvName returns a formatted repo version constraint env name.
// It carries the original version selector requested by the user (e.g. ">=0.0.1").
func FormatRepoVersionConstraintEnvName(repoName string) string {
	return fmt.Sprintf("TRDL_USE_%s_VERSION_CONSTRAINT", formatRepoName(repoName))
}

// formatRepoName returns a formatted repository name.
// It replaces all non-alphanumeric characters with underscores and converts the result to uppercase.
func formatRepoName(repoName string) string {
	re := regexp.MustCompile("[^a-zA-Z0-9_]+")
	formattedName := re.ReplaceAllString(repoName, "_")
	return strings.ToUpper(formattedName)
}

// slugifyConstraint returns a filesystem-safe key for the version constraint.
func slugifyConstraint(constraint string) string {
	sum := sha256.Sum256([]byte(constraint))
	return hex.EncodeToString(sum[:])[:16]
}
