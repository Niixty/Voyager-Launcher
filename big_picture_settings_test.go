//go:build windows

package main

import (
  "encoding/json"
  "testing"
)
func TestBigPictureOptionsDefaultsAndLegacy(t *testing.T){
    d:=defaultBigPictureOptions()
    if d.DefaultBackgroundHeroColor || d.View!="grid" || d.Background!="hero" || d.ButtonStyle!="auto" || !d.ShowTitle || !d.ShowPlatform || !d.BlurBackground || d.ScrollDirection!="vertical" || d.Columns!=4 || d.Rows!=2 || d.IconSize!=100 {t.Fatalf("invalid default: %+v",d)}
    var fromOld BigPictureOptions=d
    if err:=json.Unmarshal([]byte(`{"view":"cartridge","background":"solid","dim":80}`),&fromOld);err!=nil{t.Fatal(err)}
    if fromOld.DefaultBackgroundHeroColor || fromOld.ButtonStyle!="auto" || !fromOld.ShowTitle || !fromOld.ShowPlatform || !fromOld.BlurBackground || fromOld.View!="cartridge" || fromOld.Dim!=80 ||
        fromOld.ScrollDirection!="vertical" || fromOld.Columns!=4 || fromOld.Rows!=2 || fromOld.IconSize!=100 {
        t.Fatalf("backward compatibility: %+v",fromOld)
    }
}
func TestBigPictureOptionsNormalization(t *testing.T) {
    o:=BigPictureOptions{View:"untrusted",Background:"other",ButtonStyle:"unknown",ScrollDirection:"diagonal",Columns:200,Rows:9,IconSize:999,Dim:400}
    n:=o.normalized()
    if n.View!="grid" || n.Background!="hero" || n.ButtonStyle!="auto" || n.Dim!=90 || n.ScrollDirection!="vertical" || n.Columns!=8 || n.Rows!=4 || n.IconSize!=140 {t.Fatalf("invalid options persisted: %+v",n)}
    n.Dim=0
    if n.normalized().Dim!=10 {t.Fatal("dim min should clamp")}
    n.Columns=0;n.Rows=0;n.IconSize=0
    defaults:=n.normalized()
    if defaults.Columns!=4 || defaults.Rows!=2 || defaults.IconSize!=100 {t.Fatalf("invalid layout defaults: %+v",defaults)}
    n.IconSize=113
    if n.normalized().IconSize!=110 {t.Fatal("icon size must snap to 10% steps")}
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

func TestBigPictureScrollingLayoutPreference(t *testing.T) {
    for _,direction:=range []string{"vertical","horizontal"} {
        options:=defaultBigPictureOptions()
        options.ScrollDirection=direction
        options.Columns=6
        options.Rows=3
        options.IconSize=130
        encoded,err:=json.Marshal(options)
        if err!=nil {t.Fatal(err)}
        var restored BigPictureOptions
        if err:=json.Unmarshal(encoded,&restored);err!=nil {t.Fatal(err)}
        restored=restored.normalized()
        if restored.ScrollDirection!=direction || restored.Columns!=6 || restored.Rows!=3 || restored.IconSize!=130 {
            t.Fatalf("layout preference not preserved: %+v",restored)
        }
    }
}

func TestBigPictureHeroWavePalettePreference(t *testing.T) {
    options:=defaultBigPictureOptions()
    options.Background="solid"
    options.DefaultBackgroundHeroColor=true
    encoded,err:=json.Marshal(options)
    if err!=nil {t.Fatal(err)}
    var restored BigPictureOptions
    if err:=json.Unmarshal(encoded,&restored);err!=nil {t.Fatal(err)}
    restored=restored.normalized()
    if restored.Background!="solid" || !restored.DefaultBackgroundHeroColor {
        t.Fatalf("Cor da Hero no fundo padrão não persistiu: %+v",restored)
    }
}
