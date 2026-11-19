//go:build windows

package main

// Each loading window paints into its own reusable off-screen bitmap.
// The completed frame is presented in one BitBlt, eliminating partially
// cleared backgrounds/text during 40ms Cartridge animation ticks.
type overlayBackbuffer struct {
    dc uintptr
    bitmap uintptr
    previous uintptr
    width, height int
}

func (b *overlayBackbuffer) release() {
    if b == nil { return }
    if b.dc != 0 {
        if b.previous != 0 && b.previous != ^uintptr(0) {
            call(pSelectObject, b.dc, b.previous)
        }
        if b.bitmap != 0 {call(pDeleteObject,b.bitmap)}
        call(pDeleteDC,b.dc)
    }
    *b=overlayBackbuffer{}
}

func (o *overlayState) dropBuffer(hwnd uintptr) {
    if o == nil || o.buffers == nil {return}
    if b:=o.buffers[hwnd];b!=nil {b.release();delete(o.buffers,hwnd)}
}
func (o *overlayState) freeBuffers(){
    if o==nil{return}
    for hwnd := range o.buffers {o.dropBuffer(hwnd)}
}
func (o *overlayState) paintBuffered(hwnd,hdc uintptr, width,height int) {
    if o==nil || width<1 || height<1{return}
    if o.buffers==nil{o.buffers=make(map[uintptr]*overlayBackbuffer)}
    back:=o.buffers[hwnd]
    if back!=nil && (back.width!=width || back.height!=height) {
        o.dropBuffer(hwnd)
        back=nil
    }
    if back==nil {
        dc:=call(pCreateCompatibleDC,hdc)
        if dc!=0 {
            bitmap:=call(pCreateCompatibleBitmap,hdc,uintptr(width),uintptr(height))
            if bitmap!=0 {
                previous:=call(pSelectObject,dc,bitmap)
                if previous!=0 && previous!=^uintptr(0) {
                    back=&overlayBackbuffer{dc:dc,bitmap:bitmap,previous:previous,width:width,height:height}
                    o.buffers[hwnd]=back
                } else {
                    call(pDeleteObject,bitmap)
                    call(pDeleteDC,dc)
                }
            } else {call(pDeleteDC,dc)}
        }
    }
    targetDC:=hdc
    if back!=nil{targetDC=back.dc}
    painter:=paintFromHDC(targetDC)
    if painter==nil{return}
    drawOverlay(painter,o.cfg,0,0,width,height,o.percent,o.phase,o.detail,
        o.cartridge,o.background,o.accent,o.cartridgeOffset(),o.layout,o.fail!="")
    painter.close()
    if back!=nil {
        const SRCCOPY = 0x00CC0020
        call(pBitBlt,hdc,0,0,uintptr(width),uintptr(height),back.dc,0,0,SRCCOPY)
    }
}
