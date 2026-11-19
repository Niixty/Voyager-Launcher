//go:build windows

package main

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "time"

    ll "voyager/native/internal/loadinglayout"
)

// A single shared layout is the default for both future and existing games.
// A per-game layout still takes priority, allowing later personalization.
func globalLoadingLayoutFile() string {
    return filepath.Join(rootDir,"loading_default.json")
}

func loadEffectiveLoadingLayout(id string)(ll.Layout,error) {
    if validID.MatchString(id) {
        path:=loadingLayoutFile(id)
        if _,err:=os.Stat(path);err==nil {
            return ll.Load(path)
        }else if !errors.Is(err,os.ErrNotExist){
            return ll.Default(),err
        }
    }
    path:=globalLoadingLayoutFile()
    if _,err:=os.Stat(path);err==nil {
        return ll.Load(path)
    }else if !errors.Is(err,os.ErrNotExist){
        return ll.Default(),err
    }
    return ll.Default(),nil
}

// Keep the previous individual file in its game folder so applying a new
// default never permanently destroys an earlier hand-crafted layout.
func backupOldLoadingLayout(path string) error {
    old,err:=os.ReadFile(path)
    if errors.Is(err,os.ErrNotExist){return nil}
    if err!=nil{return err}
    backup:=fmt.Sprintf("%s.before_global_%s.json",path,time.Now().Format("20060102_150405.000000000"))
    return os.WriteFile(backup,old,0600)
}

// Apply the editor's current layout to ALL existing games and as the
// default for future games. Report partial failures instead of claiming
// that every game was updated.
func applyDefaultLoadingLayout(layout ll.Layout)(int,error){
    layout=layout.Normalize()
    if err:=ll.Save(globalLoadingLayoutFile(),layout);err!=nil{
        return 0,fmt.Errorf("não foi possível salvar o layout padrão: %w",err)
    }
    saved:=0
    var problems []error
    for _,id:=range listGames() {
        path:=loadingLayoutFile(id)
        if err:=backupOldLoadingLayout(path);err!=nil{
            problems=append(problems,fmt.Errorf("%s: backup: %w",id,err))
            continue
        }
        if err:=ll.Save(path,layout);err!=nil{
            problems=append(problems,fmt.Errorf("%s: salvar: %w",id,err))
            continue
        }
        saved++
    }
    return saved,errors.Join(problems...)
}
