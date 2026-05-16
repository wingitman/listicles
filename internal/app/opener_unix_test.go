//go:build unix

package app

import (
	"os/exec"
	"testing"
)

func TestDetachOpenerProcess_StartsNewSession(t *testing.T) {
	c := exec.Command("true")
	detachOpenerProcess(c)
	if c.SysProcAttr == nil || !c.SysProcAttr.Setsid {
		t.Fatal("opener process should start in a new session")
	}
}
