//go:build windows

package main

import (
	"os"
	"path/filepath"
	"testing"
	"voyager/native/internal/updatefiles"
)

func TestApplyTransaction(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "VoyagerLauncher.exe")
	stage := filepath.Join(dir, "VoyagerLauncher.next.exe")
	put := func(p, v string) {
		t.Helper()
		if e := os.WriteFile(p, []byte(v), 0600); e != nil {
			t.Fatal(e)
		}
	}
	expect := func(p, v string) {
		t.Helper()
		b, e := os.ReadFile(p)
		if e != nil || string(b) != v {
			t.Fatalf("%s: %q %v", p, b, e)
		}
	}
	put(target, "old")
	put(target+".bak", "older")
	if e := apply(target, stage); e != nil {
		t.Fatal(e)
	}
	expect(target+".bak", "older")
	put(stage, "new")
	put(stage+".sha256", "incorrect")
	if e := apply(target, stage); e == nil {
		t.Fatal("accepted corrupt stage")
	}
	expect(target, "old")
	digest, e := updatefiles.Digest(stage)
	if e != nil {
		t.Fatal(e)
	}
	put(stage+".sha256", digest)
	if e = apply(target, stage); e != nil {
		t.Fatal(e)
	}
	expect(target, "new")
	expect(target+".bak", "old")
	if e = os.Remove(target); e != nil {
		t.Fatal(e)
	}
	if e = apply(target, stage); e != nil {
		t.Fatal(e)
	}
	expect(target, "old")
}
func TestExclusiveLock(t *testing.T) {
	dir := t.TempDir()
	release, e := updatefiles.Lock(dir)
	if e != nil {
		t.Fatal(e)
	}
	if other, e := updatefiles.Lock(dir); e == nil {
		other()
		t.Fatal("second writer acquired lock")
	}
	release()
	release, e = updatefiles.Lock(dir)
	if e != nil {
		t.Fatal(e)
	}
	release()
}
