//go:build windows

package main

import (
 "fmt"
 "image"
 "image/color"
 "math"
 "unsafe"

 "voyager/native/internal/loadingprogress"
)

// A GDI+ bitmap is read once per display size, then recolored in memory only
// when the loading percentage changes. No PNG disk IO or full-screen redraws.
func loadingPixels(source *img,w,h int) (*image.NRGBA,error) {
 if source==nil||w<1||h<1{return nil,fmt.Errorf("imagem de loading ausente")}
 p,bmp,err:=newBitmapGraphics(w,h)
 if err!=nil{return nil,err}
 p.clear(0)
 p.imageCover(source,0,0,w,h)
 p.close()
 defer call(pGdipDisposeImage,bmp)
 var bits overlayBitmapData
 rc:=rect{0,0,int32(w),int32(h)}
 if status:=call(lockBitmapBits,bmp,uintptr(unsafe.Pointer(&rc)),1,0x26200A,uintptr(unsafe.Pointer(&bits)));status!=0 {
  return nil,fmt.Errorf("GDI+ BitmapLockBits falhou: %d",status)
 }
 defer call(unlockBitmapBits,bmp,uintptr(unsafe.Pointer(&bits)))
 if bits.Scan0==0{return nil,fmt.Errorf("buffer GDI+ vazio")}
 result:=image.NewNRGBA(image.Rect(0,0,w,h))
 for y:=0;y<h;y++{
  ptr:=uintptr(int64(bits.Scan0)+int64(y)*int64(bits.Stride))
  row:=unsafe.Slice((*byte)(unsafe.Pointer(ptr)),w*4)
  for x:=0;x<w;x++{
   si:=x*4;di:=y*result.Stride+x*4
   result.Pix[di]=row[si+2]
   result.Pix[di+1]=row[si+1]
   result.Pix[di+2]=row[si]
   result.Pix[di+3]=row[si+3]
  }
 }
 return result,nil
}

func loadingBitmap(pixels *image.NRGBA) (*img,error) {
 if pixels==nil{return nil,fmt.Errorf("pixels vazios")}
 w,h:=pixels.Rect.Dx(),pixels.Rect.Dy()
 if w<=0||h<=0{return nil,fmt.Errorf("dimensões inválidas")}
 p,bmp,err:=newBitmapGraphics(w,h)
 if err!=nil{return nil,err}
 p.close()
 rc:=rect{0,0,int32(w),int32(h)}
 var bits overlayBitmapData
 if status:=call(lockBitmapBits,bmp,uintptr(unsafe.Pointer(&rc)),2,0x26200A,uintptr(unsafe.Pointer(&bits)));status!=0 {
  call(pGdipDisposeImage,bmp)
  return nil,fmt.Errorf("GDI+ BitmapLockBits(W) falhou: %d",status)
 }
 if bits.Scan0==0{
  call(unlockBitmapBits,bmp,uintptr(unsafe.Pointer(&bits)))
  call(pGdipDisposeImage,bmp)
  return nil,fmt.Errorf("buffer GDI+ vazio")
 }
 for y:=0;y<h;y++ {
  ptr:=uintptr(int64(bits.Scan0)+int64(y)*int64(bits.Stride))
  row:=unsafe.Slice((*byte)(unsafe.Pointer(ptr)),w*4)
  for x:=0;x<w;x++ {
   si:=y*pixels.Stride+x*4;di:=x*4
   row[di]=pixels.Pix[si+2]
   row[di+1]=pixels.Pix[si+1]
   row[di+2]=pixels.Pix[si]
   row[di+3]=pixels.Pix[si+3]
  }
 }
 call(unlockBitmapBits,bmp,uintptr(unsafe.Pointer(&bits)))
 return &img{handle:bmp,W:w,H:h},nil
}

func loadingAccent(rgb uint32,failed bool) color.NRGBA {
 if failed {rgb=overlayTheme.Color("progressError")}
 if rgb==0 {rgb=0xffa4b3cd}
 return color.NRGBA{R:uint8(rgb>>16),G:uint8(rgb>>8),B:uint8(rgb),A:255}
}

func (p *painter) drawLoadingAsset(source *img,x,y,w,h,percent int,rgb uint32,failed bool) {
 if source==nil||w<=0||h<=0{return}
 f:=math.Min(float64(w)/float64(source.W),float64(h)/float64(source.H))
 sw:=int(math.Max(1,math.Round(float64(source.W)*f)))
 sh:=int(math.Max(1,math.Round(float64(source.H)*f)))
 sx:=x+(w-sw)/2;sy:=y+(h-sh)/2
 original,err:=loadingPixels(source,sw,sh)
 if err!=nil {p.imageContain(source,x,y,w,h);return}
 accent:=loadingAccent(rgb,failed)
 if percent>=95 {
  radius:=int(math.Max(10,math.Min(34,float64(sw)*0.07)))
  halo,err:=loadingBitmap(loadingprogress.Halo(original,radius,accent))
  if err==nil {
   // Halo behind silhouette. The double pass provides a brighter 100% finish.
   p.imageCover(halo,sx-radius,sy-radius,sw+radius*2,sh+radius*2)
   if percent==100 {p.imageCover(halo,sx-radius,sy-radius,sw+radius*2,sh+radius*2)}
   halo.close()
  }
 }
 tinted,err:=loadingBitmap(loadingprogress.Blend(original,percent,accent))
 if err!=nil {p.imageContain(source,x,y,w,h);return}
 p.imageCover(tinted,sx,sy,sw,sh)
 tinted.close()
}
