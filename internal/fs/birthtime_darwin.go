//go:build darwin

package fs

import (
	"os"
	"syscall"
	"time"
)

// FileBirthTime returns the birth/creation time of path on macOS.
// Falls back to modification time if the birth time is unavailable.
func FileBirthTime(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "N/A"
	}
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		sec := st.Birthtimespec.Sec
		nsec := int64(st.Birthtimespec.Nsec)
		if sec > 0 {
			return time.Unix(sec, nsec).Format("Jan 02 2006 15:04")
		}
	}
	return info.ModTime().Format("Jan 02 2006 15:04")
}
