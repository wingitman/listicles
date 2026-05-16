//go:build unix

package app

import (
	"os/exec"
	"syscall"
)

func detachOpenerProcess(c *exec.Cmd) {
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
