//go:build windows

package main

import (
    "math"

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

func (o *overlayState) drawAnimatedCartridge(p *painter,width,height int) {
    if o.cartridge==nil || p==nil{return}
    node:=o.layout.Find(ll.Cartridge)
    if node==nil || !node.Visible{return}
    scale:=math.Min(float64(width)/ll.VirtualWidth,float64(height)/ll.VirtualHeight)
    if scale>2.6{scale=2.6}
    if scale<=0{return}
    left:=(float64(width)-ll.VirtualWidth*scale)/2
    top:=(float64(height)-ll.VirtualHeight*scale)/2
    x:=int(math.Round(left+node.X*scale))
    y:=int(math.Round(top+(node.Y+o.cartridgeOffset())*scale))
    w:=int(math.Max(1,math.Round(node.W*scale)))
    h:=int(math.Max(1,math.Round(node.H*scale)))
    p.imageContain(o.cartridge,x,y,w,h)
}

func (o *overlayState) paintBuffered(hwnd,windowDC uintptr,width,height int) {
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
            o.buffers[hwnd]=b
        }else{
            if dc!=0{call(pSelectObject,dc,previous);call(pDeleteObject,bmp);call(pDeleteDC,dc)}
            if staticDC!=0{call(pSelectObject,staticDC,staticPrevious);call(pDeleteObject,staticBmp);call(pDeleteDC,staticDC)}
        }
    }
    if b==nil {
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
        // Compose the static Hero, all text and the progress bar just once
        // per status change, with the moving cartridge omitted.
        drawOverlay(p,o.cfg,0,0,width,height,o.percent,o.phase,o.detail,
            nil,o.background,o.accent,0,o.layout,failed)
        p.close()
        b.cachedPercent=o.percent
        b.cachedPhase=o.phase
        b.cachedDetail=o.detail
        b.cachedFailed=failed
        b.valid=true
    }
    const SRCCOPY=0x00CC0020
    call(pBitBlt,b.dc,0,0,uintptr(width),uintptr(height),b.staticDC,0,0,SRCCOPY)
    if o.cartridge!=nil {
        p:=paintFromHDC(b.dc)
        o.drawAnimatedCartridge(p,width,height)
        p.close()
    }
    call(pBitBlt,windowDC,0,0,uintptr(width),uintptr(height),b.dc,0,0,SRCCOPY)
}
