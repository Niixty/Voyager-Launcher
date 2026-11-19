package main

import (
	"fmt"
	"golang.org/x/sys/windows"
	"runtime"
	"syscall"
	"unsafe"
)

// SHELLEXECUTEINFOW with SEE_MASK_NOCLOSEPROCESS returns the resolved shortcut's process.
func launchShortcut(path, args, dir string) (uint32, error) {
	type executeInfo struct {
		Size, Mask                        uint32
		Owner                             uintptr
		Verb, File, Parameters, Directory *uint16
		Show                              int32
		Instance, IDList, Class, ClassKey uintptr
		HotKey                            uint32
		Icon, Process                     uintptr
	}
	file, e := syscall.UTF16PtrFromString(path)
	if e != nil {
		return 0, e
	}
	params, e := syscall.UTF16PtrFromString(args)
	if e != nil {
		return 0, e
	}
	cwd, e := syscall.UTF16PtrFromString(dir)
	if e != nil {
		return 0, e
	}
	info := executeInfo{Mask: 0x40 | 0x100, File: file, Parameters: params, Directory: cwd, Show: SW_SHOWNORMAL}
	info.Size = uint32(unsafe.Sizeof(info))
	ok, _, err := syscall.NewLazyDLL("shell32.dll").NewProc("ShellExecuteExW").Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(info)
	if ok == 0 {
		return 0, fmt.Errorf("não foi possível abrir o atalho: %w", err)
	}
	if info.Process == 0 {
		return 0, nil
	}
	defer windows.CloseHandle(windows.Handle(info.Process))
	return windows.GetProcessId(windows.Handle(info.Process))
}
