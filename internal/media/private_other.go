//go:build !windows

package media

import "os"

func privateDirectory(path string) error { return os.Chmod(path, 0700) }
