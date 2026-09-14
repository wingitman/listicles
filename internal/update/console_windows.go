package update

import (
	"os"
	"os/exec"
	"syscall"
)

func startCMDConsole(script string) error {
	cmd := exec.Command("cmd.exe")
	// Pass the path through the environment so percent signs in usernames are
	// not expanded a second time. A new console also works without wt.exe.
	cmd.Env = append(os.Environ(), "LISTICLES_UPDATE_SCRIPT="+script)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000010, // CREATE_NEW_CONSOLE
		CmdLine:       `cmd.exe /d /v:off /k ""%LISTICLES_UPDATE_SCRIPT%""`,
	}
	return cmd.Start()
}
