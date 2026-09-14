//go:build !windows

package update

import "errors"

func startCMDConsole(string) error {
	return errors.New("CMD updates require Windows")
}
