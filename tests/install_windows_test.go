package installers

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/wingitman/listicles/internal/config"
	"golang.org/x/sys/windows/registry"
)

// A copy of the test executable stands in for Go and the release executable.
// Configuration uses the real implementation; no TUI or Go download is needed.
func TestMain(m *testing.M) {
	if os.Getenv("LISTICLES_TEST_HELPER") != "1" {
		os.Exit(m.Run())
	}
	executable, _ := os.Executable()
	if strings.EqualFold(filepath.Base(executable), "fc.exe") {
		// Wine 11 does not implement FC /B. Windows uses the real system FC.
		if len(os.Args) != 4 {
			os.Exit(2)
		}
		a, errA := os.ReadFile(os.Args[2])
		b, errB := os.ReadFile(os.Args[3])
		if errA != nil || errB != nil || !bytes.Equal(a, b) {
			os.Exit(1)
		}
		os.Exit(0)
	}
	if strings.EqualFold(filepath.Base(executable), "go.exe") {
		if os.Getenv("FAIL_BUILD") == "1" {
			os.Exit(2)
		}
		for i, arg := range os.Args {
			if arg == "-o" && i+1 < len(os.Args) {
				data, err := os.ReadFile(executable)
				if err == nil {
					err = os.WriteFile(os.Args[i+1], data, 0600)
				}
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				os.Exit(0)
			}
		}
		os.Exit(2)
	}
	if len(os.Args) > 1 && os.Args[1] == "--ensure-config" {
		if os.Getenv("FAIL_CONFIG") == "1" {
			os.Exit(3)
		}
		reset := len(os.Args) > 2 && os.Args[2] == "--default"
		if err := config.EnsureConfig(reset); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	if len(os.Args) > 2 && os.Args[1] == "--cd-file" {
		if os.Getenv("CHECK_ARGS") == "1" && (len(os.Args) != 5 || os.Args[3] != "--dir" || os.Args[4] != "somewhere with spaces") {
			os.Exit(8)
		}
		if target := os.Getenv("CD_TARGET"); target != "" {
			if err := os.WriteFile(os.Args[2], []byte(target), 0600); err != nil {
				os.Exit(1)
			}
		}
		if os.Getenv("FAIL_APP") == "1" {
			os.Exit(7)
		}
		os.Exit(0)
	}
	os.Exit(1)
}

func TestCMDInstall(t *testing.T) {
	if os.Getenv("LISTICLES_CMD_TESTS") != "1" {
		t.Skip("set LISTICLES_CMD_TESTS=1 in a disposable Windows account/Wine prefix: tests temporarily edit HKCU user PATH")
	}
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Environment`, registry.ALL_ACCESS)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { key.Close() })
	old, kind, oldErr := key.GetStringValue("Path")
	t.Cleanup(func() {
		if oldErr == registry.ErrNotExist {
			_ = key.DeleteValue("Path")
		} else if kind == registry.EXPAND_SZ {
			_ = key.SetExpandStringValue("Path", old)
		} else {
			_ = key.SetStringValue("Path", old)
		}
	})
	if oldErr != nil && oldErr != registry.ErrNotExist {
		t.Fatal(oldErr)
	}
	root := t.TempDir()
	repo := filepath.Join(root, "source & space!")
	local := filepath.Join(root, "local & space!")
	tools := filepath.Join(root, "tools")
	for _, dir := range []string{repo, local, tools, filepath.Join(repo, "shell"), filepath.Join(repo, "releases", "windows")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	copyFile := func(from, to string) {
		t.Helper()
		data, err := os.ReadFile(from)
		if os.Getenv("LISTICLES_CMD_TRACE") == "1" && strings.HasSuffix(from, ".cmd") {
			data = []byte(strings.ReplaceAll(string(data), "@echo off", "@echo on"))
		}
		if err == nil {
			err = os.WriteFile(to, data, 0600)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"install.cmd", "uninstall.cmd", `shell\l.cmd`, `shell\cmd-path.cmd`} {
		copyFile(filepath.Join("..", name), filepath.Join(repo, name))
	}
	self, _ := os.Executable()
	if os.Getenv("WINEPREFIX") != "" {
		copyFile(self, filepath.Join(tools, "fc.exe"))
	}
	release := filepath.Join(repo, "releases", "windows", "listicles.exe")
	copyFile(self, release)
	t.Setenv("LOCALAPPDATA", local)
	t.Setenv("APPDATA", filepath.Join(root, "config"))
	t.Setenv("PATH", tools+`;`+filepath.Join(os.Getenv("SystemRoot"), "System32"))
	t.Setenv("LISTICLES_TEST_HELPER", "1")
	t.Setenv("PROCESSOR_ARCHITECTURE", "AMD64")
	t.Setenv("PROCESSOR_ARCHITEW6432", "")
	cmdPath := filepath.Join(os.Getenv("SystemRoot"), "System32", "cmd.exe")
	run := func(t *testing.T, command string, fail bool) string {
		t.Helper()
		cmd := exec.Command(cmdPath, "/d", "/v:off", "/c", command)
		cmd.Dir = repo
		out, err := cmd.CombinedOutput()
		if (err != nil) != fail {
			t.Fatalf("%s: err=%v\n%s", command, err, out)
		}
		return string(out)
	}
	if os.Getenv("LISTICLES_CMD_TRACE") == "1" {
		t.Log(run(t, "where fc & set PATH", false))
	}
	basePath := `C:\Other;;"C:\Quoted & path";%USERPROFILE%\bin;C:\Bang!;`
	setPath := func(value string) {
		t.Helper()
		if err := key.SetExpandStringValue("Path", value); err != nil {
			t.Fatal(err)
		}
	}
	getPath := func() string {
		t.Helper()
		value, typ, err := key.GetStringValue("Path")
		if err != nil || typ != registry.EXPAND_SZ {
			t.Fatalf("PATH: %v type=%v", err, typ)
		}
		return value
	}
	setPath(basePath)
	dest := filepath.Join(local, "Programs", "listicles")
	t.Run("release and repeated install preserve PATH and config", func(t *testing.T) {
		run(t, "install.cmd", false)
		want := basePath + ";" + dest
		if got := getPath(); got != want {
			t.Fatalf("PATH got %q want %q", got, want)
		}
		cfgPath := config.ConfigPath()
		data, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatal(err)
		}
		custom := regexp.MustCompile(`show_hidden\s*=\s*false`).ReplaceAllString(string(data), "show_hidden = true")
		if custom == string(data) {
			t.Fatal("could not customize config")
		}
		if err := os.WriteFile(cfgPath, []byte(custom), 0600); err != nil {
			t.Fatal(err)
		}
		run(t, "install.cmd", false)
		if got := getPath(); got != want {
			t.Fatalf("reinstall changed PATH: %q", got)
		}
		cfg, err := config.Load()
		if err != nil || !cfg.Display.ShowHidden {
			t.Fatalf("custom setting lost: %v", err)
		}
		run(t, "install.cmd --default", false)
		cfg, err = config.Load()
		if err != nil || cfg.Display.ShowHidden {
			t.Fatalf("reset failed: %v", err)
		}
	})
	t.Run("source and build failure never fall back", func(t *testing.T) {
		copyFile(self, filepath.Join(tools, "go.exe"))
		run(t, "install.cmd", false)
		before, _ := os.ReadFile(filepath.Join(dest, "listicles.exe"))
		t.Setenv("FAIL_BUILD", "1")
		run(t, "install.cmd", true)
		after, _ := os.ReadFile(filepath.Join(dest, "listicles.exe"))
		if string(before) != string(after) {
			t.Fatal("failed build changed installed binary")
		}
	})
	if err := os.Remove(filepath.Join(tools, "go.exe")); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	t.Run("missing release and config failure", func(t *testing.T) {
		if err := os.Rename(release, release+".save"); err != nil {
			t.Fatal(err)
		}
		run(t, "install.cmd", true)
		if err := os.Rename(release+".save", release); err != nil {
			t.Fatal(err)
		}
		t.Setenv("FAIL_CONFIG", "1")
		run(t, "install.cmd", true)
	})
	t.Run("existing command replacement and conflict warning", func(t *testing.T) {
		custom := filepath.Join(root, "existing tools")
		if err := os.Mkdir(custom, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(custom, "listicles.exe"), []byte("old version"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(tools, "l.cmd"), []byte("@exit /b 0\r\n"), 0600); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", os.Getenv("PATH")+";"+custom)
		out := run(t, "install.cmd", false)
		if !strings.Contains(out, "WARNING: l currently resolves") {
			t.Fatalf("missing conflict warning: %s", out)
		}
		got, err := os.ReadFile(filepath.Join(custom, "listicles.exe"))
		want, _ := os.ReadFile(release)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("existing executable not replaced: %v", err)
		}
		before := getPath()
		run(t, `uninstall.cmd "`+custom+`"`, false)
		if got := getPath(); got != before {
			t.Fatalf("shared PATH entry removed: %q", got)
		}
		_ = os.Remove(filepath.Join(tools, "l.cmd"))
		setPath(basePath + ";" + dest)
	})
	t.Run("wrapper cd and cancellation", func(t *testing.T) {
		target := filepath.Join(root, "chosen & space! %unused%")
		if err := os.Mkdir(target, 0700); err != nil {
			t.Fatal(err)
		}
		t.Setenv("CD_TARGET", target)
		t.Setenv("CHECK_ARGS", "1")
		// Run a driver beside l.cmd to avoid CALL's second expansion of a path.
		driver := "@echo off\r\ncall l.cmd --dir \"somewhere with spaces\"\r\nif errorlevel 1 exit /b %errorlevel%\r\ncd\r\n"
		if err := os.WriteFile(filepath.Join(dest, "driver.cmd"), []byte(driver), 0600); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command(cmdPath, "/d", "/v:off", "/c", "driver.cmd")
		cmd.Dir = dest
		out, err := cmd.CombinedOutput()
		if err != nil || !strings.HasSuffix(strings.TrimSpace(string(out)), target) {
			t.Fatalf("wrapper cd: %v\n%s", err, out)
		}
		t.Run("cross-drive", func(t *testing.T) {
			// Use a disposable mapping rather than assuming a second disk exists.
			if _, err := os.Stat(`Q:\`); !os.IsNotExist(err) {
				t.Skip("Q: already in use")
			}
			if err := exec.Command("subst.exe", "Q:", root).Run(); err != nil {
				t.Skipf("SUBST unavailable: %v", err)
			}
			t.Cleanup(func() { _ = exec.Command("subst.exe", "Q:", "/d").Run() })
			mapped := `Q:\` + filepath.Base(target)
			if _, err := os.Stat(mapped); err != nil {
				t.Skipf("SUBST mapping is not visible: %v", err)
			}
			t.Setenv("CD_TARGET", mapped)
			cmd := exec.Command(cmdPath, "/d", "/v:off", "/c", "driver.cmd")
			cmd.Dir = dest
			out, err := cmd.CombinedOutput()
			if err != nil || !strings.HasSuffix(strings.TrimSpace(string(out)), mapped) {
				t.Fatalf("cross-drive cd: %v\n%s", err, out)
			}
		})
		t.Setenv("CD_TARGET", "")
		cmd = exec.Command(cmdPath, "/d", "/v:off", "/c", "driver.cmd")
		cmd.Dir = dest
		out, err = cmd.CombinedOutput()
		if err != nil || !strings.HasSuffix(strings.TrimSpace(string(out)), dest) {
			t.Fatalf("wrapper cancel: %v\n%s", err, out)
		}
		t.Setenv("CD_TARGET", target)
		t.Setenv("FAIL_APP", "1")
		cmd = exec.Command(cmdPath, "/d", "/v:off", "/c", "driver.cmd")
		cmd.Dir = dest
		if err := cmd.Run(); err == nil {
			t.Fatal("wrapper swallowed application failure")
		}
		_ = os.Remove(filepath.Join(dest, "driver.cmd"))
	})
	t.Run("oversized PATH refuses modification", func(t *testing.T) {
		long := strings.Repeat(`C:\LongPath;`, 1000)
		setPath(long)
		run(t, "install.cmd", true)
		if getPath() != long {
			t.Fatal("long PATH was altered")
		}
		setPath(basePath + ";" + dest)
	})
	t.Run("uninstall preserves unrelated PATH and config", func(t *testing.T) {
		run(t, "uninstall.cmd", false)
		if got := getPath(); got != basePath {
			t.Fatalf("uninstall PATH: %q want %q", got, basePath)
		}
		if _, err := os.Stat(config.ConfigPath()); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dest, "l.cmd")); !os.IsNotExist(err) {
			t.Fatalf("wrapper still exists: %v", err)
		}
	})
}
