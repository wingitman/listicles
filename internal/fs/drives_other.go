//go:build !windows

package fs

func logicalDrivesMask() (uint32, error) {
	return 0, nil
}
