// Package jsonstore commits complete documents in the destination directory.
// A sibling backup keeps the previous bytes, including malformed legacy data.
package jsonstore

import (
	"errors"
	"os"
	"path/filepath"
)

func Write(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	old, err := os.ReadFile(path)
	if err == nil {
		if err = replace(path+".backup", old); err != nil {
			return err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return replace(path, data)
}
func replace(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".document-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closed := f.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	return os.Rename(f.Name(), path)
}
