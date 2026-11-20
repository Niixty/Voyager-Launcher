//go:build windows

package main

import (
    "os"
    "path/filepath"
    "strings"
    "testing"
)

func TestSteamGridAssetURLSafety(t *testing.T) {
    for _,example:=range []struct{url string; allowed bool}{
        {"https://cdn2.steamgriddb.com/hero/sample.png",true},
        {"https://cdn.steamgriddb.com/grid/a.jpg",true},
        {"https://www.steamgriddb.com/asset.png",true},
        {"http://cdn2.steamgriddb.com/grid/a.jpg",false},
        {"https://steamgriddb.com.evil.example/grid.png",false},
        {"https://cdn2.steamgriddb.com:444/grid.png",false},
        {"https://user:password@cdn2.steamgriddb.com/grid.png",false},
        {"file:///C:/Windows/System32/drivers/etc/hosts",false},
        {"https://127.0.0.1/test.png",false},
        {"",false},
    } {
        if got:=steamGridURLAllowed(example.url);got!=example.allowed {
            t.Fatalf("validacao URL %q: obtido %v, esperado %v",example.url,got,example.allowed)
        }
    }
}

func TestSteamGridRequiresSavedTitle(t *testing.T) {
    previous:=rootDir
    rootDir=t.TempDir()
    defer func(){rootDir=previous}()
    id:="NoTitle"
    if err:=os.MkdirAll(gameDir(id),0755);err!=nil{t.Fatal(err)}
    path:=configFile(id)
    if err:=os.WriteFile(path,[]byte(`{"mode":"executavel","executablePath":"game.exe"}`),0600);err!=nil{t.Fatal(err)}
    _,err:=steamGridSavedTitle(id)
    if err==nil || !strings.Contains(err.Error(),"nome do jogo") {t.Fatalf("nome vazio não detectado: %v",err)}
    if err:=os.WriteFile(path,[]byte(`{"title":"Super Mario World"}`),0600);err!=nil{t.Fatal(err)}
    name,err:=steamGridSavedTitle(id)
    if err!=nil || name!="Super Mario World" {t.Fatalf("nome salvo inválido: %q / %v",name,err)}
    if filepath.Base(path)!="config.json" {t.Fatal(path)}
}

func TestSteamGridDPAPIRoundtrip(t *testing.T) {
    raw:=[]byte("steamgriddb-example-api-key-123456")
    encrypted,err:=sgdbCipher(raw,true)
    if err!=nil{t.Fatal(err)}
    if string(encrypted)==string(raw){t.Fatal("a chave foi salva sem criptografia")}
    decrypted,err:=sgdbCipher(encrypted,false)
    if err!=nil{t.Fatal(err)}
    if string(decrypted)!=string(raw){t.Fatal("a chave DPAPI não foi restaurada")}
}

func TestSteamGridRejectInvalidCollection(t *testing.T) {
    s:=&Studio{}
    for _,kind:=range []string{"cartridge","logo","","../../file"} {
        if _,err:=s.SteamGridImages(1,kind,0);err==nil{t.Fatalf("tipo %q aceito",kind)}
    }
    if _,err:=s.SteamGridImages(1,"hero",-1);err==nil{t.Fatal("pagina negativa aceita")}
}
