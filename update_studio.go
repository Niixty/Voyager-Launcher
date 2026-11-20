//go:build windows

package main

import (
    "context"
    "encoding/hex"
    "encoding/json"
    "errors"
    "fmt"
    "io"
    "net/http"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
)

type UpdateInfo struct {
    Current string `json:"current"`
    Latest string `json:"latest"`
    Available bool `json:"available"`
    Checking bool `json:"checking"`
    Downloading bool `json:"downloading"`
    Ready bool `json:"ready"`
    Progress int `json:"progress"`
    Stage string `json:"stage"`
    Error string `json:"error"`
    Preview bool `json:"preview"`
}

var probeUpdateMu sync.Mutex

func currentUpdateInfo() UpdateInfo {
    updateState.Lock()
    defer updateState.Unlock()
    ready:=updateState.pending!=nil
    latest:=updateState.available
    if ready {latest=updateState.pending.Version}
    return UpdateInfo{
        Current:version,
        Latest:latest,
        Available:latest!="" && newerVersion(latest,version),
        Ready:ready,Checking:updateState.checking,
        Downloading:updateState.downloading,Progress:updateState.progress,
        Stage:updateState.stage,Error:updateState.lastError,
        Preview:strings.HasSuffix(version,"-preview"),
    }
}

func probeUpdateStatus()(result UpdateInfo, err error) {
    if strings.HasSuffix(version,"-preview") {
        updateState.Lock()
        updateState.checking=false
        updateState.stage="Versão de desenvolvimento"
        updateState.Unlock()
        return currentUpdateInfo(),nil
    }
    probeUpdateMu.Lock()
    defer probeUpdateMu.Unlock()
    updateState.Lock()
    if updateState.downloading {
        updateState.Unlock()
        return currentUpdateInfo(),nil
    }
    updateState.checking=true
    updateState.lastError=""
    updateState.Unlock()
    defer func(){
        updateState.Lock()
        updateState.checking=false
        updateState.Unlock()
        result=currentUpdateInfo()
    }()
    ctx,cancel:=context.WithTimeout(context.Background(),18*time.Second)
    defer cancel()
    req,err:=http.NewRequestWithContext(ctx,"GET",releasesEndpoint,nil)
    if err!=nil{return rememberUpdateProbeError(err)}
    req.Header.Set("Accept","application/vnd.github+json")
    req.Header.Set("User-Agent","VoyagerLauncher/"+version)
    response,err:=updateHTTPClient().Do(req)
    if err!=nil{return rememberUpdateProbeError(fmt.Errorf("não foi possível consultar o GitHub: %w",err))}
    defer response.Body.Close()
    if response.StatusCode==http.StatusNotFound {
        updateState.Lock()
        updateState.available=""
        updateState.stage="Nenhuma release publicada"
        updateState.Unlock()
        return currentUpdateInfo(),nil
    }
    if response.StatusCode!=http.StatusOK {
        return rememberUpdateProbeError(fmt.Errorf("GitHub retornou HTTP %d",response.StatusCode))
    }
    var release githubRelease
    if err=json.NewDecoder(io.LimitReader(response.Body,2<<20)).Decode(&release);err!=nil {
        return rememberUpdateProbeError(fmt.Errorf("resposta inválida do GitHub: %w",err))
    }
    if !release.Draft && !release.Prerelease && newerVersion(release.TagName,version) {
        valid:=false
        for _,a:=range release.Assets {
            if a.Name!=releaseAsset {continue}
            if a.Size<200000 || a.Size>100*1024*1024 || !strings.HasPrefix(a.Digest,"sha256:") || len(a.Digest)!=71 {
                return rememberUpdateProbeError(errors.New("release sem executável verificável"))
            }
            if _,err=hex.DecodeString(strings.TrimPrefix(a.Digest,"sha256:"));err!=nil{return rememberUpdateProbeError(err)}
            if !strings.HasPrefix(a.DownloadURL,"https://github.com/Niixty/Voyager-Launcher/releases/download/") {
                return rememberUpdateProbeError(errors.New("endereço da release não é confiável"))
            }
            valid=true
            break
        }
        if !valid {return rememberUpdateProbeError(errors.New("release sem VoyagerLauncher.exe"))}
        updateState.Lock()
        updateState.available=release.TagName
        if updateState.pending==nil {updateState.stage="Nova versão disponível"}
        updateState.Unlock()
    } else {
        updateState.Lock()
        if updateState.pending==nil {updateState.available=""}
        updateState.stage="O Voyager está atualizado"
        updateState.Unlock()
    }
    return currentUpdateInfo(),nil
}

func rememberUpdateProbeError(err error)(UpdateInfo,error) {
    updateState.Lock()
    updateState.lastError=err.Error()
    updateState.stage="Falha ao verificar atualizações"
    updateState.Unlock()
    return currentUpdateInfo(),err
}

func (s *Studio) CheckUpdateStatus()(UpdateInfo,error) {
    return probeUpdateStatus()
}
func (s *Studio) UpdateInfo() UpdateInfo {
    return currentUpdateInfo()
}

func (s *Studio) PrepareUpdate()(string,error) {
    updateState.Lock()
    if updateState.downloading {
        updateState.Unlock()
        return "",errors.New("o download já está em andamento")
    }
    updateState.downloading=true
    updateState.progress=0
    updateState.stage="Conectando ao GitHub"
    updateState.lastError=""
    updateState.Unlock()
    latest,err:=checkAndStageUpdate()
    updateState.Lock()
    updateState.downloading=false
    if err!=nil {
        updateState.lastError=err.Error()
        updateState.stage="Download não concluído"
    } else if latest=="" {
        updateState.stage="Nenhuma atualização nova"
    } else {
        updateState.stage="Download concluído e verificado"
        updateState.progress=100
    }
    updateState.Unlock()
    if err!=nil{return "",err}
    if latest=="" {return "",errors.New("nenhuma versão nova disponível")}
    return latest,nil
}

// The helper starts only after an explicit user action. Closing normally
// never installs an update in the background.
func (s *Studio) InstallUpdateNow() error {
    updateState.Lock()
    defer updateState.Unlock()
    if updateState.downloading {return errors.New("aguarde o download terminar")}
    if updateState.pending==nil {return errors.New("nenhuma atualização preparada")}
    if _,err:=os.Stat(updateState.pending.File);err!=nil {
        return fmt.Errorf("arquivo de atualização ausente: %w",err)
    }
    if _,err:=os.Stat(filepath.Join(rootDir,"VoyagerUpdater.exe"));err!=nil {
        return fmt.Errorf("atualizador do Windows ausente: %w",err)
    }
    updateState.approved=true
    updateState.stage="Fechando o Voyager para instalar"
    return nil
}
