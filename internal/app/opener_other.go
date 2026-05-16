//go:build !unix && !windows

package app

import "os/exec"

func detachOpenerProcess(c *exec.Cmd) {
}
