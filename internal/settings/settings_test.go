package settings

import (
 "os"
 "path/filepath"
 "testing"
)

func TestThemes(t *testing.T){
 if len(Presets)!=8 {t.Fatal("missing presets")}
 for _,p:=range Presets {
  if !ValidTheme(p.ID)||p.Accent==0||p.Dark==0 {t.Fatal(p.ID)}
 }
 if PresetFor("invalid").ID!="cinza" {t.Fatal("fallback")}
}

func TestSaveAndLoad(t *testing.T){
 p:=filepath.Join(t.TempDir(),"app","settings.json")
 c:=Load(p)
 if !c.ShowPercentage||!c.BlurBackground {t.Fatal("defaults")}
 c.Theme="roxo";c.ShowPercentage=false;c.BackgroundDim=88
 if err:=Save(p,c);err!=nil {t.Fatal(err)}
 got:=Load(p)
 if got.Theme!="roxo"||got.ShowPercentage||got.BackgroundDim!=88 {t.Fatalf("roundtrip: %+v",got)}
 got.Theme="verde"
 if err:=Save(p,got);err!=nil {t.Fatal(err)}
 if Load(p).Theme!="verde" {t.Fatal("second save")}
 if err:=os.WriteFile(p,[]byte("{\"theme\":\"invalid\",\"backgroundDim\":100}"),0600);err!=nil {t.Fatal(err)}
 if got=Load(p);got.Theme!="cinza"||got.BackgroundDim!=78||!got.ShowPercentage {t.Fatalf("invalid defaults: %+v",got)}
 if err:=os.WriteFile(p,[]byte("invalid"),0600);err!=nil {t.Fatal(err)}
 if Load(p).Theme!="cinza" {t.Fatal("invalid json")}
}
