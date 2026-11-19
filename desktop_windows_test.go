package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
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

func TestCartridgeImportKeepsAlphaAfterRejectedReplacement(t *testing.T) {
	old := rootDir
	rootDir = t.TempDir()
	defer func() { rootDir = old }()
	if err := initGraphics(); err != nil {
		t.Fatal(err)
	}
	defer stopGraphics()
	s := &Studio{}
	if err := writeConfig("Fixture", gameConfig{Title: "Fixture", Mode: "executavel", ExecutablePath: "fake.exe"}); err != nil {
		t.Fatal(err)
	}
	src := image.NewNRGBA(image.Rect(0, 0, 12, 8))
	for y := 2; y < 6; y++ {
		for x := 3; x < 9; x++ {
			src.SetNRGBA(x, y, color.NRGBA{G: 255, A: 255})
		}
	}
	path := filepath.Join(rootDir, "input.png")
	f, e := os.Create(path)
	if e != nil {
		t.Fatal(e)
	}
	e = png.Encode(f, src)
	f.Close()
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetArt("Fixture", "cartridge", path); e != nil {
		t.Fatal(e)
	}
	dest := filepath.Join(gameDir("Fixture"), "cartridge.png")
	b, e := os.ReadFile(dest)
	if e != nil {
		t.Fatal(e)
	}
	pic, e := png.Decode(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	if pic.Bounds().Dx() != 1536 || pic.Bounds().Dy() != 1024 {
		t.Fatal(pic.Bounds())
	}
	_, _, _, edge := pic.At(0, 0).RGBA()
	_, green, _, center := pic.At(768, 512).RGBA()
	if edge != 0 || center != 65535 || green != 65535 {
		t.Fatalf("alpha edge=%d center=%d green=%d", edge, center, green)
	}
	if e = os.WriteFile(path, []byte("broken"), 0600); e != nil {
		t.Fatal(e)
	}
	if e = s.SetArt("Fixture", "cartridge", path); e == nil {
		t.Fatal("accepted broken image")
	}
	after, e := os.ReadFile(dest)
	if e != nil || !bytes.Equal(b, after) {
		t.Fatal("previous cartridge changed")
	}
}
