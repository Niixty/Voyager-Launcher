package loadingprogress

import (
 "image"
 "image/color"
 "testing"
)

func TestPremultipliedFramesAndSubpixelMovement(t *testing.T){
 src:=image.NewNRGBA(image.Rect(0,0,2,2))
 src.SetNRGBA(0,0,color.NRGBA{R:200,G:80,B:40,A:128})
 src.SetNRGBA(1,1,color.NRGBA{R:255,G:255,B:255,A:255})
 frames:=PremultipliedFrames(src,4)
 if len(frames)!=4 {t.Fatalf("expected 4 phases, got %d",len(frames))}
 for _,buf:=range frames {
  if len(buf)!=2*4*4 {t.Fatalf("unexpected frame length %d",len(buf))}
  for i:=0;i<len(buf);i+=4{
   if buf[i]>buf[i+3] || buf[i+1]>buf[i+3] || buf[i+2]>buf[i+3] {
    t.Fatalf("color not premultiplied at %d: %v",i,buf[i:i+4])
   }
  }
 }
 // Phase 0 has its first sprite pixel starting at y=1.
 if frames[0][(1*2+0)*4+3]!=128 {t.Fatal("wrong alpha at integer position")}
 // Phase 1 distributes the top pixel between adjacent rows.
 top:=frames[1][(1*2+0)*4+3]
 bottom:=frames[1][(2*2+0)*4+3]
 if top>=128 || top<=0 || bottom<=0 {t.Fatalf("no subpixel interpolation: %d %d",top,bottom)}
}
func TestInvalidFrames(t *testing.T) {
 if PremultipliedFrames(nil,4)!=nil {t.Fatal("nil input accepted")}
 if PremultipliedFrames(image.NewNRGBA(image.Rect(0,0,2,2)),0)!=nil {t.Fatal("zero phases accepted")}
}
