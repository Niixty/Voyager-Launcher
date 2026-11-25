package loadingprogress

import (
 "image"
 "math"
)

// PremultipliedFrames converts a straight-alpha PNG into BGRA DIB rows with
// subpixel vertical phases. This is pure CPU work and can run off the Win32
// paint thread. Output rows include one transparent row above and below.
// All RGB values are premultiplied, as required by AlphaBlend/AC_SRC_ALPHA.
func PremultipliedFrames(src *image.NRGBA, phases int) [][]byte {
 if src==nil || phases<=0 || phases>16{return nil}
 w,h:=src.Bounds().Dx(),src.Bounds().Dy()
 if w<1 || h<1{return nil}
 outputs:=make([][]byte,phases)
 rowBytes:=w*4
 for phase:=0;phase<phases;phase++ {
  buf:=make([]byte,rowBytes*(h+2))
  offset:=float64(phase)/float64(phases)
  for dy:=0;dy<h+2;dy++ {
   y:=float64(dy)-1-offset
   y0:=int(math.Floor(y))
   fraction:=y-float64(y0)
   a0:=1-fraction
   a1:=fraction
   for x:=0;x<w;x++ {
    j:=dy*rowBytes+x*4
    // Alpha and colors are interpolated after premultiplication: doing
    // interpolation in straight RGBA would create dark PNG outlines.
    var alpha,red,green,blue float64
    for _,sample:=range []struct{sy int; weight float64}{{y0,a0},{y0+1,a1}} {
     if sample.sy<0 || sample.sy>=h || sample.weight<=0{continue}
     i:=sample.sy*src.Stride+x*4
     a:=float64(src.Pix[i+3])*sample.weight
     alpha+=a
     red+=float64(src.Pix[i])*a/255
     green+=float64(src.Pix[i+1])*a/255
     blue+=float64(src.Pix[i+2])*a/255
    }
    buf[j+0]=uint8(math.Min(255,math.Round(blue)))
    buf[j+1]=uint8(math.Min(255,math.Round(green)))
    buf[j+2]=uint8(math.Min(255,math.Round(red)))
    buf[j+3]=uint8(math.Min(255,math.Round(alpha)))
   }
  }
  outputs[phase]=buf
 }
 return outputs
}
