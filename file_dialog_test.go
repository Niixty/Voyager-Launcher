//go:build windows

package main

import "testing"

func TestFileDialogFilterEmbeddedNUL(t *testing.T) {
	input := "Imagens (*.png;*.jpg)\x00*.png;*.jpg\x00Todos (*.*)\x00*.*\x00"
	encoded := fileDialogFilter(input)
	if len(encoded) < 2 || encoded[len(encoded)-1] != 0 || encoded[len(encoded)-2] != 0 {
		t.Fatal("missing double NUL terminator")
	}
	if encoded[0] != 'I' {t.Fatal("invalid encoding")}
	count := 0
	for _,v:=range encoded {if v==0 {count++}}
	if count!=5 {t.Fatalf("expected 5 NULs, got %d",count)}
}
