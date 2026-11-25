package heroaccent

import (
 "image"
 "image/color"
 "image/png"
 "os"
 "path/filepath"
 "testing"
)
func solid(c color.NRGBA) image.Image {
 im:=image.NewNRGBA(image.Rect(0,0,240,150))
 for y:=0;y<150;y++ {for x:=0;x<240;x++ {im.SetNRGBA(x,y,c)}}
 return im
}
func TestSolidHeroColors(t *testing.T){
 for _,example:=range []struct{name string;base color.NRGBA;dominant string}{
  {"red",color.NRGBA{R:200,G:38,B:44,A:255},"red"},
  {"green",color.NRGBA{R:30,G:190,B:66,A:255},"green"},
  {"blue",color.NRGBA{R:30,G:55,B:195,A:255},"blue"},
 } {
  got:=FromImage(solid(example.base))
  r,g,b:=int((got>>16)&255),int((got>>8)&255),int(got&255)
  switch example.dominant{
  case "red":if r<=g||r<=b{t.Fatalf("%s: #%06x",example.name,got)}
  case "green":if g<=r||g<=b{t.Fatalf("%s: #%06x",example.name,got)}
  case "blue":if b<=r||b<=g{t.Fatalf("%s: #%06x",example.name,got)}
  }
 }
}
func TestGreyscaleAndMissing(t *testing.T) {
 if FromImage(solid(color.NRGBA{R:80,G:80,B:80,A:255}))!=Neutral{t.Fatal("grey fallback")}
 if FromFile("missing-file.png")!=Neutral{t.Fatal("missing fallback")}
 if FromImage(nil)!=Neutral{t.Fatal("nil fallback")}
}
func TestDominantColorWins(t *testing.T) {
 im:=image.NewNRGBA(image.Rect(0,0,200,100))
 for y:=0;y<100;y++{for x:=0;x<200;x++{
  if x<160 {im.SetNRGBA(x,y,color.NRGBA{R:30,G:80,B:220,A:255})}else{im.SetNRGBA(x,y,color.NRGBA{R:230,G:60,B:15,A:255})}
 }}
 got:=FromImage(im)
 r,g,b:=int((got>>16)&255),int((got>>8)&255),int(got&255)
 if b<=r||b<=g {t.Fatalf("blue expected: #%08x",got)}
}
func TestMostFrequentColorNotMinoritySaturatedAccent(t *testing.T) {
    im:=image.NewNRGBA(image.Rect(0,0,200,100))
    blue:=color.NRGBA{R:98,G:119,B:151,A:255}
    red:=color.NRGBA{R:245,G:30,B:34,A:255}
    for y:=0;y<100;y++ {
        for x:=0;x<200;x++ {
            if x<130 {im.SetNRGBA(x,y,blue)} else {im.SetNRGBA(x,y,red)}
        }
    }
    got:=FromImage(im)
    r,g,b:=int((got>>16)&255),int((got>>8)&255),int(got&255)
    if b<=r || b<=g {t.Fatalf("cor mais frequente era azul, obtido #%08x",got)}
}
func TestPNGFile(t *testing.T) {
 path:=filepath.Join(t.TempDir(),"hero.png")
 f,err:=os.Create(path);if err!=nil{t.Fatal(err)}
 if err=png.Encode(f,solid(color.NRGBA{R:26,G:180,B:85,A:255}));err!=nil{t.Fatal(err)}
 if err=f.Close();err!=nil{t.Fatal(err)}
 got:=FromFile(path)
 if (got>>8)&255 <= (got>>16)&255{t.Fatalf("green expected: #%08x",got)}
}


func TestAdjustBrightnessIndependentOfDominantSelection(t *testing.T) {
    base:=uint32(0xff5080c0)
    if AdjustBrightness(base,100)!=base {t.Fatal("100% precisa preservar a Hero")}
    dark:=AdjustBrightness(base,50)
    if (dark&0xff)!=0x60 || ((dark>>16)&0xff)!=0x28 {t.Fatalf("escurecimento: %#x",dark)}
    light:=AdjustBrightness(base,150)
    if (light&0xff)<=0xc0 || ((light>>16)&0xff)<=0x50 {t.Fatalf("clareamento: %#x",light)}
    if AdjustBrightness(base,-1)!=0xff000000 {t.Fatal("0% é preto")}
    if AdjustBrightness(base,250)!=0xffffffff {t.Fatal("200% é branco")}
    if AdjustBrightness(0,100)!=Neutral {t.Fatal("fallback neutro")}
}
