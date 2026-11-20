//go:build windows

package main

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
)

// Keep the external updater in sync with the manager. Older versions of
// VoyagerUpdater.exe do not understand --restart, so installing only the
// launcher would break automatic reopen on the next update.
func downloadUpdatedHelper(ctx context.Context,release githubRelease) error {
    var found bool
    for _,asset:=range release.Assets {
        if asset.Name!="VoyagerUpdater.exe" {continue}
        found=true
        if asset.Size<50000 || asset.Size>50*1024*1024 {
            return errors.New("tamanho do atualizador fora dos limites")
        }
        if !strings.HasPrefix(asset.Digest,"sha256:") || len(asset.Digest)!=71 {
            return errors.New("atualizador sem SHA-256 verificável")
        }
        expected:=strings.TrimPrefix(asset.Digest,"sha256:")
        if _,err:=hex.DecodeString(expected);err!=nil{return err}
        if !strings.HasPrefix(asset.DownloadURL,"https://github.com/Niixty/Voyager-Launcher/releases/download/"){
            return errors.New("URL do atualizador inválida")
        }
        req,err:=http.NewRequestWithContext(ctx,"GET",asset.DownloadURL,nil)
        if err!=nil{return err}
        req.Header.Set("User-Agent","VoyagerLauncher/"+version)
        res,err:=updateHTTPClient().Do(req)
        if err!=nil{return fmt.Errorf("não foi possível baixar o atualizador: %w",err)}
        if res.StatusCode!=http.StatusOK {
            res.Body.Close()
            return fmt.Errorf("download do atualizador retornou HTTP %d",res.StatusCode)
        }
        file,err:=os.CreateTemp(rootDir,".voyager-helper-*.tmp")
        if err!=nil{res.Body.Close();return err}
        temp:=file.Name()
        defer os.Remove(temp)
        hash:=sha256.New()
        count,copyErr:=io.CopyN(io.MultiWriter(file,hash),res.Body,asset.Size+1)
        closeErr:=file.Close()
        res.Body.Close()
        if copyErr!=nil && copyErr!=io.EOF{return copyErr}
        if closeErr!=nil || count!=asset.Size{return errors.New("download do atualizador incompleto")}
        if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)),expected){
            return errors.New("SHA-256 inválido no atualizador")
        }
        dest:=filepath.Join(rootDir,"VoyagerUpdater.exe")
        backup:=dest+".previous"
        _=os.Remove(backup)
        hadOld:=false
        if _,err=os.Stat(dest);err==nil {
            if err=os.Rename(dest,backup);err!=nil{return fmt.Errorf("não foi possível preparar o atualizador anterior: %w",err)}
            hadOld=true
        } else if !errors.Is(err,os.ErrNotExist) {return err}
        if err=os.Rename(temp,dest);err!=nil {
            if hadOld {_=os.Rename(backup,dest)}
            return fmt.Errorf("não foi possível atualizar o auxiliar: %w",err)
        }
        return nil
    }
    if !found{return errors.New("release sem VoyagerUpdater.exe")}
    return nil
}
