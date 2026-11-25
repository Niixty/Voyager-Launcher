//go:build windows

package main

import (
    "encoding/json"
    "os"
    "path/filepath"
    "voyager/native/internal/jsonstore"
)

// Saved separately from global Settings to avoid changing a player's game
// library, Hero, custom loading layouts or streaming configuration.
type BigPictureOptions struct {
    View string `json:"view"` // grid | cartridge
    Background string `json:"background"` // hero | solid
    ShowTitle bool `json:"showTitle"`
    ShowPlatform bool `json:"showPlatform"`
    BlurBackground bool `json:"blurBackground"`
    Dim int `json:"dim"`
}
func defaultBigPictureOptions() BigPictureOptions {
    return BigPictureOptions{
        View:"grid",Background:"hero",ShowTitle:true,ShowPlatform:true,
        BlurBackground:true,Dim:65,
    }
}
func (o BigPictureOptions) normalized() BigPictureOptions {
    if o.View!="grid" && o.View!="cartridge" {o.View="grid"}
    if o.Background!="hero" && o.Background!="solid" {o.Background="hero"}
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
