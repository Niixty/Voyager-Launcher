package loadinglayout

import (
  "os"
  "path/filepath"
  "testing"
)

func TestDefaults(t *testing.T){
  d:=Default()
  if len(d.Elements)!=8 {t.Fatalf("expected 8 nodes, got %d",len(d.Elements))}
  for _,node:=range d.Elements{
   if !node.Visible || node.W<=0 || node.X+node.W>1600 || node.Y+node.H>900 {t.Fatal(node)}
  }
}
func TestRoundtripVisibilityAndOrdering(t *testing.T){
 p:=filepath.Join(t.TempDir(),"Games","Sample","loading_layout.json")
 l:=Default()
 l.Find(Title).X=320
 l.Find(Description).Visible=false
 l.MoveLayer(Cartridge,1)
 if err:=Save(p,l);err!=nil{t.Fatal(err)}
 actual,err:=Load(p)
 if err!=nil{t.Fatal(err)}
 if actual.Find(Title).X!=320 || actual.Find(Description).Visible {t.Fatal("lost settings")}
 if actual.Elements[0].ID!=Platform {t.Fatal("ordering lost")}
 if _,err=os.Stat(p);err!=nil{t.Fatal(err)}
 if err=Save(p,actual);err!=nil{t.Fatal(err)}
}
func TestClampAndFallback(t *testing.T){
 l:=Default()
 l.Find(Title).X=-100
 l.Find(Title).W=5000
 l.Find(Title).H=0
 l.Find(Title).FontSize=200
 l.Find(Phase).Color="bad-value"
 n:=l.Normalize()
 title:=n.Find(Title)
 if title.X<0 || title.W>1600 || title.FontSize>90 || title.H<=0 {t.Fatal(title)}
 if n.Find(Phase).Color!="auto" {t.Fatal("bad color")}
 if Snap(46,10)!=50 || Snap(44,10)!=40 {t.Fatal("snap")}
}
func TestTextAlignmentPersistsAndValidates(t *testing.T) {
    l := Default()
    title := l.Find(Title)
    title.TextAlign = "right"
    title.VerticalAlign = "bottom"
    path := filepath.Join(t.TempDir(), "loading_layout.json")
    if err := Save(path, l); err != nil { t.Fatal(err) }
    loaded, err := Load(path)
    if err != nil { t.Fatal(err) }
    actual := loaded.Find(Title)
    if actual.TextAlign != "right" || actual.VerticalAlign != "bottom" {
        t.Fatalf("alinhamentos não persistidos: %+v", actual)
    }
    // Compatibilidade com layouts antigos sem campos de alinhamento.
    legacy := Default()
    legacy.Find(Title).TextAlign = ""
    legacy.Find(Title).VerticalAlign = ""
    normalizedLegacy := legacy.Normalize()
    legacyNormalized := normalizedLegacy.Find(Title)
    if legacyNormalized.TextAlign != "left" || legacyNormalized.VerticalAlign != "top" {
        t.Fatalf("padrão legado inesperado: %+v", legacyNormalized)
    }
    loaded.Find(Title).TextAlign = "invalid"
    loaded.Find(Title).VerticalAlign = "invalid"
    loaded.Find(Cartridge).TextAlign = "right"
    loaded.Find(Cartridge).VerticalAlign = "bottom"
    normalized := loaded.Normalize()
    if normalized.Find(Title).TextAlign != "left" || normalized.Find(Title).VerticalAlign != "top" { t.Fatal("alinhamento inválido não foi corrigido") }
    if normalized.Find(Cartridge).TextAlign != "" || normalized.Find(Cartridge).VerticalAlign != "" { t.Fatal("alinhamento indevido no cartucho") }
}

func TestUnknownElementsIgnored(t *testing.T){
 l:=Default()
 l.Elements=append(l.Elements,Element{ID:"bad",X:10,Y:10,W:100,H:100,Visible:true})
 l.Elements=append(l.Elements,l.Elements[0])
 if len(l.Normalize().Elements)!=8 {t.Fatal("unknown or duplicate not discarded")}
}
