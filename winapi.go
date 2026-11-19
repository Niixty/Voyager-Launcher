//go:build windows

package main

import (
	"fmt"
	"runtime"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	WS_OVERLAPPEDWINDOW               = 0x00CF0000
	WS_POPUP                          = 0x80000000
	WS_VISIBLE                        = 0x10000000
	WS_CHILD                          = 0x40000000
	WS_CLIPSIBLINGS                   = 0x04000000
	WS_CLIPCHILDREN                   = 0x02000000
	WS_EX_TOPMOST                     = 0x00000008
	WS_EX_TOOLWINDOW                  = 0x00000080
	WS_EX_NOACTIVATE                  = 0x08000000
	WS_EX_LAYERED                     = 0x00080000
	WS_TABSTOP                        = 0x00010000
	ES_AUTOHSCROLL                    = 0x0080
	ES_MULTILINE                      = 4
	ES_AUTOVSCROLL                    = 0x0040
	ES_READONLY                       = 0x0800
	WS_VSCROLL                        = 0x00200000
	WM_VOYAGER_UPDATE                 = 0x8001
	WM_DESTROY                        = 2
	WM_CREATE                         = 1
	WM_CLOSE                          = 0x10
	WM_PAINT                          = 0xF
	WM_SIZE                           = 5
	WM_COMMAND                        = 0x111
	WM_LBUTTONDOWN                    = 0x201
	WM_MOUSEWHEEL                     = 0x20A
	WM_CTLCOLORSTATIC                 = 0x138
	WM_CTLCOLOREDIT                   = 0x133
	WM_SETFONT                        = 0x30
	WM_GETMINMAXINFO                  = 0x24
	SW_SHOW                           = 5
	SW_HIDE                           = 0
	SW_RESTORE                        = 9
	SWP_NOSIZE                        = 1
	SWP_NOMOVE                        = 2
	SWP_NOACTIVATE                    = 0x10
	SWP_SHOWWINDOW                    = 0x40
	HWND_TOPMOST                      = ^uintptr(0)
	HWND_NOTOPMOST                    = ^uintptr(1)
	GWL_EXSTYLE                       = -20
	IDC_ARROW                         = 32512
	COLOR_WINDOW                      = 5
	DT_LEFT                           = 0
	DT_VCENTER                        = 4
	DT_SINGLELINE                     = 0x20
	DT_END_ELLIPSIS                   = 0x8000
	GW_OWNER                          = 4
	GW_CHILD                          = 5
	MONITOR_DEFAULTTOPRIMARY          = 1
	SM_CXSCREEN                       = 0
	SM_CYSCREEN                       = 1
	SM_CXVIRTUALSCREEN                = 78
	SM_CYVIRTUALSCREEN                = 79
	SM_XVIRTUALSCREEN                 = 76
	SM_YVIRTUALSCREEN                 = 77
	TH32CS_SNAPPROCESS                = 0x00000002
	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	FILE_ATTRIBUTE_NORMAL             = 0x80
	OFN_FILEMUSTEXIST                 = 0x1000
	OFN_PATHMUSTEXIST                 = 0x800
	OFN_EXPLORER                      = 0x80000
	SW_SHOWNORMAL                     = 1
	LWA_ALPHA                         = 2
	CS_HREDRAW                        = 2
	CS_VREDRAW                        = 1
	WS_MINIMIZEBOX                    = 0x20000
	DWMWA_CLOAKED                     = 14
	DWMWA_USE_IMMERSIVE_DARK_MODE     = 20
	VK_ESCAPE                         = 27
	WM_KEYDOWN                        = 0x100
	WM_TIMER                          = 0x113
	WM_ERASEBKGND                     = 0x14
)

type point struct{ X, Y int32 }
type rect struct{ Left, Top, Right, Bottom int32 }

func (r rect) w() int32 { return r.Right - r.Left }
func (r rect) h() int32 { return r.Bottom - r.Top }

type msg struct {
	Hwnd           uintptr
	Message        uint32
	_              uint32
	WParam, LParam uintptr
	Time           uint32
	Pt             point
	Private        uint32
}
type wndClass struct {
	Size                                                            uint32
	Style                                                           uint32
	WndProc                                                         uintptr
	ClsExtra, WndExtra                                              int32
	Instance, Icon, Cursor, Background, MenuName, ClassName, IconSm uintptr
}
type paintStruct struct {
	Hdc                uintptr
	Erase              int32
	RcPaint            rect
	Restore, IncUpdate int32
	Reserved           [32]byte
}
type minMaxInfo struct {
	Reserved     point
	MaxSize      point
	MaxPosition  point
	MinTrackSize point
	MaxTrackSize point
}
type monitorInfo struct {
	Size          uint32
	Monitor, Work rect
	Flags         uint32
}
type processEntry struct {
	Size          uint32
	CntUsage      uint32
	ProcessID     uint32
	DefaultHeapID uintptr
	ModuleID      uint32
	Threads       uint32
	ParentID      uint32
	PriClassBase  int32
	Flags         uint32
	ExeFile       [260]uint16
}
type openFileName struct {
	Size                                 uint32
	Owner, Instance                      uintptr
	Filter, CustomFilter                 uintptr
	MaxCustFilter, FilterIndex           uint32
	File                                 uintptr
	MaxFile                              uint32
	FileTitle                            uintptr
	MaxFileTitle                         uint32
	InitialDir, Title                    uintptr
	Flags                                uint32
	FileOffset, FileExtension            uint16
	DefExt, CustData, Hook, TemplateName uintptr
	Reserved                             uintptr
	Reserved2                            uint32
	FlagsEx                              uint32
}
type guid struct {
	Data1        uint32
	Data2, Data3 uint16
	Data4        [8]byte
}

var (
	k32                          = syscall.NewLazyDLL("kernel32.dll")
	u32                          = syscall.NewLazyDLL("user32.dll")
	g32                          = syscall.NewLazyDLL("gdi32.dll")
	dwm                          = syscall.NewLazyDLL("dwmapi.dll")
	sh32                         = syscall.NewLazyDLL("shell32.dll")
	ole32                        = syscall.NewLazyDLL("ole32.dll")
	comdlg                       = syscall.NewLazyDLL("comdlg32.dll")
	gdip                         = syscall.NewLazyDLL("gdiplus.dll")
	pGetModuleHandle             = k32.NewProc("GetModuleHandleW")
	pSetLastError                = k32.NewProc("SetLastError")
	pRegisterClass               = u32.NewProc("RegisterClassExW")
	pCreateWindow                = u32.NewProc("CreateWindowExW")
	pDefWindowProc               = u32.NewProc("DefWindowProcW")
	pShowWindow                  = u32.NewProc("ShowWindow")
	pUpdateWindow                = u32.NewProc("UpdateWindow")
	pGetMessage                  = u32.NewProc("GetMessageW")
	pTranslateMessage            = u32.NewProc("TranslateMessage")
	pDispatchMessage             = u32.NewProc("DispatchMessageW")
	pPostQuitMessage             = u32.NewProc("PostQuitMessage")
	pDestroyWindow               = u32.NewProc("DestroyWindow")
	pInvalidate                  = u32.NewProc("InvalidateRect")
	pBeginPaint                  = u32.NewProc("BeginPaint")
	pEndPaint                    = u32.NewProc("EndPaint")
	pGetClientRect               = u32.NewProc("GetClientRect")
	pGetWindowRect               = u32.NewProc("GetWindowRect")
	pSetWindowPos                = u32.NewProc("SetWindowPos")
	pSetWindowText               = u32.NewProc("SetWindowTextW")
	pGetWindowText               = u32.NewProc("GetWindowTextW")
	pGetWindowTextLength         = u32.NewProc("GetWindowTextLengthW")
	pSendMessage                 = u32.NewProc("SendMessageW")
	pSetTimer                    = u32.NewProc("SetTimer")
	pKillTimer                   = u32.NewProc("KillTimer")
	pLoadIcon                    = u32.NewProc("LoadIconW")
	pLoadCursor                  = u32.NewProc("LoadCursorW")
	pIsWindowVisible             = u32.NewProc("IsWindowVisible")
	pIsIconic                    = u32.NewProc("IsIconic")
	pGetWindowThreadProcessId    = u32.NewProc("GetWindowThreadProcessId")
	pEnumWindows                 = u32.NewProc("EnumWindows")
	pEnumDisplayMonitors         = u32.NewProc("EnumDisplayMonitors")
	pGetMonitorInfo              = u32.NewProc("GetMonitorInfoW")
	pGetSystemMetrics            = u32.NewProc("GetSystemMetrics")
	pSetLayeredWindowAttributes  = u32.NewProc("SetLayeredWindowAttributes")
	pShowCursor                  = u32.NewProc("ShowCursor")
	pSetFocus                    = u32.NewProc("SetFocus")
	pSetForegroundWindow         = u32.NewProc("SetForegroundWindow")
	pPostMessage                 = u32.NewProc("PostMessageW")
	pSetWindowLongPtr            = u32.NewProc("SetWindowLongPtrW")
	pSetBkMode                   = g32.NewProc("SetBkMode")
	pSetTextColor                = g32.NewProc("SetTextColor")
	pSetBkColor                  = g32.NewProc("SetBkColor")
	pCreateSolidBrush            = g32.NewProc("CreateSolidBrush")
	pCreateCompatibleDC         = g32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap     = g32.NewProc("CreateCompatibleBitmap")
	pSelectObject               = g32.NewProc("SelectObject")
	pBitBlt                     = g32.NewProc("BitBlt")
	pDeleteDC                   = g32.NewProc("DeleteDC")
	pDeleteObject                = g32.NewProc("DeleteObject")
	pCreateFont                  = g32.NewProc("CreateFontW")
	pDwmSet                      = dwm.NewProc("DwmSetWindowAttribute")
	pDwmGet                      = dwm.NewProc("DwmGetWindowAttribute")
	pShellExecute                = sh32.NewProc("ShellExecuteW")
	pSHGetKnownFolderPath        = sh32.NewProc("SHGetKnownFolderPath")
	pCoTaskMemFree               = ole32.NewProc("CoTaskMemFree")
	pGetOpenFileName             = comdlg.NewProc("GetOpenFileNameW")
	pScreenToClient             = u32.NewProc("ScreenToClient")
	pCreateSnapshot              = k32.NewProc("CreateToolhelp32Snapshot")
	pProcessFirst                = k32.NewProc("Process32FirstW")
	pProcessNext                 = k32.NewProc("Process32NextW")
	pCloseHandle                 = k32.NewProc("CloseHandle")
	pOpenProcess                 = k32.NewProc("OpenProcess")
	pQueryFullProcessImageName   = k32.NewProc("QueryFullProcessImageNameW")
	pGetProcessId                = k32.NewProc("GetProcessId")
	pExpandEnv                   = k32.NewProc("ExpandEnvironmentStringsW")
	pGdipStartup                 = gdip.NewProc("GdiplusStartup")
	pGdipShutdown                = gdip.NewProc("GdiplusShutdown")
	pGdipGraphicsFromHDC         = gdip.NewProc("GdipCreateFromHDC")
	pGdipDeleteGraphics          = gdip.NewProc("GdipDeleteGraphics")
	pGdipGraphicsClear           = gdip.NewProc("GdipGraphicsClear")
	pGdipSetSmoothingMode        = gdip.NewProc("GdipSetSmoothingMode")
	pGdipSetInterpolationMode    = gdip.NewProc("GdipSetInterpolationMode")
	pGdipCreateSolidFill         = gdip.NewProc("GdipCreateSolidFill")
	pGdipDeleteBrush             = gdip.NewProc("GdipDeleteBrush")
	pGdipFillRectangle           = gdip.NewProc("GdipFillRectangleI")
	pGdipFillEllipse             = gdip.NewProc("GdipFillEllipseI")
	pGdipDrawString              = gdip.NewProc("GdipDrawString")
	pGdipCreateFontFamily        = gdip.NewProc("GdipCreateFontFamilyFromName")
	pGdipDeleteFontFamily        = gdip.NewProc("GdipDeleteFontFamily")
	pGdipCreateFont              = gdip.NewProc("GdipCreateFont")
	pGdipDeleteFont              = gdip.NewProc("GdipDeleteFont")
	pGdipLoadImage               = gdip.NewProc("GdipLoadImageFromFile")
	pGdipDisposeImage            = gdip.NewProc("GdipDisposeImage")
	pGdipGetImageWidth           = gdip.NewProc("GdipGetImageWidth")
	pGdipGetImageHeight          = gdip.NewProc("GdipGetImageHeight")
	pGdipDrawImageRectRectI      = gdip.NewProc("GdipDrawImageRectRectI")
	pGdipCreateBitmapFromScan0   = gdip.NewProc("GdipCreateBitmapFromScan0")
	pGdipGetImageGraphicsContext = gdip.NewProc("GdipGetImageGraphicsContext")
	pGdipSaveImage               = gdip.NewProc("GdipSaveImageToFile")
	pGdipSetCompositingQuality   = gdip.NewProc("GdipSetCompositingQuality")
	pGdipSetTextRenderingHint    = gdip.NewProc("GdipSetTextRenderingHint")
)

func call(p *syscall.LazyProc, a ...uintptr) uintptr { r, _, _ := p.Call(a...); return r }
func ptrStr(s string) uintptr                        { return uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(s))) }
func utf16zero(p []uint16) string                    { return syscall.UTF16ToString(p) }
func loword(v uintptr) uint16                        { return uint16(v & 0xFFFF) }
func wordXY(v uintptr) (int32, int32) {
	return int32(int16(v & 0xFFFF)), int32(int16((v >> 16) & 0xFFFF))
}
func show(hwnd uintptr, how int)     { call(pShowWindow, hwnd, uintptr(how)) }
func setText(hwnd uintptr, s string) { call(pSetWindowText, hwnd, ptrStr(s)) }
func getText(hwnd uintptr) string {
	if hwnd == 0 {
		return ""
	}
	n := int(call(pGetWindowTextLength, hwnd))
	b := make([]uint16, n+2)
	call(pGetWindowText, hwnd, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	return syscall.UTF16ToString(b)
}
func client(hwnd uintptr) rect {
	var r rect
	call(pGetClientRect, hwnd, uintptr(unsafe.Pointer(&r)))
	return r
}
func invalid(hwnd uintptr) {
	if hwnd != 0 {
		call(pInvalidate, hwnd, 0, 1)
	}
}
func setDarkTitle(hwnd uintptr) {
	v := uint32(1)
	ret := call(pDwmSet, hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE, uintptr(unsafe.Pointer(&v)), 4)
	if ret != 0 {
		call(pDwmSet, hwnd, 19, uintptr(unsafe.Pointer(&v)), 4)
	}
}
func messageBox(hwnd uintptr, msg, title string) {
	call(u32.NewProc("MessageBoxW"), hwnd, ptrStr(msg), ptrStr(title), 0x40)
}
func windowsLoop() {
	var m msg
	for {
		r := call(pGetMessage, uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if r == 0 || r == ^uintptr(0) {
			break
		}
		call(pTranslateMessage, uintptr(unsafe.Pointer(&m)))
		call(pDispatchMessage, uintptr(unsafe.Pointer(&m)))
	}
}
func exitLog(err error) {
	if err != nil {
		messageBox(0, err.Error(), "Voyager Launcher")
	}
}
func winError(detail string) error { return fmt.Errorf("%s (Windows)", detail) }
func memoryKeep(x ...interface{}) {
	for _, v := range x {
		runtime.KeepAlive(v)
	}
}

// Win32 dialog filters contain embedded UTF-16 NUL separators.
// syscall.StringToUTF16 panics on such text, so encode the runes directly.
func fileDialogFilter(filter string) []uint16 {
	encoded := utf16.Encode([]rune(filter))
	if len(encoded) == 0 || encoded[len(encoded)-1] != 0 {
		encoded = append(encoded, 0)
	}
	return append(encoded, 0)
}

func openFile(hwnd uintptr, filter, initial string) string {
	fs := fileDialogFilter(filter)
	b := make([]uint16, 32768)
	initialDir := initial
	if initialDir == "" {
		initialDir = ""
	}
	of := openFileName{Size: uint32(unsafe.Sizeof(openFileName{})), Owner: hwnd, Filter: uintptr(unsafe.Pointer(&fs[0])), File: uintptr(unsafe.Pointer(&b[0])), MaxFile: uint32(len(b)), Flags: OFN_FILEMUSTEXIST | OFN_PATHMUSTEXIST | OFN_EXPLORER, Title: ptrStr("Selecionar arquivo")}
	if initialDir != "" {
		of.InitialDir = ptrStr(initialDir)
	}
	result := call(pGetOpenFileName, uintptr(unsafe.Pointer(&of)))
	memoryKeep(fs, b, of)
	if result != 0 {
		return syscall.UTF16ToString(b)
	}
	return ""
}

func knownPictures() string {
	folder := guid{0x33E28130, 0x4E1E, 0x4676, [8]byte{0x83, 0x5A, 0x98, 0x39, 0x5C, 0x3B, 0xC3, 0xBB}}
	var path uintptr
	hr := call(pSHGetKnownFolderPath, uintptr(unsafe.Pointer(&folder)), 0, 0, uintptr(unsafe.Pointer(&path)))
	if hr != 0 || path == 0 {
		return ""
	}
	defer call(pCoTaskMemFree, path)
	chars := (*[1 << 16]uint16)(unsafe.Pointer(path))
	end := 0
	for end < 32768 && chars[end] != 0 {
		end++
	}
	return syscall.UTF16ToString(chars[:end])
}

func expandWindows(s string) string {
	if s == "" {
		return ""
	}
	b := make([]uint16, 32768)
	r := call(pExpandEnv, ptrStr(s), uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	if r == 0 || r >= uintptr(len(b)) {
		return osExpandFallback(s)
	}
	return syscall.UTF16ToString(b)
}
func osExpandFallback(s string) string { return s }
