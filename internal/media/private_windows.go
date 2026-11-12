//go:build windows

package media

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"syscall"
)

// hideWindow prevents console windows from flashing when the GUI app runs console tools.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}

func privateDirectory(path string) error {
	u, err := user.Current()
	if err != nil {
		return errors.New("não foi possível identificar o usuário para proteger temporários")
	}
	// Absolute system executable, SID based grant, inheritance removed.
	tool := filepath.Join(os.Getenv("SystemRoot"), "System32", "icacls.exe")
	cmd := exec.Command(tool, path, "/inheritance:r", "/grant:r", "*"+u.Uid+":(OI)(CI)F")
	hideWindow(cmd)
	if err := cmd.Run(); err != nil {
		return errors.New("não foi possível aplicar ACL privada")
	}
	return nil
}
