//go:build windows

package main

import (
    "fmt"
    "image"
    "math"
    "syscall"
    "time"
    "unsafe"

    "voyager/native/internal/loadingprogress"
    "voyager/native/internal/heroaccent"
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
    sprite, foreground, halo *img
    source *image.NRGBA
    glowRadius, spritePercent int
    spriteFailed bool
    alphaFrames [4]overlayAlphaFrame
    haloFrame overlayAlphaFrame
    alphaReady bool
    spriteRequests chan overlaySpriteRequest
    spriteResults chan overlaySpriteResult
    spriteDone chan struct{}
    spriteRequestedPercent int
    spriteRequestedFailed bool
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
    b.halo.close()
    if b.spriteDone!=nil {close(b.spriteDone);b.spriteDone=nil}
    b.haloFrame.release()
    for i:=range b.alphaFrames {b.alphaFrames[i].release()}
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
var (
    createDIBSection = g32.NewProc("CreateDIBSection")
    alphaBlend = syscall.NewLazyDLL("msimg32.dll").NewProc("AlphaBlend")
    lockBitmapBits = gdip.NewProc("GdipBitmapLockBits")
    unlockBitmapBits = gdip.NewProc("GdipBitmapUnlockBits")
)

// Cache four pre-multiplied alpha sprites with quarter-pixel offsets.
// Each animated frame uses a fast GDI AlphaBlend instead of costly GDI+
// image scaling; transparent cartridge backgrounds are preserved.
type overlayAlphaFrame struct {dc, bitmap, previous, pixels uintptr}
func (f *overlayAlphaFrame) release() {
    if f.dc!=0 {
        call(pSelectObject,f.dc,f.previous)
        call(pDeleteObject,f.bitmap)
        call(pDeleteDC,f.dc)
    }
    *f=overlayAlphaFrame{}
}
type overlayBitmapData struct {
    Width, Height uint32
    Stride, PixelFormat int32
    Scan0, Reserved uintptr
}
type overlayBitmapInfo struct {
    Size uint32
    Width, Height int32
    Planes, BitCount uint16
    Compression, SizeImage uint32
    XPelsPerMeter, YPelsPerMeter int32
    ClrUsed, ClrImportant uint32
}
func rasterizedAlphaFrame(source *img,w,h int,phase float64)(overlayAlphaFrame,bool){
    // One transparent row above and below accommodates fractional movement.
    p,bmp,err:=newBitmapGraphics(w,h+2)
    if err!=nil{return overlayAlphaFrame{},false}
    p.clear(0)
    bits:=func(v float64)uintptr{return uintptr(math.Float32bits(float32(v)))}
    status:=call(drawImageFloat,p.g,source.handle,bits(0),bits(1+phase),bits(float64(w)),bits(float64(h)))
    p.close()
    defer call(pGdipDisposeImage,bmp)
    if status!=0{return overlayAlphaFrame{},false}
    r:=rect{0,0,int32(w),int32(h+2)}
    var data overlayBitmapData
    // PixelFormat32bppPARGB requests premultiplied transparency from GDI+.
    if call(lockBitmapBits,bmp,uintptr(unsafe.Pointer(&r)),1,0xE200B,uintptr(unsafe.Pointer(&data)))!=0 {
        return overlayAlphaFrame{},false
    }
    defer call(unlockBitmapBits,bmp,uintptr(unsafe.Pointer(&data)))
    info:=overlayBitmapInfo{Size:40,Width:int32(w),Height:-int32(h+2),Planes:1,BitCount:32}
    var dest uintptr
    hbmp:=call(createDIBSection,0,uintptr(unsafe.Pointer(&info)),0,uintptr(unsafe.Pointer(&dest)),0,0)
    if hbmp==0||dest==0{return overlayAlphaFrame{},false}
    rowSize:=w*4
    for y:=0;y<h+2;y++ {
        src:=unsafe.Slice((*byte)(unsafe.Pointer(uintptr(int64(data.Scan0)+int64(y)*int64(data.Stride)))),rowSize)
        dst:=unsafe.Slice((*byte)(unsafe.Pointer(dest+uintptr(y*rowSize))),rowSize)
        copy(dst,src)
    }
    dc:=call(pCreateCompatibleDC,0)
    if dc==0{call(pDeleteObject,hbmp);return overlayAlphaFrame{},false}
    old:=call(pSelectObject,dc,hbmp)
    if old==0||old==^uintptr(0){
        call(pDeleteDC,dc);call(pDeleteObject,hbmp)
        return overlayAlphaFrame{},false
    }
    return overlayAlphaFrame{dc:dc,bitmap:hbmp,previous:old},true
}

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
    radius:=int(math.Max(10,math.Min(34,float64(b.w)*0.07)))
    b.glowRadius=radius
    marginX:=int(math.Max(50,float64(b.w)*0.20))
    marginY:=int(math.Max(50,float64(b.h)*0.20+11*s))
    b.motion=rect{int32(b.x-marginX),int32(math.Floor(b.baseY))-int32(marginY),int32(b.x+b.w+marginX),int32(math.Ceil(b.baseY))+int32(b.h)+int32(marginY)}
    source,err:=loadingPixels(o.cartridge,b.w,b.h)
    if err!=nil {
        // Keep the regular image as a safe fallback if pixel extraction fails.
        p,handle,e:=newBitmapGraphics(b.w,b.h)
        if e!=nil{return}
        p.clear(0)
        p.imageCover(o.cartridge,0,0,b.w,b.h)
        p.close()
        b.sprite=&img{handle:handle,W:b.w,H:b.h}
        return
    }
    b.source=source
    cartAccent:=heroaccent.AdjustBrightness(o.accent,o.layout.CartridgeBrightnessLevel())
    accent:=loadingAccent(cartAccent,false)
    if glow,e:=loadingBitmap(loadingprogress.Halo(source,radius,accent));e==nil {
        b.halo=glow
        // Pre-render the silhouette glow once; Win32 AlphaBlend will vary
        // only opacity each frame, instead of creating GDI+ image contexts.
        if frame,ok:=rasterizedAlphaFrame(glow,glow.W,glow.H,0);ok {
            b.haloFrame=frame
        }
    }
    // Stable backbuffer even when async tint calculation has not completed.
    if image,e:=loadingBitmap(loadingprogress.Blend(source,o.percent,accent));e==nil {b.sprite=image}
    for i:=range b.alphaFrames {
        frame,ok:=createReusableAlphaFrame(b.w,b.h)
        if !ok {
            for j:=range b.alphaFrames {b.alphaFrames[j].release()}
            return // The GDI+ fallback still renders the original sprite.
        }
        b.alphaFrames[i]=frame
    }
    b.startSpriteWorker(cartAccent)
    b.updateSprite(o)
}

// The tinted sprite is calculated outside WM_PAINT, and each phase is copied
// into a reusable top-down DIB only once the worker delivers completed pixels.
type overlaySpriteRequest struct {percent int; failed bool}
type overlaySpriteResult struct {
    percent int
    failed bool
    frames [][]byte
}

func createReusableAlphaFrame(w,h int)(overlayAlphaFrame,bool) {
    info:=overlayBitmapInfo{Size:40,Width:int32(w),Height:-int32(h+2),Planes:1,BitCount:32}
    var memory uintptr
    bitmap:=call(createDIBSection,0,uintptr(unsafe.Pointer(&info)),0,uintptr(unsafe.Pointer(&memory)),0,0)
    if bitmap==0||memory==0{return overlayAlphaFrame{},false}
    dc:=call(pCreateCompatibleDC,0)
    if dc==0 {call(pDeleteObject,bitmap);return overlayAlphaFrame{},false}
    previous:=call(pSelectObject,dc,bitmap)
    if previous==0||previous==^uintptr(0) {
        call(pDeleteDC,dc)
        call(pDeleteObject,bitmap)
        return overlayAlphaFrame{},false
    }
    return overlayAlphaFrame{dc:dc,bitmap:bitmap,previous:previous,pixels:memory},true
}

func (b *overlayBackbuffer) startSpriteWorker(accent uint32) {
    if b.source==nil || b.spriteRequests!=nil{return}
    b.spriteRequests=make(chan overlaySpriteRequest,1)
    b.spriteResults=make(chan overlaySpriteResult,1)
    b.spriteDone=make(chan struct{})
    b.spriteRequestedPercent=-1
    source:=b.source // immutable for the worker's lifetime
    requests:=b.spriteRequests
    results:=b.spriteResults
    done:=b.spriteDone
    phases:=len(b.alphaFrames)
    go func(){
        for {
            var request overlaySpriteRequest
            select {
            case <-done:return
            case request=<-requests:
            }
            // Drop intermediate percentages. Only the newest tint matters;
            // animation frames themselves still run at display refresh rate.
            for {
                select {
                case request=<-requests:
                default:goto prepare
                }
            }
        prepare:
            tinted:=loadingprogress.Blend(source,request.percent,loadingAccent(accent,request.failed))
            frames:=loadingprogress.PremultipliedFrames(tinted,phases)
            if len(frames)!=phases{continue}
            ready:=overlaySpriteResult{percent:request.percent,failed:request.failed,frames:frames}
            select {
            case <-done:return
            case results<-ready:
            default:
                select{case <-results:default:}
                select{case <-done:return;case results<-ready:default:}
            }
        }
    }()
}

func (b *overlayBackbuffer) updateSprite(o *overlayState) {
    if b.spriteRequests==nil{return}
    request:=overlaySpriteRequest{percent:o.percent,failed:o.fail!=""}
    if request.percent!=b.spriteRequestedPercent || request.failed!=b.spriteRequestedFailed {
        b.spriteRequestedPercent=request.percent
        b.spriteRequestedFailed=request.failed
        select {
        case b.spriteRequests<-request:
        default:
            select{case <-b.spriteRequests:default:}
            select{case b.spriteRequests<-request:default:}
        }
    }
    // A ready image replaces the active sprite without any GDI+ rasterization.
    // Copying into preallocated DIB memory is much faster than recreating 4
    // GDI+ graphics contexts and their AlphaBlend sources for every 4%.
    for {
        select {
        case next:=<-b.spriteResults:
            if next.percent!=o.percent || next.failed!=(o.fail!="") {continue}
            expected:=b.w*(b.h+2)*4
            if len(next.frames)!=len(b.alphaFrames){continue}
            valid:=true
            for i,frame:=range b.alphaFrames {
                if frame.pixels==0 || len(next.frames[i])!=expected {valid=false;break}
            }
            if !valid{continue}
            for i,frame:=range b.alphaFrames {
                dest:=unsafe.Slice((*byte)(unsafe.Pointer(frame.pixels)),expected)
                copy(dest,next.frames[i])
            }
            b.alphaReady=true
            b.spritePercent=next.percent
            b.spriteFailed=next.failed
        default:return
        }
    }
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
    b.updateSprite(o)
    if !b.valid || b.cachedPhase!=o.phase ||
        b.cachedDetail!=o.detail || b.cachedFailed!=failed {
        p:=paintFromHDC(b.staticDC)
        below:=o.layout.Clone()
        below.Elements=withoutDynamicPercent(below.Elements)
        if o.cartOverlapsForeground() {
            for i,n:=range o.layout.Elements {
                if n.ID!=ll.Cartridge{continue}
                above:=o.layout.Clone()
                above.Elements=withoutDynamicPercent(above.Elements[i+1:])
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
                for j,entry:=range below.Elements {
                    if entry.ID==ll.Cartridge {below.Elements=below.Elements[:j];break}
                }
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
        offset,scale:=o.loadingMotion(time.Now())
        position:=b.baseY+offset*b.scale
        drawW:=float64(b.w)*scale
        drawH:=float64(b.h)*scale
        drawX:=float64(b.x)-(drawW-float64(b.w))/2
        drawY:=position-(drawH-float64(b.h))/2
        bits:=func(v float64)uintptr{return uintptr(math.Float32bits(float32(v)))}
        if b.halo!=nil {
            // Start with a barely visible glow at 0% and increase it
            // continuously. The alpha mask follows the PNG silhouette.
            amount:=math.Max(0,math.Min(1,float64(o.percent)/100))
            alpha:=int(22+210*math.Pow(amount,1.35))
            if alpha>255 {alpha=255}
            radius:=float64(b.glowRadius)*scale
            gx:=int(math.Round(drawX-radius))
            gy:=int(math.Round(drawY-radius))
            gw:=int(math.Max(1,math.Round(drawW+2*radius)))
            gh:=int(math.Max(1,math.Round(drawH+2*radius)))
            if b.haloFrame.dc!=0 {
                blend:=uintptr(0x01000000|uint32(alpha)<<16)
                call(alphaBlend,b.dc,uintptr(gx),uintptr(gy-1),uintptr(gw),uintptr(gh+2),
                    b.haloFrame.dc,0,0,uintptr(b.halo.W),uintptr(b.halo.H+2),blend)
            } else {
                p:=paintFromHDC(b.dc)
                call(drawImageFloat,p.g,b.halo.handle,bits(float64(gx)),bits(float64(gy)),bits(float64(gw)),bits(float64(gh)))
                p.close()
            }
        }
        usedFastPath:=false
        if b.alphaReady {
            base:=math.Floor(drawY)
            fraction:=drawY-base
            phase:=int(fraction*float64(len(b.alphaFrames)))
            if phase<0{phase=0}
            if phase>=len(b.alphaFrames){phase=len(b.alphaFrames)-1}
            source:=b.alphaFrames[phase]
            dw:=int(math.Max(1,math.Round(drawW)))
            dh:=int(math.Max(1,math.Round(drawH)))+2
            usedFastPath=call(alphaBlend,b.dc,uintptr(int(math.Round(drawX))),uintptr(int(base)-1),
                uintptr(dw),uintptr(dh),source.dc,0,0,uintptr(b.w),
                uintptr(b.h+2),0x01FF0000)!=0
        }
        if !usedFastPath || b.foreground!=nil {
            p:=paintFromHDC(b.dc)
            if !usedFastPath {
                call(pGdipSetInterpolationMode,p.g,3)
                call(drawImageFloat,p.g,b.sprite.handle,bits(drawX),bits(drawY),bits(drawW),bits(drawH))
            }
            if b.foreground!=nil {
                call(pGdipDrawImageRectRectI,p.g,b.foreground.handle,x,y,w,h,x,y,w,h,2,0,0,0)
            }
            p.close()
        }
    }
    percentArea:=o.percentageRect(width,height)
    if intersectsRect(damage,percentArea) && percentArea.w()>0 {
        p:=paintFromHDC(b.dc)
        o.drawDynamicPercent(p,width,height)
        p.close()
    }
    call(pBitBlt,windowDC,x,y,w,h,b.dc,x,y,SRCCOPY)
}


func withoutDynamicPercent(elements []ll.Element) []ll.Element {
    kept:=make([]ll.Element,0,len(elements))
    for _,element:=range elements {if element.ID!=ll.Percentage{kept=append(kept,element)}}
    return kept
}
func intersectsRect(a,b rect)bool{
    return a.Left<b.Right && a.Right>b.Left && a.Top<b.Bottom && a.Bottom>b.Top
}
func (o *overlayState) percentageRect(width,height int) rect {
    n:=o.layout.Find(ll.Percentage)
    if n==nil||!n.Visible||!appPrefs.ShowPercentage{return rect{}}
    scale:=math.Min(2.6,math.Min(float64(width)/ll.VirtualWidth,float64(height)/ll.VirtualHeight))
    left:=(float64(width)-ll.VirtualWidth*scale)/2
    top:=(float64(height)-ll.VirtualHeight*scale)/2
    x:=int(math.Round(left+n.X*scale))
    y:=int(math.Round(top+n.Y*scale))
    w:=int(math.Max(1,math.Round(n.W*scale)))
    h:=int(math.Max(1,math.Round(n.H*scale)))
    return rect{Left:int32(x-2),Top:int32(y-2),Right:int32(x+w+2),Bottom:int32(y+h+2)}
}
func (o *overlayState) drawDynamicPercent(p *painter,width,height int) {
    n:=o.layout.Find(ll.Percentage)
    if n==nil||!n.Visible||!appPrefs.ShowPercentage{return}
    area:=o.percentageRect(width,height)
    scale:=math.Min(2.6,math.Min(float64(width)/ll.VirtualWidth,float64(height)/ll.VirtualHeight))
    color:=overlayTheme.Color("percentage")
    switch n.Color {
    case "white":color=overlayTheme.Color("gameTitle")
    case "muted":color=overlayTheme.Color("status")
    case "hero":color=o.accent
    }
    p.textAligned(int(area.Left)+2,int(area.Top)+2,int(area.w())-4,int(area.h())-4,
        fmt.Sprintf("%d%%",o.percent),n.FontSize*float32(scale),
        palette(color),false,n.TextAlign,n.VerticalAlign)
}
func (o *overlayState) invalidateProgress() {
    for _,hw:=range o.windows {
        b:=o.buffers[hw]
        if b==nil {invalid(hw);continue}
        if b.sprite!=nil {call(pInvalidate,hw,uintptr(unsafe.Pointer(&b.motion)),0)}
        area:=o.percentageRect(b.width,b.height)
        if area.w()>0 && area.h()>0 {
            call(pInvalidate,hw,uintptr(unsafe.Pointer(&area)),0)
        }
    }
}

// Cache the usual disjoint layout. When a later layer crosses the floating cart,
// compose in document order so the editor, screenshot and stream agree.
func(o *overlayState) cartOverlapsForeground()bool{
 var cart *ll.Element
 for i:=range o.layout.Elements{
    n:=&o.layout.Elements[i]
    if n.ID==ll.Cartridge{
        if !n.Visible || o.cartridge==nil{return false}
        cart=n
        continue
    }
    // Percentage is painted dynamically after the sprite, not baked into
    // a full-screen foreground layer. This keeps the 60 Hz path lightweight.
    if n.ID==ll.Percentage {continue}
    if cart!=nil && (n.Visible || o.fail!="" && n.ID==ll.Phase) &&
        n.X<cart.X+cart.W && n.X+n.W>cart.X &&
        n.Y<cart.Y+cart.H+11 && n.Y+n.H>cart.Y-11 {return true}
}
 return false
}
