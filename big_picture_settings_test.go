//go:build windows

package main

import (
  "encoding/json"
  "testing"
)
func TestBigPictureOptionsDefaultsAndLegacy(t *testing.T){
    d:=defaultBigPictureOptions()
    if d.View!="grid" || d.Background!="hero" || !d.ShowTitle || !d.ShowPlatform || !d.BlurBackground {t.Fatalf("invalid default: %+v",d)}
    var fromOld BigPictureOptions=d
    if err:=json.Unmarshal([]byte(`{"view":"cartridge","background":"solid","dim":80}`),&fromOld);err!=nil{t.Fatal(err)}
    if !fromOld.ShowTitle || !fromOld.ShowPlatform || !fromOld.BlurBackground || fromOld.View!="cartridge" || fromOld.Dim!=80 {
        t.Fatalf("backward compatibility: %+v",fromOld)
    }
}
func TestBigPictureOptionsNormalization(t *testing.T) {
    o:=BigPictureOptions{View:"untrusted",Background:"other",Dim:400}
    n:=o.normalized()
    if n.View!="grid" || n.Background!="hero" || n.Dim!=90 {t.Fatalf("invalid options persisted: %+v",n)}
    n.Dim=0
    if n.normalized().Dim!=10 {t.Fatal("dim min should clamp")}
}
