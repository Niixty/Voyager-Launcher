package loadinglayout

import (
  "os"
  "path/filepath"
  "testing"
)

func TestDefaults(t *testing.T){
  d:=Default()
  if len(d.Elements)!=7 {t.Fatalf("expected 7 nodes, got %d",len(d.Elements))}
  for _,node:=range d.Elements{
   if (node.ID!=Progress && !node.Visible) || node.W<=0 || node.X+node.W>1600 || node.Y+node.H>900 {t.Fatal(node)}
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
 if title.X!=-100 || title.W>1600 || title.FontSize>90 || title.H<=0 {t.Fatal(title)}
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
 if len(l.Normalize().Elements)!=7 {t.Fatal("unknown or duplicate not discarded")}
}


func TestOffscreenCoordinatesRoundtrip(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"loading_layout.json")
 l:=Default()
 l.Find(Cartridge).Y=-50
 l.Find(Cartridge).X=-200
 l.Find(Title).X=1740
 l.Find(Description).Y=995
 l.Find(Phase).Y=-120
 if err:=Save(path,l);err!=nil{t.Fatal(err)}
 loaded,err:=Load(path)
 if err!=nil{t.Fatal(err)}
 for _,item:=range []struct{ID string;X,Y float64}{
    {Cartridge,-200,-50},
    {Title,1740,300},
    {Description,526,995},
    {Phase,526,-120},
 } {
    element:=loaded.Find(item.ID)
    if element==nil || element.X!=item.X || element.Y!=item.Y {
        t.Fatalf("invalid off-screen persistence for %s: %+v",item.ID,element)
    }
 }
}
func TestPositionSafetyLimits(t *testing.T){
 l:=Default()
 l.Find(Cartridge).X=-90000
 l.Find(Title).Y=90000
 normalized:=l.Normalize()
 if normalized.Find(Cartridge).X!=PositionMin || normalized.Find(Title).Y!=PositionMax {t.Fatal(normalized)}
}

func TestLegacyHeadingRemovedWhilePreservingOtherNodes(t *testing.T) {
    layout:=Default()
    layout.Elements=append(layout.Elements,Element{ID:Heading,X:500,Y:400,W:300,H:50,Visible:true,FontSize:26})
    if normalized:=layout.Normalize();normalized.Find(Heading)!=nil || len(normalized.Elements)!=7 {
        t.Fatalf("a camada Mensagem não pode aparecer após normalização: %+v",normalized)
    }
    path:=filepath.Join(t.TempDir(),"legacy_loading.json")
    if err:=Save(path,layout);err!=nil{t.Fatal(err)}
    restored,err:=Load(path)
    if err!=nil{t.Fatal(err)}
    if restored.Find(Heading)!=nil || len(restored.Elements)!=7 {t.Fatal("camada legada reapareceu")}
}
func TestTitleAlwaysUsesHeroColorIncludingLegacyLayouts(t *testing.T) {
    defaults:=Default()
    if got:=defaults.Find(Title).Color;got!="hero" {t.Fatalf("cor padrão do título: %q",got)}
    for _,oldColor:=range []string{"white","muted","auto","hero",""} {
        old:=Default()
        old.Find(Title).Color=oldColor
        old.Find(Title).X=-50
        old.Find(Title).FontSize=52
        normalized:=old.Normalize()
        n:=normalized.Find(Title)
        if n.Color!="hero" || n.X!=-50 || n.FontSize!=52 {
            t.Fatalf("migração de cor %q alterou layout: %+v",oldColor,n)
        }
        path:=filepath.Join(t.TempDir(),"loading_layout.json")
        if err:=Save(path,old);err!=nil {t.Fatal(err)}
        loaded,err:=Load(path)
        if err!=nil {t.Fatal(err)}
        if got:=loaded.Find(Title).Color;got!="hero" {t.Fatalf("cor legada %q não migrou: %q",oldColor,got)}
    }
}
func TestHalfPixelCenterCoordinatesSurviveRoundtrip(t *testing.T){
    layout:=Default()
    cartridge:=layout.Find(Cartridge)
    cartridge.X=(VirtualWidth-cartridge.W)/2
    cartridge.Y=(VirtualHeight-cartridge.H)/2
    if cartridge.X!=647.5 || cartridge.Y!=244.5 {t.Fatalf("centro esperado 647.5/244.5, obtido %.2f/%.2f",cartridge.X,cartridge.Y)}
    path:=filepath.Join(t.TempDir(),"center.json")
    if err:=Save(path,layout);err!=nil{t.Fatal(err)}
    loaded,err:=Load(path)
    if err!=nil{t.Fatal(err)}
    got:=loaded.Find(Cartridge)
    if got==nil || got.X!=647.5 || got.Y!=244.5 {t.Fatalf("centro perdeu precisão: %+v",got)}
}
