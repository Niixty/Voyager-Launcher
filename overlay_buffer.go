//go:build windows

package main

import (
    "math"
    "unsafe"

    ll "voyager/native/internal/loadinglayout"
)

// The Hero, text and loading indicators are only repainted when their values
// change. A 60Hz frame restores the cached image and draws only the cartridge.
type overlayBackbuffer struct {
    dc, bitmap, previous uintptr
    staticDC, staticBitmap, staticPrevious uintptr
    width, height int
    valid bool
    cachedPercent int
    cachedPhase, cachedDetail string
    sprite, foreground *img
    x, w, h int
    baseY, scale float64
    motion rect
    cachedFailed bool
}

func createOverlaySurface(windowDC uintptr,w,h int)(uintptr,uintptr,uintptr) {
    dc:=call(pCreateCompatibleDC,windowDC)
    if dc==0{return 0,0,0}
    bmp:=call(pCreateCompatibleBitmap,windowDC,uintptr(w),uintptr(h))
    if bmp==0 {
        call(pDeleteDC,dc)
        return 0,0,0
    }
    previous:=call(pSelectObject,dc,bmp)
    if previous==0 || previous==^uintptr(0) {
        call(pDeleteObject,bmp)
        call(pDeleteDC,dc)
        return 0,0,0
    }
    return dc,bmp,previous
}

func (b *overlayBackbuffer) release() {
    if b==nil{return}
    b.sprite.close()
    b.foreground.close()
    if b.staticDC!=0 {
        call(pSelectObject,b.staticDC,b.staticPrevious)
        call(pDeleteObject,b.staticBitmap)
        call(pDeleteDC,b.staticDC)
    }
    if b.dc!=0 {
        call(pSelectObject,b.dc,b.previous)
        call(pDeleteObject,b.bitmap)
        call(pDeleteDC,b.dc)
    }
    *b=overlayBackbuffer{}
}
func (o *overlayState) dropBuffer(hwnd uintptr) {
    if o==nil || o.buffers==nil{return}
    if b:=o.buffers[hwnd];b!=nil{b.release();delete(o.buffers,hwnd)}
}
func (o *overlayState) freeBuffers(){
    if o==nil{return}
    for hwnd:=range o.buffers{o.dropBuffer(hwnd)}
}

var drawImageFloat = gdip.NewProc("GdipDrawImageRect")

func (b *overlayBackbuffer) prepareSprite(o *overlayState) {
    n:=o.layout.Find(ll.Cartridge)
    if n==nil || !n.Visible || o.cartridge==nil{return}
    s:=math.Min(2.6,math.Min(float64(b.width)/ll.VirtualWidth,float64(b.height)/ll.VirtualHeight))
    b.scale=s
    w:=int(math.Max(1,math.Round(n.W*s)))
    h:=int(math.Max(1,math.Round(n.H*s)))
    f:=math.Min(float64(w)/float64(o.cartridge.W),float64(h)/float64(o.cartridge.H))
    b.w=int(math.Max(1,float64(o.cartridge.W)*f))
    b.h=int(math.Max(1,float64(o.cartridge.H)*f))
    b.x=int(math.Round((float64(b.width)-ll.VirtualWidth*s)/2+n.X*s))+(w-b.w)/2
    b.baseY=(float64(b.height)-ll.VirtualHeight*s)/2+n.Y*s+float64((h-b.h)/2)
    b.motion=rect{int32(b.x-2),int32(math.Floor(b.baseY-11*s))-2,int32(b.x+b.w+2),int32(math.Ceil(b.baseY+11*s))+int32(b.h)+2}
    p,handle,err:=newBitmapGraphics(b.w,b.h)
    if err!=nil{return}
    p.clear(0)
    p.imageCover(o.cartridge,0,0,b.w,b.h)
    p.close()
    b.sprite=&img{handle,b.w,b.h}
}

func (o *overlayState) invalidateMotion(){
    for _,hw:=range o.windows{
        b:=o.buffers[hw]
        if b==nil || b.sprite==nil{invalid(hw);continue}
        call(pInvalidate,hw,uintptr(unsafe.Pointer(&b.motion)),0)
    }
}

func (o *overlayState) paintBuffered(hwnd,windowDC uintptr,width,height int,damage rect) {
    if o==nil || width<1 || height<1{return}
    if o.buffers==nil{o.buffers=make(map[uintptr]*overlayBackbuffer)}
    b:=o.buffers[hwnd]
    if b!=nil && (b.width!=width || b.height!=height){
        o.dropBuffer(hwnd)
        b=nil
    }
    if b==nil {
        dc,bmp,previous:=createOverlaySurface(windowDC,width,height)
        staticDC,staticBmp,staticPrevious:=createOverlaySurface(windowDC,width,height)
        if dc!=0 && staticDC!=0 {
            b=&overlayBackbuffer{
                dc:dc,bitmap:bmp,previous:previous,
                staticDC:staticDC,staticBitmap:staticBmp,
                staticPrevious:staticPrevious,width:width,height:height,
            }
            b.prepareSprite(o)
            o.buffers[hwnd]=b
        }else{
            if dc!=0{call(pSelectObject,dc,previous);call(pDeleteObject,bmp);call(pDeleteDC,dc)}
            if staticDC!=0{call(pSelectObject,staticDC,staticPrevious);call(pDeleteObject,staticBmp);call(pDeleteDC,staticDC)}
        }
    }
    if b==nil || (o.cartridge!=nil && o.layout.Find(ll.Cartridge)!=nil && o.layout.Find(ll.Cartridge).Visible && b.sprite==nil) {
        // If allocation fails, keep the full-loading behavior functional.
        p:=paintFromHDC(windowDC)
        drawOverlay(p,o.cfg,0,0,width,height,o.percent,o.phase,o.detail,
            o.cartridge,o.background,o.accent,o.cartridgeOffset(),o.layout,o.fail!="")
        p.close()
        return
    }

    failed:=o.fail!=""
    if !b.valid || b.cachedPercent!=o.percent || b.cachedPhase!=o.phase ||
        b.cachedDetail!=o.detail || b.cachedFailed!=failed {
        p:=paintFromHDC(b.staticDC)
        below:=o.layout
        if o.cartOverlapsForeground() {
            for i,n:=range o.layout.Elements {
                if n.ID!=ll.Cartridge{continue}
                above:=o.layout
                above.Elements=above.Elements[i+1:]
                fp,handle,err:=newBitmapGraphics(width,height)
                if err!=nil{
                    p.close()
                    p=paintFromHDC(windowDC)
                    drawOverlay(p,o.cfg,0,0,width,height,o.percent,o.phase,o.detail,o.cartridge,o.background,o.accent,o.cartridgeOffset(),o.layout,failed)
                    p.close()
                    return
                }
                drawOverlayPass(fp,o.cfg,0,0,width,height,o.percent,o.phase,o.detail,nil,nil,o.accent,0,above,failed,false)
                fp.close()
                b.foreground.close()
                b.foreground=&img{handle,width,height}
                below.Elements=below.Elements[:i]
                break
            }
        }
        drawOverlay(p,o.cfg,0,0,width,height,o.percent,o.phase,o.detail,nil,o.background,o.accent,0,below,failed)
        p.close()
        b.cachedPercent=o.percent
        b.cachedPhase=o.phase
        b.cachedDetail=o.detail
        b.cachedFailed=failed
        b.valid=true
    }
    const SRCCOPY=0x00CC0020
    x,y,w,h:=uintptr(damage.Left),uintptr(damage.Top),uintptr(damage.w()),uintptr(damage.h())
    call(pBitBlt,b.dc,x,y,w,h,b.staticDC,x,y,SRCCOPY)
    if b.sprite!=nil {
        p:=paintFromHDC(b.dc)
        call(pGdipSetInterpolationMode,p.g,3) // Bilinear translation of the already-sized sprite.
        bits:=func(v float64)uintptr{return uintptr(math.Float32bits(float32(v)))}
        call(drawImageFloat,p.g,b.sprite.handle,bits(float64(b.x)),bits(b.baseY+o.cartridgeOffset()*b.scale),bits(float64(b.w)),bits(float64(b.h)))
        if b.foreground!=nil {
            call(pGdipDrawImageRectRectI,p.g,b.foreground.handle,x,y,w,h,x,y,w,h,2,0,0,0)
        }
        p.close()
    }
    call(pBitBlt,windowDC,x,y,w,h,b.dc,x,y,SRCCOPY)
}

// Cache the usual disjoint layout. When a later layer crosses the floating cart,
// compose in document order so the editor, screenshot and stream agree.
func(o *overlayState) cartOverlapsForeground()bool{
 var cart *ll.Element
 for i:=range o.layout.Elements{n:=&o.layout.Elements[i];if n.ID==ll.Cartridge{if !n.Visible||o.cartridge==nil{return false};cart=n;continue};if cart!=nil && (n.Visible||o.fail!=""&&n.ID==ll.Phase) && n.X<cart.X+cart.W && n.X+n.W>cart.X && n.Y<cart.Y+cart.H+11 && n.Y+n.H>cart.Y-11{return true}}
 return false
}
