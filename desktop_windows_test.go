package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStudioProtectsExistingGames(t *testing.T) {
	old := rootDir
	rootDir = t.TempDir()
	defer func() { rootDir = old }()
	os.MkdirAll(gameDir("Sample"), 0755)
	original := []byte(`{"schemaVersion":8,"title":"Sample","mode":"executavel","executablePath":"sample.exe","grid":"custom.jpg","metadata":{"a":true}}`)
	if e := os.WriteFile(configFile("Sample"), original, 0600); e != nil {
		t.Fatal(e)
	}
	s := &Studio{}
	c, e := readConfig("Sample")
	if e != nil {
		t.Fatal(e)
	}
	c.Title = "Edited"
	if s.SaveGame("Sample", c, true) == nil {
		t.Fatal("duplicate creation overwrote existing game")
	}
	if e = s.SaveGame("Sample", c, false); e != nil {
		t.Fatal(e)
	}
	b, _ := os.ReadFile(configFile("Sample"))
	for _, v := range []string{"metadata", "custom.jpg", "Edited"} {
		if !strings.Contains(string(b), v) {
			t.Fatal(string(b))
		}
	}
	if e = s.CopyGame("Sample", "Clone"); e != nil {
		t.Fatal(e)
	}
	if e = s.Archive("Clone"); e != nil {
		t.Fatal(e)
	}
	dirs, e := os.ReadDir(filepath.Join(rootDir, "Games-archive"))
	if e != nil || len(dirs) != 1 {
		t.Fatal(dirs, e)
	}
	os.WriteFile(configFile("Sample"), []byte("broken"), 0600)
	if e = s.SaveGame("Sample", c, false); e == nil {
		t.Fatal("corrupt document overwritten")
	}
}
