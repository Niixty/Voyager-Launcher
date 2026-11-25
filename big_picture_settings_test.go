//go:build windows

package main

import (
  "encoding/json"
  "testing"
)
func TestBigPictureOptionsDefaultsAndLegacy(t *testing.T){
    d:=defaultBigPictureOptions()
    if d.View!="grid" || d.Background!="hero" || d.ButtonStyle!="auto" || !d.ShowTitle || !d.ShowPlatform || !d.BlurBackground {t.Fatalf("invalid default: %+v",d)}
    var fromOld BigPictureOptions=d
    if err:=json.Unmarshal([]byte(`{"view":"cartridge","background":"solid","dim":80}`),&fromOld);err!=nil{t.Fatal(err)}
    if fromOld.ButtonStyle!="auto" || !fromOld.ShowTitle || !fromOld.ShowPlatform || !fromOld.BlurBackground || fromOld.View!="cartridge" || fromOld.Dim!=80 {
        t.Fatalf("backward compatibility: %+v",fromOld)
    }
}
func TestBigPictureOptionsNormalization(t *testing.T) {
    o:=BigPictureOptions{View:"untrusted",Background:"other",ButtonStyle:"unknown",Dim:400}
    n:=o.normalized()
    if n.View!="grid" || n.Background!="hero" || n.ButtonStyle!="auto" || n.Dim!=90 {t.Fatalf("invalid options persisted: %+v",n)}
    n.Dim=0
    if n.normalized().Dim!=10 {t.Fatal("dim min should clamp")}
}

func TestBigPictureControllerGlyphPreference(t *testing.T) {
    for _,style:=range []string{"auto","xbox","playstation"} {
        options:=defaultBigPictureOptions()
        options.ButtonStyle=style
        encoded,err:=json.Marshal(options)
        if err!=nil{t.Fatal(err)}
        var restored BigPictureOptions
        if err:=json.Unmarshal(encoded,&restored);err!=nil{t.Fatal(err)}
        if restored.normalized().ButtonStyle!=style {t.Fatalf("botões não persistiram: %q",style)}
    }
}
