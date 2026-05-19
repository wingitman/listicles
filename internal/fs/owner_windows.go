//go:build windows

package fs

// FileOwner returns "N/A" on Windows.
func FileOwner(path string) string {
	return "N/A"
}
