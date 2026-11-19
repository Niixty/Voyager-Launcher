//go:build windows

package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"syscall"
 "strings"
	"unsafe"
)

type rectF struct{ X, Y, W, H float32 }
type startupInput struct {
	Version                uint32
	Debug                  uintptr
	SuppressBGThread       int32
	SuppressExternalCodecs int32
}
type palette uint32

const (
	cBG     palette = 0xff0b0b0d
	cPanel  palette = 0xff151518
	cSide   palette = 0xff141416
	cField  palette = 0xff222226
	cEdge   palette = 0xff35353a
	cWhite  palette = 0xfff4f4f6
	cMuted  palette = 0xff9e9ea8
	cAccent palette = 0xffbabec8
	cButton palette = 0xff303036
)

var gdipToken uintptr

func initGraphics() error {
	in := startupInput{Version: 1}
	status := call(pGdipStartup, uintptr(unsafe.Pointer(&gdipToken)), uintptr(unsafe.Pointer(&in)), 0)
	if status != 0 {
		return fmt.Errorf("GDI+ falhou: %d", status)
	}
	return nil
}
func stopGraphics() {
	if gdipToken != 0 {
		call(pGdipShutdown, gdipToken)
	}
}
func rgba(r, g, b int) palette { return palette(0xff000000 | r<<16 | g<<8 | b) }

type img struct {
	handle uintptr
	W, H   int
}

func loadImg(path string) *img {
	if path == "" {
		return nil
	}
	if _, e := os.Stat(path); e != nil {
		return nil
	}
	var h uintptr
	if call(pGdipLoadImage, uintptr(wide(path)), uintptr(unsafe.Pointer(&h))) != 0 || h == 0 {
		return nil
	}
	var w, hh uint32
	call(pGdipGetImageWidth, h, uintptr(unsafe.Pointer(&w)))
	call(pGdipGetImageHeight, h, uintptr(unsafe.Pointer(&hh)))
	if w == 0 || hh == 0 {
		call(pGdipDisposeImage, h)
		return nil
	}
	return &img{handle: h, W: int(w), H: int(hh)}
}
func (i *img) close() {
	if i != nil && i.handle != 0 {
		call(pGdipDisposeImage, i.handle)
		i.handle = 0
	}
}

type painter struct{ g uintptr }

func paintFromHDC(hdc uintptr) *painter {
	var g uintptr
	call(pGdipGraphicsFromHDC, hdc, uintptr(unsafe.Pointer(&g)))
	p := &painter{g}
	p.smooth()
	return p
}
func (p *painter) smooth() {
	call(pGdipSetSmoothingMode, p.g, 4)
	call(pGdipSetInterpolationMode, p.g, 7)
	call(pGdipSetCompositingQuality, p.g, 4)
	call(pGdipSetTextRenderingHint, p.g, 4)
}
func (p *painter) close() {
	if p.g != 0 {
		call(pGdipDeleteGraphics, p.g)
		p.g = 0
	}
}
func (p *painter) clear(c palette) { call(pGdipGraphicsClear, p.g, uintptr(c)) }
func (p *painter) fill(x, y, w, h int, c palette) {
	if w <= 0 || h <= 0 {
		return
	}
	var b uintptr
	if call(pGdipCreateSolidFill, uintptr(c), uintptr(unsafe.Pointer(&b))) != 0 {
		return
	}
	call(pGdipFillRectangle, p.g, b, uintptr(x), uintptr(y), uintptr(w), uintptr(h))
	call(pGdipDeleteBrush, b)
}
func (p *painter) ellipse(x, y, w, h int, c palette) {
	if w <= 0 || h <= 0 {
		return
	}
	var b uintptr
	call(pGdipCreateSolidFill, uintptr(c), uintptr(unsafe.Pointer(&b)))
	call(pGdipFillEllipse, p.g, b, uintptr(x), uintptr(y), uintptr(w), uintptr(h))
	call(pGdipDeleteBrush, b)
}
func (p *painter) text(x, y, w, h int, s string, size float32, c palette, bold bool) {
	if s == "" || w <= 0 || h <= 0 {
		return
	}
	var family, font, brush uintptr
	call(pGdipCreateFontFamily, uintptr(wide("Segoe UI")), 0, uintptr(unsafe.Pointer(&family)))
	if family == 0 {
		return
	}
	style := uintptr(0)
	if bold {
		style = 1
	}
	call(pGdipCreateFont, family, uintptr(math.Float32bits(size)), style, 2, uintptr(unsafe.Pointer(&font)))
	// GDI+ GdipCreateFont takes a floating point value in the second slot (via 32-bit float bits in integer register).
	if font == 0 {
		call(pGdipDeleteFontFamily, family)
		return
	}
	call(pGdipCreateSolidFill, uintptr(c), uintptr(unsafe.Pointer(&brush)))
	t := syscall.StringToUTF16(strings.ReplaceAll(s,"\x00","�"))
	rc := rectF{float32(x), float32(y), float32(w), float32(h)}
	call(pGdipDrawString, p.g, uintptr(unsafe.Pointer(&t[0])), uintptr(len(t)-1), font, uintptr(unsafe.Pointer(&rc)), 0, brush)
	call(pGdipDeleteBrush, brush)
	call(pGdipDeleteFont, font)
	call(pGdipDeleteFontFamily, family)
}
func (p *painter) imageCover(im *img, x, y, w, h int) {
	if im == nil || w <= 0 || h <= 0 {
		return
	}
	sx, sy, sw, sh := 0, 0, im.W, im.H
	want := float64(w) / float64(h)
	src := float64(im.W) / float64(im.H)
	if src > want {
		sw = int(math.Round(float64(im.H) * want))
		sx = (im.W - sw) / 2
	} else {
		sh = int(math.Round(float64(im.W) / want))
		sy = (im.H - sh) / 2
	}
	if sw < 1 || sh < 1 {
		return
	}
	call(pGdipDrawImageRectRectI, p.g, im.handle, uintptr(x), uintptr(y), uintptr(w), uintptr(h), uintptr(sx), uintptr(sy), uintptr(sw), uintptr(sh), 2, 0, 0, 0)
}
func (p *painter) imageContain(im *img, x, y, w, h int) {
	if im == nil {
		return
	}
	f := math.Min(float64(w)/float64(im.W), float64(h)/float64(im.H))
	ww := int(float64(im.W) * f)
	hh := int(float64(im.H) * f)
	p.imageCover(im, x+(w-ww)/2, y+(h-hh)/2, ww, hh)
}
func newBitmapGraphics(w, h int) (*painter, uintptr, error) {
	var bmp, gra uintptr
	if w <= 0 || h <= 0 {
		return nil, 0, fmt.Errorf("dimensões inválidas")
	}
	// PixelFormat32bppARGB 0x26200A.
	s := call(pGdipCreateBitmapFromScan0, uintptr(w), uintptr(h), 0, 0x26200A, 0, uintptr(unsafe.Pointer(&bmp)))
	if s != 0 {
		return nil, 0, fmt.Errorf("GDI+ não criou bitmap (%d)", s)
	}
	s = call(pGdipGetImageGraphicsContext, bmp, uintptr(unsafe.Pointer(&gra)))
	if s != 0 {
		call(pGdipDisposeImage, bmp)
		return nil, 0, fmt.Errorf("GDI+ não criou contexto (%d)", s)
	}
	p := &painter{gra}
	p.smooth()
	return p, bmp, nil
}
func savePNG(bmp uintptr, dest string) error {
	encoder := guid{0x557cf406, 0x1a04, 0x11d3, [8]byte{0x9a, 0x73, 0, 0, 0xf8, 0x1e, 0xf3, 0x2e}}
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	rc := call(pGdipSaveImage, bmp, uintptr(wide(dest)), uintptr(unsafe.Pointer(&encoder)), 0)
	if rc != 0 {
		return fmt.Errorf("falha ao gravar PNG: status %d", rc)
	}
	return nil
}
func cropToPNG(src, dst string, w, h int) error {
	if src == "" {
		return nil
	}
	if absPath(src) == absPath(dst) {
		return nil
	}
	im := loadImg(src)
	if im == nil {
		return fmt.Errorf("imagem inválida: %s", src)
	}
	defer im.close()
	p, bmp, err := newBitmapGraphics(w, h)
	if err != nil {
		return err
	}
	p.clear(0xff111114)
	p.imageCover(im, 0, 0, w, h)
	p.close()
	defer call(pGdipDisposeImage, bmp)
	tmp := dst + ".tmp.png"
	defer os.Remove(tmp)
	if err := savePNG(bmp, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}
// fitTransparentPNG contains the cartridge without cropping, retaining alpha.
func fitTransparentPNG(src, dst string, w, h int) error {
	if src == "" { return nil }
	if absPath(src) == absPath(dst) { return nil }
	im := loadImg(src)
	if im == nil { return fmt.Errorf("imagem Cartridge inválida: %s", src) }
	defer im.close()
	p, bmp, err := newBitmapGraphics(w, h)
	if err != nil { return err }
	p.clear(0x00000000)
	p.imageContain(im, 0, 0, w, h)
	p.close()
	defer call(pGdipDisposeImage, bmp)
	tmp := dst + ".tmp.png"
	defer os.Remove(tmp)
	if err := savePNG(bmp, tmp); err != nil { return err }
	return os.Rename(tmp, dst)
}

func absPath(s string) string {
	p, e := filepath.Abs(s)
	if e != nil {
		return s
	}
	return p
}
