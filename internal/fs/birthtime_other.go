//go:build !linux && !darwin

package fs

import "os"

// FileBirthTime returns the modification time as a proxy for birth time on
// platforms where true birth time is not easily accessible (e.g. Windows).
func FileBirthTime(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "N/A"
	}
	return info.ModTime().Format("Jan 02 2006 15:04")
}
