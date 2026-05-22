//go:build windows

package fs

import "golang.org/x/sys/windows"

func logicalDrivesMask() (uint32, error) {
	mask, err := windows.GetLogicalDrives()
	return mask, err
}
