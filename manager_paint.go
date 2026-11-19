//go:build windows

package main

// Render off-screen and blit one complete frame. Native EDIT child windows
// continue to draw normally; the manager background no longer flickers.
func (m *managerState) paintBuffered(windowDC uintptr) {
	if m.windowW < 1 || m.windowH < 1 { return }
	memDC := call(pCreateCompatibleDC, windowDC)
	if memDC == 0 {
		p := paintFromHDC(windowDC); m.paint(p); p.close(); return
	}
	defer call(pDeleteDC, memDC)
	bitmap := call(pCreateCompatibleBitmap, windowDC, uintptr(m.windowW), uintptr(m.windowH))
	if bitmap == 0 {
		p := paintFromHDC(windowDC); m.paint(p); p.close(); return
	}
	defer call(pDeleteObject, bitmap)
	old := call(pSelectObject, memDC, bitmap)
	if old == 0 || old == ^uintptr(0) {
		p := paintFromHDC(windowDC); m.paint(p); p.close(); return
	}
	defer call(pSelectObject, memDC, old)
	p := paintFromHDC(memDC)
	m.paint(p)
	p.close()
	const SRCCOPY = 0x00CC0020
	call(pBitBlt, windowDC, 0, 0, uintptr(m.windowW), uintptr(m.windowH), memDC, 0, 0, SRCCOPY)
}
