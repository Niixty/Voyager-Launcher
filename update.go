//go:build windows

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"voyager/native/internal/updatefiles"
)

// Build tags replace this value in published releases.
var version = "0.9.0-preview"

const releasesEndpoint = "https://api.github.com/repos/Niixty/Voyager-Launcher/releases/latest"
const releaseAsset = "VoyagerLauncher.exe"
const stagedAsset = "VoyagerLauncher.next.exe"

type githubRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name        string `json:"name"`
		DownloadURL string `json:"browser_download_url"`
		Digest      string `json:"digest"`
		Size        int64  `json:"size"`
	} `json:"assets"`
}

type pendingUpdate struct{ Version, File string }

var updateState struct {
	sync.Mutex
	pending *pendingUpdate
	status  string
    approved bool
    available string
    checking bool
    downloading bool
    progress int
    stage string
    lastError string
}

func newerVersion(remote, local string) bool {
	parse := func(s string) ([]int, error) {
		s = strings.TrimPrefix(strings.TrimSpace(s), "v")
		p := strings.Split(s, ".")
		if len(p) != 3 {
			return nil, errors.New("version must be major.minor.patch")
		}
		v := make([]int, 3)
		for i := range p {
			n, e := strconv.Atoi(p[i])
			if e != nil || n < 0 {
				return nil, errors.New("invalid version")
			}
			v[i] = n
		}
		return v, nil
	}
	r, err := parse(remote)
	if err != nil {
		return false
	}
	l, err := parse(local)
	if err != nil {
		return false
	}
	for i := range r {
		if r[i] != l[i] {
			return r[i] > l[i]
		}
	}
	return false
}

func updateHTTPClient() *http.Client {
	return &http.Client{Timeout: 45 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 8 {
			return fmt.Errorf("redirecionamentos demais")
		}
		if req.URL.Scheme != "https" {
			return fmt.Errorf("download sem HTTPS não permitido")
		}
		// GitHub's release files are served through signed redirects on these hosts.
		host := strings.ToLower(req.URL.Hostname())
		if host != "github.com" && host != "objects.githubusercontent.com" && host != "release-assets.githubusercontent.com" && host != "api.github.com" {
			return fmt.Errorf("host inesperado: %s", host)
		}
		return nil
	}}
}

func checkAndStageUpdate() (string, error) {
	if strings.HasSuffix(version, "-preview") {
		return "", nil
	}
    updateState.Lock()
    updateState.progress=0
    updateState.stage="Preparando download"
    updateState.Unlock()
	unlock, err := updatefiles.Lock(rootDir)
	if err != nil {
		return "", err
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", releasesEndpoint, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "VoyagerLauncher/"+version)
	response, err := updateHTTPClient().Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode == 404 {
		return "", nil
	} // No release published yet.
	if response.StatusCode != 200 {
		return "", fmt.Errorf("GitHub retornou HTTP %d", response.StatusCode)
	}
	var release githubRelease
	if err = json.NewDecoder(io.LimitReader(response.Body, 2<<20)).Decode(&release); err != nil {
		return "", err
	}
	if release.Draft || release.Prerelease || !newerVersion(release.TagName, version) {
		return "", nil
	}
	for _, asset := range release.Assets {
		if asset.Name != releaseAsset {
			continue
		}
		if asset.Size < 200000 || asset.Size > 100*1024*1024 {
			return "", errors.New("tamanho do executável fora dos limites")
		}
		if !strings.HasPrefix(asset.Digest, "sha256:") || len(asset.Digest) != 71 {
			return "", errors.New("release sem SHA-256 verificável")
		}
		if _, err := hex.DecodeString(strings.TrimPrefix(asset.Digest, "sha256:")); err != nil {
			return "", err
		}
		if !strings.HasPrefix(asset.DownloadURL, "https://github.com/Niixty/Voyager-Launcher/releases/download/") {
			return "", errors.New("origem do download inválida")
		}
		req, err = http.NewRequestWithContext(ctx, "GET", asset.DownloadURL, nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("User-Agent", "VoyagerLauncher/"+version)
		res, err := updateHTTPClient().Do(req)
		if err != nil {
			return "", err
		}
		if res.StatusCode != 200 {
			res.Body.Close()
			return "", fmt.Errorf("erro no download: HTTP %d", res.StatusCode)
		}
		file, err := os.CreateTemp(rootDir, ".voyager-download-*.tmp")
		if err != nil {
			res.Body.Close()
			return "", err
		}
		tmp := file.Name()
		defer os.Remove(tmp)
		hash := sha256.New()
        updateState.Lock()
        updateState.stage="Baixando atualização"
        updateState.progress=0
        updateState.Unlock()
        meter:=&updateDownloadMeter{source:res.Body,total:asset.Size}
		bytes, e := io.CopyN(io.MultiWriter(file, hash), meter, asset.Size+1)
		closeErr := file.Close()
		res.Body.Close()
		if e != nil && e != io.EOF {
			_ = os.Remove(tmp)
			return "", e
		}
		if closeErr != nil || bytes != asset.Size {
			_ = os.Remove(tmp)
			return "", errors.New("download incompleto")
		}
        updateState.Lock()
        updateState.stage="Validando integridade SHA-256"
        updateState.progress=96
        updateState.Unlock()
		if !strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), strings.TrimPrefix(asset.Digest, "sha256:")) {
			_ = os.Remove(tmp)
			return "", errors.New("SHA-256 inválido")
		}
		dest := filepath.Join(rootDir, stagedAsset)

		if err = os.WriteFile(dest+".sha256", []byte(strings.TrimPrefix(asset.Digest, "sha256:")), 0600); err != nil {
			return "", err
		}
		// A previous downloaded stage may exist. Replace only after the new
        // download and its digest were completely verified.
        if err=os.Remove(dest);err!=nil && !errors.Is(err,os.ErrNotExist) {return "",err}
		if err = os.Rename(tmp, dest); err != nil {
			_ = os.Remove(tmp)
			return "", err
		}
        updateState.Lock()
        updateState.stage="Preparando atualizador de reinicialização"
        updateState.progress=98
        updateState.Unlock()
        if err=downloadUpdatedHelper(ctx,release);err!=nil {
            return "",fmt.Errorf("não foi possível atualizar VoyagerUpdater.exe: %w",err)
        }
		updateState.Lock()
		updateState.pending = &pendingUpdate{Version: release.TagName, File: dest}
        updateState.available=release.TagName
        updateState.stage="Download verificado"
        updateState.progress=100
		updateState.Unlock()
		return release.TagName, nil
	}
	return "", errors.New("release sem VoyagerLauncher.exe")
}

// The classic manager also checks only metadata. Installing requires an
// explicit button click in the modern Settings view.

type updateDownloadMeter struct {
    source io.Reader
    total int64
    received int64
}
func (p *updateDownloadMeter) Read(chunk []byte)(int,error){
    n,e:=p.source.Read(chunk)
    p.received+=int64(n)
    if p.total>0 && n>0 {
        percent:=int(p.received*94/p.total)
        if percent>94 {percent=94}
        updateState.Lock()
        if percent>updateState.progress {updateState.progress=percent}
        updateState.Unlock()
    }
    return n,e
}

func startUpdateCheck(hwnd uintptr) {
 go func(){
    result,err:=probeUpdateStatus()
    updateState.Lock()
    if err!=nil {updateState.status="Não foi possível consultar atualizações: "+err.Error()}
    if err==nil && result.Available {updateState.status="Nova versão "+result.Latest+" disponível nas Configurações"}
    if err==nil && !result.Available {updateState.status="Voyager atualizado"}
    updateState.Unlock()
    call(pPostMessage,hwnd,WM_VOYAGER_UPDATE,0,0)
 }()
}

func launchPendingUpdate() error {
	updateState.Lock()
	u := updateState.pending
    approved:=updateState.approved
	updateState.Unlock()
	if u == nil || !approved {
		return nil
	}
	helper := filepath.Join(rootDir, "VoyagerUpdater.exe")
	if _, err := os.Stat(helper); err != nil {
		return fmt.Errorf("atualizador ausente: %w", err)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(helper, "--apply", exe, u.File, strconv.Itoa(os.Getpid()), "--restart")
	cmd.Dir = rootDir
	return cmd.Start()
}

// A tarefa agendada nao deve mostrar MessageBox quando nao houver rede.
func logSilentUpdateError(err error) {
	if err == nil {
		return
	}
	directory, e := os.UserCacheDir()
	if e != nil {
		return
	}
	folder := filepath.Join(directory, "VoyagerLauncher")
	if e = os.MkdirAll(folder, 0755); e != nil {
		return
	}
	file, e := os.OpenFile(filepath.Join(folder, "update-check.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if e != nil {
		return
	}
	defer file.Close()
	fmt.Fprintf(file, "%s %v\n", time.Now().Format(time.RFC3339), err)
}
