package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("LISTICLES_UPDATE_TEST_HELPER") != "1" {
		os.Exit(m.Run())
	}
	file, err := os.OpenFile(os.Getenv("UPDATE_LOG"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		os.Exit(1)
	}
	args := strings.Join(os.Args[1:], " ")
	fmt.Fprintln(file, args)
	file.Close()
	if os.Getenv("FAIL_GIT") != "" && strings.Contains(args, os.Getenv("FAIL_GIT")) {
		os.Exit(1)
	}
	switch {
	case strings.HasPrefix(args, "symbolic-ref"):
		fmt.Println("main")
	case args == "rev-parse HEAD":
		fmt.Println("abcdef1234")
	case strings.Contains(args, "@{u}"):
		fmt.Println("origin/main")
	}
	os.Exit(0)
}

func TestCMDUpdateExecution(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "repo & space!")
	tools := filepath.Join(root, "tools")
	for _, dir := range []string{repo, tools} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	exe, _ := os.Executable()
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"git.exe", "recorder.exe"} {
		if err := os.WriteFile(filepath.Join(tools, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	log := filepath.Join(root, "log")
	t.Setenv("PATH", tools+";"+filepath.Join(os.Getenv("SystemRoot"), "System32"))
	t.Setenv("LISTICLES_UPDATE_TEST_HELPER", "1")
	t.Setenv("UPDATE_LOG", log)
	for _, tc := range []struct {
		name                         string
		latest, missing, failInstall bool
		failGit                      string
	}{
		{name: "latest", latest: true},
		{name: "rollback"},
		{name: "older revision lacks CMD installer", missing: true},
		{name: "install fails", failInstall: true},
		{name: "fetch fails", latest: true, failGit: "fetch"},
		{name: "checkout fails", failGit: "checkout --detach"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_ = os.Remove(log)
			installer := filepath.Join(repo, "install.cmd")
			_ = os.Remove(installer)
			if !tc.missing {
				code := "0"
				if tc.failInstall {
					code = "1"
				}
				if err := os.WriteFile(installer, []byte("@echo off\r\necho installed>>\"%UPDATE_LOG%\"\r\nexit /b "+code+"\r\n"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("FAIL_GIT", tc.failGit)
			script, err := writeCMDScript(InstallRequest{RepoPath: repo, TargetCommit: "abcdef", Latest: tc.latest, RecorderBinary: filepath.Join(tools, "recorder.exe")})
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(script)
			cmd := exec.Command("cmd.exe", "/d", "/v:off", "/c", script)
			cmd.Stdin = strings.NewReader("\r\n") // dismiss PAUSE
			out, err := cmd.CombinedOutput()
			failed := tc.missing || tc.failInstall || tc.failGit != ""
			if (err != nil) != failed {
				t.Fatalf("err=%v\n%s", err, out)
			}
			calls, _ := os.ReadFile(log)
			recorded := strings.Contains(string(calls), "--record-update")
			if recorded == failed {
				t.Fatalf("metadata recording incorrect:\n%s", calls)
			}
			if !strings.Contains(string(calls), "checkout main") {
				t.Fatalf("original branch not restored:\n%s", calls)
			}
			if !failed && !strings.Contains(string(out), "update complete") {
				t.Fatalf("missing completion: %s", out)
			}
		})
	}
}

func TestCMDConsoleLaunch(t *testing.T) {
	if os.Getenv("LISTICLES_CMD_TESTS") != "1" {
		t.Skip("opt in to opening a Windows console with LISTICLES_CMD_TESTS=1")
	}
	dir := t.TempDir()
	script := filepath.Join(dir, "launch & space.cmd")
	marker := filepath.Join(dir, "started")
	t.Setenv("LISTICLES_LAUNCH_MARKER", marker)
	if err := os.WriteFile(script, []byte("@echo off\r\necho started>\"%LISTICLES_LAUNCH_MARKER%\"\r\nexit\r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := startCMDConsole(script); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(marker); err == nil {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("detached CMD never ran its script")
}
