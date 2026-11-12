//go:build !windows

package media

import (
	"os"
	"os/exec"
)

func privateDirectory(path string) error { return os.Chmod(path, 0700) }

func hideWindow(*exec.Cmd) {}
