//go:build linux

package fs

// FileBirthTime returns a message indicating that birth time is not reliably
// available on Linux via standard syscalls.
func FileBirthTime(path string) string {
	return "N/A (see modified)"
}
