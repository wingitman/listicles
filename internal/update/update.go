package update

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/wingitman/listicles/internal/config"
)

const defaultRepoURL = "https://github.com/wingitman/listicles.git"

// Commit is one git commit shown in update prompts/history.
type Commit struct {
	Hash    string
	Short   string
	Subject string
	Body    string
	Date    string
}

// Info describes the local source checkout and available upstream commits.
type Info struct {
	RepoPath       string
	Branch         string
	Upstream       string
	CurrentCommit  string
	LatestCommit   string
	Available      []Commit
	History        []Commit
	CheckError     string
	UpdatesEnabled bool
}

// InstallRequest describes the install the detached helper should run.
type InstallRequest struct {
	RepoPath       string
	TargetCommit   string
	Latest         bool
	Terminal       string
	RecorderBinary string
}

// Check ensures a source checkout exists, fetches its origin, and returns the
// commits newer than currentCommit on the checkout's current branch/upstream.
func Check(cfg *config.Config, currentCommit string, historyLimit int) Info {
	info := Info{UpdatesEnabled: cfg == nil || !cfg.Updates.DisableChecks}
	if cfg != nil && cfg.Updates.DisableChecks {
		return info
	}

	repoPath, err := ensureRepoPath(cfg)
	if err != nil {
		info.CheckError = err.Error()
		return info
	}
	info.RepoPath = repoPath

	if out, err := git(repoPath, "fetch", "--prune", "--all"); err != nil {
		info.CheckError = strings.TrimSpace(out)
		if info.CheckError == "" {
			info.CheckError = err.Error()
		}
		return info
	}

	branch, _ := gitTrim(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	info.Branch = branch
	upstream, err := gitTrim(repoPath, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	if err != nil || upstream == "" {
		if branch != "" && branch != "HEAD" {
			upstream = "origin/" + branch
		} else {
			upstream = "origin/HEAD"
		}
	}
	info.Upstream = upstream

	if currentCommit == "" || currentCommit == "dev" {
		if cfg != nil && cfg.Updates.CurrentCommit != "" {
			currentCommit = cfg.Updates.CurrentCommit
		}
	}
	if currentCommit == "" || currentCommit == "dev" {
		currentCommit, _ = gitTrim(repoPath, "rev-parse", "HEAD")
	}
	info.CurrentCommit = currentCommit
	info.LatestCommit, _ = gitTrim(repoPath, "rev-parse", upstream)

	if currentCommit != "" && info.LatestCommit != "" && currentCommit != info.LatestCommit {
		info.Available = gitLog(repoPath, fmt.Sprintf("%s..%s", currentCommit, upstream), historyLimit)
	}
	info.History = gitLog(repoPath, "HEAD", historyLimit)
	return info
}

// LaunchDetached writes an update script and opens it in a separate terminal.
func LaunchDetached(req InstallRequest) error {
	if strings.TrimSpace(req.RepoPath) == "" {
		return errors.New("missing update repo path")
	}
	if runtime.GOOS == "windows" {
		return launchWindows(req)
	}
	return launchUnix(req)
}

func ensureRepoPath(cfg *config.Config) (string, error) {
	if cfg != nil && cfg.Updates.RepoPath != "" && isGitRepo(cfg.Updates.RepoPath) {
		return cfg.Updates.RepoPath, nil
	}
	if cwd, err := os.Getwd(); err == nil && isListiclesRepo(cwd) {
		_ = config.RecordUpdateMetadata("", cwd)
		return cwd, nil
	}
	repoPath := filepath.Join(config.ConfigDir(), "listicles-src")
	if isGitRepo(repoPath) {
		_ = config.RecordUpdateMetadata("", repoPath)
		return repoPath, nil
	}
	if err := os.MkdirAll(filepath.Dir(repoPath), 0755); err != nil {
		return "", err
	}
	cmd := exec.Command("git", "clone", defaultRepoURL, repoPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("clone update repo: %v: %s", err, strings.TrimSpace(string(out)))
	}
	_ = config.RecordUpdateMetadata("", repoPath)
	return repoPath, nil
}

func isListiclesRepo(path string) bool {
	if !isGitRepo(path) {
		return false
	}
	remote, err := gitTrim(path, "remote", "get-url", "origin")
	if err != nil {
		return false
	}
	return strings.Contains(remote, "listicles")
}

func isGitRepo(path string) bool {
	if path == "" {
		return false
	}
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	out, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(out)) == "true"
}

func gitTrim(repoPath string, args ...string) (string, error) {
	out, err := git(repoPath, args...)
	return strings.TrimSpace(out), err
}

func git(repoPath string, args ...string) (string, error) {
	cmdArgs := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", cmdArgs...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func gitLog(repoPath string, rev string, limit int) []Commit {
	if limit < 1 {
		limit = 12
	}
	format := "%H%x1f%h%x1f%s%x1f%b%x1f%ad%x1e"
	args := []string{"log", "--date=short", "--format=" + format, "-n", fmt.Sprint(limit)}
	if rev != "" {
		args = append(args, rev)
	}
	out, err := git(repoPath, args...)
	if err != nil {
		return nil
	}
	records := strings.Split(out, "\x1e")
	commits := make([]Commit, 0, len(records))
	for _, rec := range records {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, "\x1f", 5)
		if len(parts) < 5 {
			continue
		}
		commits = append(commits, Commit{
			Hash:    parts[0],
			Short:   parts[1],
			Subject: parts[2],
			Body:    strings.TrimSpace(parts[3]),
			Date:    parts[4],
		})
	}
	return commits
}

func launchUnix(req InstallRequest) error {
	script, err := writeUnixScript(req)
	if err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("osascript", "-e", fmt.Sprintf(`tell application "Terminal" to do script %q`, script))
		return cmd.Start()
	}
	terminal, args, err := terminalCommand(req.Terminal, script)
	if err != nil {
		return err
	}
	cmd := exec.Command(terminal, args...)
	return cmd.Start()
}

func writeUnixScript(req InstallRequest) (string, error) {
	dir := filepath.Join(os.TempDir(), "listicles-updates")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("update-%d.sh", time.Now().UnixNano()))
	var b bytes.Buffer
	b.WriteString("#!/bin/sh\nset -eu\n")
	b.WriteString("repo=" + shQuote(req.RepoPath) + "\n")
	b.WriteString("target=" + shQuote(req.TargetCommit) + "\n")
	b.WriteString("recorder=" + shQuote(req.RecorderBinary) + "\n")
	b.WriteString("cd \"$repo\"\n")
	b.WriteString("prev_ref=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || printf HEAD)\n")
	b.WriteString("restore_ref=$prev_ref\n")
	b.WriteString("git fetch --prune --all\n")
	if req.Latest {
		b.WriteString("if [ \"$prev_ref\" != HEAD ]; then\n")
		b.WriteString("  if git rev-parse --abbrev-ref --symbolic-full-name '@{u}' >/dev/null 2>&1; then git pull --ff-only; else git merge --ff-only \"origin/$prev_ref\"; fi\n")
		b.WriteString("elif [ -n \"$target\" ]; then git checkout --detach \"$target\"\n")
		b.WriteString("fi\n")
	} else {
		b.WriteString("git checkout --detach \"$target\"\n")
	}
	b.WriteString("make install\n")
	b.WriteString("installed=$(git rev-parse HEAD)\n")
	b.WriteString("if [ -n \"$recorder\" ] && [ -x \"$recorder\" ]; then \"$recorder\" --record-update --update-commit \"$installed\" --update-repo \"$repo\"; fi\n")
	b.WriteString("if [ \"$restore_ref\" != HEAD ]; then git checkout \"$restore_ref\" >/dev/null 2>&1 || true; fi\n")
	b.WriteString("printf '\\nlisticles update complete: %s\\n' \"$installed\"\n")
	b.WriteString("printf 'Press Enter to close...'; read _\n")
	if err := os.WriteFile(path, b.Bytes(), 0755); err != nil {
		return "", err
	}
	return path, nil
}

func launchWindows(req InstallRequest) error {
	terminalName := strings.TrimSuffix(strings.ToLower(filepath.Base(req.Terminal)), ".exe")
	customTerminal := req.Terminal != "" && terminalName != "cmd" && terminalName != "powershell" && terminalName != "pwsh"
	// A binary on PATH may still be denied by AppLocker or domain script policy.
	// Probe an actual file, under normal policy, rather than bypassing policy.
	shell := ""
	if terminalName != "cmd" {
		shell = usablePowerShell(func(ctx context.Context, name string, args ...string) error {
			return exec.CommandContext(ctx, name, args...).Run()
		})
	}
	if shell == "" {
		script, err := writeCMDScript(req)
		if err != nil {
			return err
		}
		if terminalName == "wt" {
			return exec.Command(req.Terminal, "cmd.exe", "/d", "/v:off", "/k", script).Start()
		}
		if customTerminal {
			return exec.Command(req.Terminal, script).Start()
		}
		return startCMDConsole(script)
	}
	script, err := writeWindowsScript(req)
	if err != nil {
		return err
	}
	if terminalName == "wt" {
		return exec.Command(req.Terminal, shell, "-NoProfile", "-NoExit", "-File", script).Start()
	}
	if customTerminal {
		return exec.Command(req.Terminal, script).Start()
	}
	if _, err := exec.LookPath("wt.exe"); err == nil {
		return exec.Command("wt.exe", shell, "-NoProfile", "-NoExit", "-File", script).Start()
	}
	return exec.Command(shell, "-NoProfile", "-NoExit", "-File", script).Start()
}

func usablePowerShell(run func(context.Context, string, ...string) error) string {
	probe, err := os.CreateTemp("", "listicles-probe-*.ps1")
	if err != nil {
		return ""
	}
	defer os.Remove(probe.Name())
	_, writeErr := probe.WriteString("exit 0\r\n")
	closeErr := probe.Close()
	if writeErr != nil || closeErr != nil {
		return ""
	}
	for _, shell := range []string{"powershell.exe", "pwsh.exe"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := run(ctx, shell, "-NoProfile", "-NonInteractive", "-File", probe.Name())
		cancel()
		if err == nil {
			return shell
		}
	}
	return ""
}

// Batch SET values must be literal, single-line, quoted strings. Percent signs
// are doubled in script source; delayed expansion stays disabled throughout.
func cmdValue(value string) (string, error) {
	if strings.ContainsAny(value, "\x00\r\n\"") {
		return "", errors.New("invalid quote or control character in CMD update argument")
	}
	return strings.ReplaceAll(value, "%", "%%"), nil
}

func writeCMDScript(req InstallRequest) (string, error) {
	values := []string{req.RepoPath, req.TargetCommit, req.RecorderBinary}
	for i, value := range values {
		literal, err := cmdValue(value)
		if err != nil {
			return "", err
		}
		values[i] = literal
	}
	latest := ""
	if req.Latest {
		latest = "1"
	}
	content := fmt.Sprintf(`@echo off
setlocal EnableExtensions DisableDelayedExpansion
chcp 65001 >nul
set "repo=%s"
set "target=%s"
set "recorder=%s"
set "latest=%s"
pushd "%%repo%%" || goto failed
set "restore="
for /f "delims=" %%%%R in ('git symbolic-ref --quiet --short HEAD 2^>nul') do set "restore=%%%%R"
if not defined restore for /f "delims=" %%%%R in ('git rev-parse HEAD') do set "restore=%%%%R"
if not defined restore goto failed_pop
git fetch --prune --all || goto failed_pop
if not defined latest goto checkout
git rev-parse --abbrev-ref --symbolic-full-name "@{u}" >nul 2>&1
if errorlevel 1 goto no_upstream
git pull --ff-only || goto failed_pop
goto install
:no_upstream
set "branch="
for /f "delims=" %%%%R in ('git symbolic-ref --quiet --short HEAD 2^>nul') do set "branch=%%%%R"
if not defined branch goto checkout
git merge --ff-only "origin/%%branch%%" || goto failed_pop
goto install
:checkout
if not defined target goto failed_pop
git checkout --detach "%%target%%" || goto failed_pop
:install
if not exist install.cmd (
    echo ERROR: This revision has no install.cmd. CMD installation of older revisions is unsupported.
    goto failed_pop
)
call install.cmd
if errorlevel 1 goto failed_pop
set "installed="
for /f "delims=" %%%%R in ('git rev-parse HEAD') do set "installed=%%%%R"
if not defined installed goto failed_pop
if defined recorder (
    "%%recorder%%" --record-update --update-commit "%%installed%%" --update-repo "%%repo%%"
    if errorlevel 1 goto failed_pop
)
git checkout "%%restore%%" || goto failed_pop
popd
echo listicles update complete: %%installed%%
pause
exit /b 0
:failed_pop
if defined restore git checkout "%%restore%%"
popd
:failed
echo ERROR: Update failed. Review the output above; no automatic retry was attempted.
pause
exit /b 1
`, values[0], values[1], values[2], latest)
	file, err := os.CreateTemp("", "listicles-update-*.cmd")
	if err != nil {
		return "", err
	}
	_, writeErr := file.WriteString(strings.ReplaceAll(content, "\n", "\r\n"))
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		os.Remove(file.Name())
		return "", errors.Join(writeErr, closeErr)
	}
	return file.Name(), nil
}

func writeWindowsScript(req InstallRequest) (string, error) {
	dir := filepath.Join(os.TempDir(), "listicles-updates")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("update-%d.ps1", time.Now().UnixNano()))
	latest := "$false"
	if req.Latest {
		latest = "$true"
	}
	content := fmt.Sprintf(`$ErrorActionPreference = 'Stop'
$repo = %s
$target = %s
$recorder = %s
$latest = %s
Set-Location $repo
$prevRef = (git rev-parse --abbrev-ref HEAD).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Could not determine the original checkout.' }
git fetch --prune --all
if ($LASTEXITCODE -ne 0) { throw 'Git fetch failed.' }
if ($latest) {
    if ($prevRef -ne 'HEAD') {
        git rev-parse --abbrev-ref --symbolic-full-name '@{u}' *> $null
        if ($LASTEXITCODE -eq 0) { git pull --ff-only } else { git merge --ff-only "origin/$prevRef" }
    } elseif ($target) {
        git checkout --detach $target
    }
} else {
    git checkout --detach $target
}
if ($LASTEXITCODE -ne 0) { throw 'Git update failed; installation was not attempted.' }
& .\install.ps1 -Update
if ($LASTEXITCODE -ne 0) { throw 'listicles installation failed; update metadata was not recorded.' }
$installed = (git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Could not determine the installed commit.' }
if ($recorder -and (Test-Path $recorder)) {
    & $recorder --record-update --update-commit $installed --update-repo $repo
    if ($LASTEXITCODE -ne 0) { throw 'Could not record update metadata.' }
}
if ($prevRef -ne 'HEAD') {
    git checkout $prevRef | Out-Null
    if ($LASTEXITCODE -ne 0) { throw 'Could not restore the original checkout.' }
}
Write-Host ""
Write-Host "listicles update complete: $installed" -ForegroundColor Green
Read-Host 'Press Enter to close'
`, psQuote(req.RepoPath), psQuote(req.TargetCommit), psQuote(req.RecorderBinary), latest)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return "", err
	}
	return path, nil
}

func terminalCommand(preferred string, script string) (string, []string, error) {
	if preferred != "" {
		return preferred, []string{"-e", script}, nil
	}
	candidates := []struct {
		name string
		args []string
	}{
		{"x-terminal-emulator", []string{"-e", script}},
		{"gnome-terminal", []string{"--", script}},
		{"konsole", []string{"-e", script}},
		{"xfce4-terminal", []string{"-e", script}},
		{"alacritty", []string{"-e", script}},
		{"kitty", []string{script}},
		{"wezterm", []string{"start", "--", script}},
		{"foot", []string{script}},
	}
	for _, c := range candidates {
		if _, err := exec.LookPath(c.name); err == nil {
			return c.name, c.args, nil
		}
	}
	return "", nil, errors.New("no supported terminal found for detached update")
}

func shQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
