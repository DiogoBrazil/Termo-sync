package media

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// The application owns this directory. Cleanup never scans unrelated temporary files.
func tempRoot() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	root := filepath.Join(base, "DepoimentoStudio", "private-media")
	if err = os.MkdirAll(root, 0700); err != nil {
		return "", err
	}
	stat, err := os.Lstat(root)
	if err != nil || !stat.IsDir() || stat.Mode()&os.ModeSymlink != 0 {
		return "", errors.New("diretório de temporários inseguro")
	}
	if err = privateDirectory(root); err != nil {
		return "", err
	}
	return root, nil
}

// Recover removes only application-owned session directories older than 24h.
// Fresh leftovers are kept to avoid deleting a concurrently running conversion.
func Recover() error {
	root, err := tempRoot()
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "session-") || !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if time.Since(info.ModTime()) > 24*time.Hour {
			if err = os.RemoveAll(filepath.Join(root, entry.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
