//go:build windows

package main

// SteamGridDB integration for the Windows Studio. Credentials never go to the
// WebView: all authorized API requests happen in this process.

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "image"
    _ "image/jpeg"
    _ "image/png"
    "io"
    "net/http"
    "net/url"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "syscall"
    "time"
    "unsafe"
)

const (
    steamGridAPIBase = "https://www.steamgriddb.com/api/v2"
    steamGridMaxImage = 16 << 20
    steamGridMaxJSON = 3 << 20
)

var (
    sgdbCrypt = syscall.NewLazyDLL("crypt32.dll")
    sgdbProtect = sgdbCrypt.NewProc("CryptProtectData")
    sgdbUnprotect = sgdbCrypt.NewProc("CryptUnprotectData")
    sgdbFree = syscall.NewLazyDLL("kernel32.dll").NewProc("LocalFree")
)

type sgdbBlob struct {
    Size uint32
    Data *byte
}

func sgdbCipher(raw []byte, encrypt bool) ([]byte, error) {
    if len(raw) == 0 || len(raw) > 8192 {
        return nil, errors.New("tamanho da chave SteamGridDB inválido")
    }
    source := sgdbBlob{Size:uint32(len(raw)),Data:&raw[0]}
    var output sgdbBlob
    proc := sgdbUnprotect
    if encrypt { proc = sgdbProtect }
    // CRYPTPROTECT_UI_FORBIDDEN keeps the desktop application non-blocking.
    result, _, err := proc.Call(
        uintptr(unsafe.Pointer(&source)), 0, 0, 0, 0, 1,
        uintptr(unsafe.Pointer(&output)),
    )
    if result == 0 {
        return nil, fmt.Errorf("o Windows não conseguiu proteger a chave SteamGridDB: %w", err)
    }
    defer sgdbFree.Call(uintptr(unsafe.Pointer(output.Data)))
    if output.Data == nil || output.Size == 0 || output.Size > 8192 {
        return nil, errors.New("resposta de proteção de dados inválida")
    }
    copied := append([]byte(nil), unsafe.Slice(output.Data, int(output.Size))...)
    return copied, nil
}

func steamGridKeyPath() (string,error) {
    dir,err:=os.UserConfigDir()
    if err!=nil{return "",err}
    return filepath.Join(dir,"VoyagerLauncher","steamgriddb.dpapi"),nil
}

func getSteamGridKey() (string,error) {
    path,err:=steamGridKeyPath()
    if err!=nil{return "",err}
    enc,err:=os.ReadFile(path)
    if err!=nil{return "",err}
    raw,err:=sgdbCipher(enc,false)
    if err!=nil{return "",err}
    value:=strings.TrimSpace(string(raw))
    for i:=range raw {raw[i]=0}
    if value=="" {return "",errors.New("chave SteamGridDB vazia")}
    return value,nil
}

func (s *Studio) SteamGridKeyConfigured() bool {
    key,err:=getSteamGridKey()
    return err==nil && key!=""
}

func (s *Studio) SaveSteamGridKey(key string) error {
    key = strings.TrimSpace(key)
    if len(key) < 12 || len(key) > 512 || strings.ContainsAny(key,"\r\n \t") {
        return errors.New("informe uma chave SteamGridDB válida")
    }
    buf:=[]byte(key)
    protected,err:=sgdbCipher(buf,true)
    for i:=range buf {buf[i]=0}
    if err!=nil{return err}
    path,err:=steamGridKeyPath()
    if err!=nil{return err}
    if err=os.MkdirAll(filepath.Dir(path),0700);err!=nil{return err}
    temp,err:=os.CreateTemp(filepath.Dir(path),".sgdb-*.tmp")
    if err!=nil{return err}
    defer os.Remove(temp.Name())
    if err=temp.Chmod(0600);err!=nil{temp.Close();return err}
    if _,err=temp.Write(protected);err!=nil{temp.Close();return err}
    if err=temp.Close();err!=nil{return err}
    // Windows cannot rename over an existing destination: replace the key.
    _=os.Remove(path)
    return os.Rename(temp.Name(),path)
}

func (s *Studio) RemoveSteamGridKey() error {
    path,err:=steamGridKeyPath()
    if err!=nil{return err}
    if err=os.Remove(path);errors.Is(err,os.ErrNotExist){return nil}
    return err
}

// The title must exist in config.json; readConfig intentionally falls back
// to the directory ID, which is not an appropriate search name.
func steamGridSavedTitle(id string) (string,error) {
    if !validID.MatchString(id) {return "",errors.New("ID de jogo inválido")}
    bytes,err:=os.ReadFile(configFile(id))
    if err!=nil{return "",err}
    var saved struct { Title string `json:"title"` }
    if err=json.Unmarshal(bytes,&saved);err!=nil{return "",err}
    title:=strings.TrimSpace(saved.Title)
    if title=="" {
        return "",errors.New("Informe primeiro o nome do jogo em Editar jogo e salve para pesquisar no SteamGridDB.")
    }
    if len([]rune(title))>180 {return "",errors.New("o nome do jogo é muito longo para a pesquisa")}
    return title,nil
}

type SteamGridGame struct {
    ID int64 `json:"id"`
    Name string `json:"name"`
}

type SteamGridImage struct {
    ID int64 `json:"id"`
    URL string `json:"url"`
    Thumb string `json:"thumb"`
    Width int `json:"width"`
    Height int `json:"height"`
}

type SteamGridImagePage struct {
    Page int `json:"page"`
    Total int `json:"total"`
    Limit int `json:"limit"`
    Items []SteamGridImage `json:"items"`
}

type steamGridEnvelope struct {
    Success bool `json:"success"`
    Data json.RawMessage `json:"data"`
    Errors []string `json:"errors"`
    Page int `json:"page"`
    Total int `json:"total"`
    Limit int `json:"limit"`
}

func steamGridURLAllowed(raw string) bool {
    u,err:=url.Parse(raw)
    if err!=nil || u.Scheme!="https" || u.User!=nil || u.Host=="" || u.Port()!="" {return false}
    host:=strings.ToLower(u.Hostname())
    return host=="steamgriddb.com" || strings.HasSuffix(host,".steamgriddb.com")
}

func steamGridClient() *http.Client {
    return &http.Client{
        Timeout:18*time.Second,
        CheckRedirect:func(req *http.Request,via []*http.Request)error{
            if len(via)>4 || !steamGridURLAllowed(req.URL.String()) {return http.ErrUseLastResponse}
            return nil
        },
    }
}

func steamGridAPIRequest(ctx context.Context, endpoint, key string) (steamGridEnvelope,error) {
    var result steamGridEnvelope
    target:=steamGridAPIBase+endpoint
    if !steamGridURLAllowed(target) {return result,errors.New("endpoint SteamGridDB inválido")}
    req,err:=http.NewRequestWithContext(ctx,"GET",target,nil)
    if err!=nil{return result,err}
    req.Header.Set("Authorization","Bearer "+key)
    req.Header.Set("Accept","application/json")
    req.Header.Set("User-Agent","VoyagerLauncher/SteamGridDB")
    resp,err:=steamGridClient().Do(req)
    if err!=nil{return result,fmt.Errorf("falha na conexão com o SteamGridDB: %w",err)}
    defer resp.Body.Close()
    switch resp.StatusCode {
    case http.StatusUnauthorized,http.StatusForbidden:
        return result,errors.New("Chave SteamGridDB rejeitada (401/403). Atualize-a em Configurações.")
    case http.StatusTooManyRequests:
        return result,errors.New("Limite de consultas SteamGridDB atingido (429). Tente novamente depois.")
    }
    if resp.StatusCode!=200 {return result,fmt.Errorf("SteamGridDB retornou HTTP %d",resp.StatusCode)}
    data,err:=io.ReadAll(io.LimitReader(resp.Body,steamGridMaxJSON+1))
    if err!=nil{return result,err}
    if len(data)>steamGridMaxJSON{return result,errors.New("resposta SteamGridDB maior que o permitido")}
    if err=json.Unmarshal(data,&result);err!=nil{return result,fmt.Errorf("resposta SteamGridDB inválida: %w",err)}
    if !result.Success {return result,fmt.Errorf("SteamGridDB não concluiu a solicitação: %s",strings.Join(result.Errors,", "))}
    return result,nil
}

// SteamGridSavedGameTitle checks the persisted name before prompting for the API key.
func (s *Studio) SteamGridSavedGameTitle(id string) (string,error) {
    return steamGridSavedTitle(id)
}

func (s *Studio) SteamGridGames(id string) ([]SteamGridGame,error) {
    title,err:=steamGridSavedTitle(id)
    if err!=nil{return nil,err}
    key,err:=getSteamGridKey()
    if err!=nil{return nil,errors.New("Configure sua chave SteamGridDB em Configurações para pesquisar.")}
    ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second)
    defer cancel()
    env,err:=steamGridAPIRequest(ctx,"/search/autocomplete/"+url.PathEscape(title),key)
    if err!=nil{return nil,err}
    var games []SteamGridGame
    if err=json.Unmarshal(env.Data,&games);err!=nil{return nil,err}
    if len(games)>35 {games=games[:35]}
    if games==nil {games=[]SteamGridGame{}}
    return games,nil
}

func (s *Studio) SteamGridImages(gameID int64,kind string,page int) (SteamGridImagePage,error) {
    out:=SteamGridImagePage{Page:page,Limit:50,Items:[]SteamGridImage{}}
    if gameID<=0 || gameID>1_000_000_000 || page<0 || page>200 || (kind!="hero"&&kind!="grid"){
        return out,errors.New("pesquisa de imagens inválida")
    }
    key,err:=getSteamGridKey()
    if err!=nil{return out,errors.New("Configure sua chave SteamGridDB em Configurações para pesquisar.")}
    collection:="grids"
    if kind=="hero" {collection="heroes"}
    ctx,cancel:=context.WithTimeout(context.Background(),20*time.Second)
    defer cancel()
    env,err:=steamGridAPIRequest(ctx,"/"+collection+"/game/"+strconv.FormatInt(gameID,10)+"?limit=50&page="+strconv.Itoa(page)+"&types=static",key)
    if err!=nil{return out,err}
    var items []SteamGridImage
    if err=json.Unmarshal(env.Data,&items);err!=nil{return out,err}
    for _,item:=range items {
        if !steamGridURLAllowed(item.URL) {continue}
        if !steamGridURLAllowed(item.Thumb) {item.Thumb=item.URL}
        out.Items=append(out.Items,item)
    }
    out.Total=env.Total
    if env.Limit>0 {out.Limit=env.Limit}
    return out,nil
}

func (s *Studio) ApplySteamGridImage(id,kind,assetURL string) error {
    if !validID.MatchString(id) || (kind!="hero"&&kind!="grid") {
        return errors.New("jogo ou tipo de imagem inválido")
    }
    if !steamGridURLAllowed(assetURL) {return errors.New("A imagem precisa vir de um endereço HTTPS do SteamGridDB")}
    if _,err:=readConfig(id);err!=nil{return err}
    ctx,cancel:=context.WithTimeout(context.Background(),35*time.Second)
    defer cancel()
    req,err:=http.NewRequestWithContext(ctx,"GET",assetURL,nil)
    if err!=nil{return err}
    resp,err:=steamGridClient().Do(req)
    if err!=nil{return fmt.Errorf("não foi possível baixar a imagem: %w",err)}
    defer resp.Body.Close()
    if resp.StatusCode!=200 {return fmt.Errorf("download SteamGridDB retornou HTTP %d",resp.StatusCode)}
    download,err:=io.ReadAll(io.LimitReader(resp.Body,steamGridMaxImage+1))
    if err!=nil{return err}
    if len(download)>steamGridMaxImage{return errors.New("Imagem maior que 16 MB")}
    mime:=http.DetectContentType(download)
    if mime!="image/png" && mime!="image/jpeg" {
        return errors.New("Formato não suportado: escolha uma imagem PNG ou JPEG estática.")
    }
    cfg,_,err:=image.DecodeConfig(bytes.NewReader(download))
    if err!=nil{return errors.New("Imagem SteamGridDB inválida ou corrompida")}
    if cfg.Width<=0 || cfg.Height<=0 || cfg.Width>10000 || cfg.Height>10000 || int64(cfg.Width)*int64(cfg.Height)>40_000_000 {
        return errors.New("Dimensões da imagem excedem o limite de segurança")
    }
    temp,err:=os.CreateTemp("","voyager-steamgrid-*.png")
    if err!=nil{return err}
    defer os.Remove(temp.Name())
    if _,err=temp.Write(download);err!=nil{temp.Close();return err}
    if err=temp.Close();err!=nil{return err}

    s.mu.Lock()
    defer s.mu.Unlock()
    if _,err=readConfig(id);err!=nil{return err}
    dir:=gameDir(id)
    output,err:=os.CreateTemp(dir,".steamgrid-output-*.png")
    if err!=nil{return err}
    candidate:=output.Name()
    if err=output.Close();err!=nil{return err}
    // GDI+ writes to candidate via a temporary PNG + rename. On Windows,
    // renaming over an existing placeholder fails, so remove it first.
    if err=os.Remove(candidate);err!=nil{return err}
    defer os.Remove(candidate)
    width,height:=600,900
    if kind=="hero" {width,height=1920,620}
    if err=cropToPNG(temp.Name(),candidate,width,height);err!=nil{return err}
    dest:=filepath.Join(dir,kind+".png")
    backup:=""
    if _,err=os.Stat(dest);err==nil {
        backup=dest+".backup-"+time.Now().Format("20060102-150405.000000000")
        if err=os.Rename(dest,backup);err!=nil{return err}
    }else if !errors.Is(err,os.ErrNotExist){return err}
    if err=os.Rename(candidate,dest);err!=nil{
        if backup!="" {_=os.Rename(backup,dest)}
        return err
    }
    return nil
}
