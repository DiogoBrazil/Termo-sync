//go:build windows

package media

import (
	"errors"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

func privateDirectory(path string) error {
	u, err := user.Current()
	if err != nil {
		return errors.New("não foi possível identificar o usuário para proteger temporários")
	}
	// Absolute system executable, SID based grant, inheritance removed.
	tool := filepath.Join(os.Getenv("SystemRoot"), "System32", "icacls.exe")
	if err := exec.Command(tool, path, "/inheritance:r", "/grant:r", "*"+u.Uid+":(OI)(CI)F").Run(); err != nil {
		return errors.New("não foi possível aplicar ACL privada")
	}
	return nil
}
