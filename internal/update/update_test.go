package update

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestUsablePowerShell(t *testing.T) {
	for _, tc := range []struct {
		name                string
		failFirst, failBoth bool
		want                string
	}{
		{name: "available", want: "powershell.exe"},
		{name: "missing or denied Windows PowerShell", failFirst: true, want: "pwsh.exe"},
		{name: "both missing or domain blocked", failFirst: true, failBoth: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var probe string
			got := usablePowerShell(func(ctx context.Context, shell string, args ...string) error {
				deadline, ok := ctx.Deadline()
				if !ok || time.Until(deadline) > 5*time.Second {
					t.Fatal("probe must have a bounded timeout")
				}
				if len(args) != 4 || strings.Join(args[:3], " ") != "-NoProfile -NonInteractive -File" {
					t.Fatalf("probe must execute a file without bypassing policy: %v", args)
				}
				probe = args[3]
				data, err := os.ReadFile(probe)
				if err != nil || string(data) != "exit 0\r\n" {
					t.Fatalf("bad probe: %q, %v", data, err)
				}
				if tc.failBoth || (tc.failFirst && shell == "powershell.exe") {
					return errors.New("execution denied")
				}
				return nil
			})
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
			if _, err := os.Stat(probe); !os.IsNotExist(err) {
				t.Fatalf("probe not cleaned up: %v", err)
			}
		})
	}
}

func TestPowerShellProbeTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("exercises real five-second deadline")
	}
	start := time.Now()
	got := usablePowerShell(func(ctx context.Context, shell string, _ ...string) error {
		if shell == "pwsh.exe" {
			return os.ErrNotExist
		}
		<-ctx.Done()
		return ctx.Err()
	})
	if got != "" || time.Since(start) > 8*time.Second {
		t.Fatalf("probe did not time out: %q, %v", got, time.Since(start))
	}
}

func TestCMDScript(t *testing.T) {
	for _, latest := range []bool{false, true} {
		path, err := writeCMDScript(InstallRequest{
			RepoPath: `C:\Work & stuff\100%PATH%!`, TargetCommit: "abcdef",
			RecorderBinary: `C:\User Space\listicles.exe`, Latest: latest,
		})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Remove(path) })
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		script := string(data)
		for _, want := range []string{
			`setlocal EnableExtensions DisableDelayedExpansion`,
			`set "repo=C:\Work & stuff\100%%PATH%%!"`,
			`call install.cmd` + "\r\n" + `if errorlevel 1 goto failed_pop`,
			`git pull --ff-only || goto failed_pop`,
			`git checkout --detach "%target%" || goto failed_pop`,
			`if not exist install.cmd`, `--record-update`, `git checkout "%restore%"`,
		} {
			if !strings.Contains(script, want) {
				t.Errorf("script missing %q", want)
			}
		}
		wantLatest := `set "latest="`
		if latest {
			wantLatest = `set "latest=1"`
		}
		if !strings.Contains(script, wantLatest) {
			t.Errorf("script missing %q", wantLatest)
		}
		if strings.Contains(strings.ToLower(script), "powershell") {
			t.Fatal("CMD fallback must not require PowerShell")
		}
	}
}

func TestCMDRejectsScriptInjection(t *testing.T) {
	for _, value := range []string{"bad\r\necho injected", `bad" & echo injected`, "bad\x00"} {
		for _, req := range []InstallRequest{{RepoPath: value}, {TargetCommit: value}, {RecorderBinary: value}} {
			if _, err := writeCMDScript(req); err == nil {
				t.Errorf("accepted %#v", req)
			}
		}
	}
}
