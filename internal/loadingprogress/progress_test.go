package loadingprogress

import (
 "image"
 "image/color"
 "testing"
)

func sample() *image.NRGBA {
 p:=image.NewNRGBA(image.Rect(0,0,10,5))
 for y:=1;y<4;y++ {for x:=1;x<9;x++ {
  if x==4&&y==2 {continue}
  p.SetNRGBA(x,y,color.NRGBA{R:115,G:115,B:115,A:200})
 }}
 return p
}
func TestTintRespectsTransparency(t *testing.T) {
 src:=sample()
 accent:=color.NRGBA{R:210,G:45,B:10,A:255}
 for _,pct:=range []int{0,1,50,95,100}{
  dst:=Blend(src,pct,accent)
  for y:=0;y<5;y++ {for x:=0;x<10;x++ {
   i:=y*dst.Stride+x*4
   if dst.Pix[i+3]!=src.Pix[i+3] {t.Fatalf("alpha changed at %d%% (%d,%d)",pct,x,y)}
   if src.Pix[i+3]==0 && (dst.Pix[i]!=0||dst.Pix[i+1]!=0||dst.Pix[i+2]!=0) {
    t.Fatalf("transparent pixel tinted at %d%% (%d,%d)",pct,x,y)
   }
  }}
 }
}
func TestFillAdvancesHorizontal(t *testing.T) {
 src:=sample()
 accent:=color.NRGBA{R:255,G:40,B:40,A:255}
 at0:=Blend(src,0,accent)
 at50:=Blend(src,50,accent)
 at100:=Blend(src,100,accent)
 if !(at50.NRGBAAt(2,2).R>at50.NRGBAAt(7,2).R) {t.Fatal("left half should be tinted first")}
 if !(at0.NRGBAAt(2,2).R<at100.NRGBAAt(2,2).R) {t.Fatal("100% should brighten")}
 if at100.NRGBAAt(7,2).R<=at50.NRGBAAt(7,2).R {t.Fatal("right half must be tinted at 100%")}
}
func TestHaloFollowsMask(t *testing.T) {
 src:=sample()
 h:=Halo(src,9,color.NRGBA{R:255,G:80,B:20,A:255})
 if h.Bounds().Dx()!=28||h.Bounds().Dy()!=23 {t.Fatal("halo dimensions")}
 if h.NRGBAAt(0,0).A!=0 {t.Fatal("halo leaked into transparent outer corner")}
 if h.NRGBAAt(10,5).A==0 {t.Fatal("glow absent around silhouette")}
}
func TestClamp(t *testing.T) {
 src:=sample()
 a:=color.NRGBA{R:255,A:255}
 less:=Blend(src,-500,a);zero:=Blend(src,0,a)
 more:=Blend(src,10000,a);full:=Blend(src,100,a)
 if !equal(less,zero)||!equal(more,full){t.Fatal("percent not clamped")}
}
func equal(a,b *image.NRGBA) bool {
 if len(a.Pix)!=len(b.Pix){return false}
 for i:=range a.Pix{if a.Pix[i]!=b.Pix[i]{return false}}
 return true
}

func TestFeatheredBoundary(t *testing.T) {
    src:=image.NewNRGBA(image.Rect(0,0,120,3))
    for y:=0;y<3;y++ {for x:=0;x<120;x++ {
        src.SetNRGBA(x,y,color.NRGBA{R:110,G:110,B:110,A:255})
    }}
    got:=Blend(src,50,color.NRGBA{R:255,G:45,B:45,A:255})
    // The transition must span several pixels, not create a sudden hard cut.
    values:=[]int{}
    for _,x:=range []int{50,54,58,62,66,70}{
        values=append(values,int(got.NRGBAAt(x,1).R))
    }
    distinct:=0
    for i:=1;i<len(values);i++ {
        if values[i]>values[i-1] {t.Fatalf("red channel reverses at %d: %v",i,values)}
        if values[i]!=values[i-1]{distinct++}
    }
    if distinct<3{t.Fatalf("expected a gradual feathered transition: %v",values)}
}
