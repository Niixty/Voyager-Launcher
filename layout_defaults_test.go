//go:build windows

package main

import (
    "os"
    "path/filepath"
    "testing"

    ll "voyager/native/internal/loadinglayout"
)

func TestGlobalLayoutForExistingAndFuture(t *testing.T) {
    old:=rootDir
    rootDir=t.TempDir()
    defer func(){rootDir=old}()
    if err:=os.MkdirAll(gameDir("Older"),0755);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(configFile("Older"),[]byte("{}"),0600);err!=nil{t.Fatal(err)}
    specific:=ll.Default()
    specific.Find(ll.Title).X=700
    if err:=ll.Save(loadingLayoutFile("Older"),specific);err!=nil{t.Fatal(err)}
    target:=ll.Default()
    target.Find(ll.Title).X=240
    count,err:=applyDefaultLoadingLayout(target)
    if err!=nil{t.Fatal(err)}
    if count!=1{t.Fatalf("expected 1 updated game, got %d",count)}
    existing,err:=loadEffectiveLoadingLayout("Older")
    if err!=nil||existing.Find(ll.Title).X!=240{t.Fatalf("existing: %v %+v",err,existing)}
    future,err:=loadEffectiveLoadingLayout("NewGame")
    if err!=nil||future.Find(ll.Title).X!=240{t.Fatalf("future: %v %+v",err,future)}
    gameFiles,err:=os.ReadDir(gameDir("Older"))
    if err!=nil{t.Fatal(err)}
    backupFound:=false
    for _,f:=range gameFiles{
        if len(f.Name())>19 && f.Name()[:19]=="loading_layout.json" {
            // Old layouts have .before_global_ timestamp suffix.
            if f.Name()!="loading_layout.json"{backupFound=true}
        }
    }
    if !backupFound{t.Fatal("existing layout was not backed up")}
    own:=future.Clone()
    own.Find(ll.Title).X=450
    if err:=ll.Save(loadingLayoutFile("NewGame"),own);err!=nil{t.Fatal(err)}
    check,err:=loadEffectiveLoadingLayout("NewGame")
    if err!=nil||check.Find(ll.Title).X!=450{t.Fatal("custom game layout should win over global default")}
    if _,err=os.Stat(filepath.Join(rootDir,"loading_default.json"));err!=nil{t.Fatal(err)}
}
func TestFallbackWhenNoGlobal(t *testing.T){
    previous:=rootDir
    rootDir=t.TempDir()
    defer func(){rootDir=previous}()
    got,err:=loadEffectiveLoadingLayout("NoGame")
    if err!=nil{t.Fatal(err)}
    if got.Find(ll.Progress)==nil{t.Fatal("factory layout missing")}
}
