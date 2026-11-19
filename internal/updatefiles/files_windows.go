package updatefiles

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/windows"
)

// Windows releases the exclusive file handle even if either process crashes.
func Lock(dir string) (func(), error) {
	p, e := windows.UTF16PtrFromString(filepath.Join(dir, ".voyager-update.lock"))
	if e != nil {
		return nil, e
	}
	h, e := windows.CreateFile(p, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_ALWAYS, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if e != nil {
		return nil, fmt.Errorf("atualização em uso ou pasta inacessível: %w", e)
	}
	return func() { windows.CloseHandle(h) }, nil
}
func Digest(name string) (string, error) {
	f, e := os.Open(name)
	if e != nil {
		return "", e
	}
	defer f.Close()
	h := sha256.New()
	_, e = io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil)), e
}
func Backup(src, dest string) error {
	f, e := os.Open(src)
	if e != nil {
		return e
	}
	defer f.Close()
	tmp, e := os.CreateTemp(filepath.Dir(dest), ".backup-*.tmp")
	if e != nil {
		return e
	}
	defer os.Remove(tmp.Name())
	_, e = io.Copy(tmp, f)
	if e == nil {
		e = tmp.Sync()
	}
	closeErr := tmp.Close()
	if e != nil {
		return e
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(tmp.Name(), dest)
}
