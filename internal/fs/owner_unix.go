//go:build !windows

package fs

import (
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// FileOwner returns "user:group" for path on Unix systems.
// On error, "N/A" is returned.
func FileOwner(path string) string {
	info, err := os.Stat(path)
	if err != nil {
		return "N/A"
	}
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "N/A"
	}
	uid := strconv.Itoa(int(st.Uid))
	gid := strconv.Itoa(int(st.Gid))

	userName := uid
	if u, err := user.LookupId(uid); err == nil {
		userName = u.Username
	}
	groupName := gid
	if g, err := user.LookupGroupId(gid); err == nil {
		groupName = g.Name
	}
	return userName + ":" + groupName
}
