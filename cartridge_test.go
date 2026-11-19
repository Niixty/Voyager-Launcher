//go:build windows

package main

import (
 "os"
 "path/filepath"
 "testing"
)

func TestCartridgeConfigAndPath(t *testing.T) {
 old:=rootDir
 rootDir=t.TempDir()
 defer func(){rootDir=old}()
 c:=gameConfig{Title:"Teste",Mode:"retroarch",RetroarchPath:"retroarch.exe",CorePath:"core.dll",RomPath:"jogo.sfc"}
 if err:=writeConfig("Teste",c);err!=nil{t.Fatal(err)}
 got,err:=readConfig("Teste")
 if err!=nil{t.Fatal(err)}
 if got.Cartridge!="cartridge.png" {t.Fatal(got.Cartridge)}
 if p:=artworkPath(got,"Teste","cartridge");p!=""{t.Fatalf("expected no cartridge, got %s",p)}
 path:=filepath.Join(gameDir("Teste"),"cartridge.png")
 if err:=os.WriteFile(path,[]byte("placeholder"),0644);err!=nil{t.Fatal(err)}
 if p:=artworkPath(got,"Teste","cartridge");p!=path{t.Fatal(p)}
}
