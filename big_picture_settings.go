//go:build windows

package main

import (
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "voyager/native/internal/heroaccent"
    "voyager/native/internal/jsonstore"
)

// Saved separately from global Settings to avoid changing a player's game
// library, Hero, custom loading layouts or streaming configuration.
type BigPictureOptions struct {
    View string `json:"view"` // grid | cartridge
    Background string `json:"background"` // hero | solid
    ButtonStyle string `json:"buttonStyle"` // auto | xbox | playstation (visual glyphs only)
    ShowTitle bool `json:"showTitle"`
    ShowPlatform bool `json:"showPlatform"`
    BlurBackground bool `json:"blurBackground"`
    Dim int `json:"dim"`
}
func defaultBigPictureOptions() BigPictureOptions {
    return BigPictureOptions{
        View:"grid",Background:"hero",ButtonStyle:"auto",ShowTitle:true,ShowPlatform:true,
        BlurBackground:true,Dim:65,
    }
}
func (o BigPictureOptions) normalized() BigPictureOptions {
    if o.View!="grid" && o.View!="cartridge" {o.View="grid"}
    if o.Background!="hero" && o.Background!="solid" {o.Background="hero"}
    if o.ButtonStyle!="auto" && o.ButtonStyle!="xbox" && o.ButtonStyle!="playstation" {o.ButtonStyle="auto"}
    if o.Dim<10 {o.Dim=10}
    if o.Dim>90 {o.Dim=90}
    return o
}
func bigPictureOptionsPath()string {
    return filepath.Join(rootDir,"bigpicture.json")
}
func loadBigPictureOptions()BigPictureOptions {
    config:=defaultBigPictureOptions()
    data,err:=os.ReadFile(bigPictureOptionsPath())
    if err!=nil{return config}
    if len(data)>64*1024 {return config}
    if json.Unmarshal(data,&config)!=nil{return defaultBigPictureOptions()}
    return config.normalized()
}
func (s *Studio) BigPictureOptions()BigPictureOptions {
    s.mu.Lock()
    defer s.mu.Unlock()
    return loadBigPictureOptions()
}
func (s *Studio) SaveBigPictureOptions(value BigPictureOptions)error {
    s.mu.Lock()
    defer s.mu.Unlock()
    value=value.normalized()
    data,err:=json.MarshalIndent(value,"","  ")
    if err!=nil{return err}
    return jsonstore.Write(bigPictureOptionsPath(),append(data,'\n'))
}

// BigPictureAccent returns the SAME dominant Hero accent as the native
// loading screen. Reuse its well-tested histogram rather than sampling
// blurred/scaled Hero pixels in the WebView. The UI caches results per game.
func (s *Studio) BigPictureAccent(id string) string {
    fallback:=func()string {return fmt.Sprintf("#%06X",heroaccent.Neutral&0xffffff)}
    if !validID.MatchString(id) {return fallback()}
    c,err:=readConfig(id)
    if err!=nil {return fallback()}
    path:=artworkPath(c,id,"hero")
    return fmt.Sprintf("#%06X",heroaccent.FromFile(path)&0xffffff)
}
