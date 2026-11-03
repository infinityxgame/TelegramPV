//go:build windows

package main

import (
	"log"
	"strings"
	"syscall"
	"unsafe"
)

func showNotice(msg string) {
	user32 := syscall.NewLazyDLL("user32.dll")
	text, err := syscall.UTF16PtrFromString(msg)
	if err != nil {
		log.Printf("[SYS] Error en texto de aviso: %v", err)
		return
	}
	title, err := syscall.UTF16PtrFromString("TelegramPV")
	if err != nil {
		log.Printf("[SYS] Error en título de aviso: %v", err)
		return
	}
	user32.NewProc("MessageBoxW").Call(0,
		uintptr(unsafe.Pointer(text)),
		uintptr(unsafe.Pointer(title)),
		0x40) // MB_ICONINFORMATION
}

func isTelegramDLRunning() bool {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procOpenMutexW := kernel32.NewProc("OpenMutexW")
	name, err := syscall.UTF16PtrFromString(`Local\TelegramDL_SingleInstance`)
	if err == nil {
		const SYNCHRONIZE = 0x00100000
		h, _, _ := procOpenMutexW.Call(SYNCHRONIZE, 0, uintptr(unsafe.Pointer(name)))
		if h != 0 {
			kernel32.NewProc("CloseHandle").Call(h)
			return true
		}
	}

	type PROCESSENTRY32 struct {
		DwSize              uint32
		CntUsage            uint32
		Th32ProcessID       uint32
		Th32DefaultHeapID   uintptr
		Th32ModuleID        uint32
		CntThreads          uint32
		Th32ParentProcessID uint32
		PcPriClassBase      int32
		DwFlags             uint32
		SzExeFile           [260]uint16
	}

	const TH32CS_SNAPPROCESS = 0x00000002
	procCreateToolhelp32Snapshot := kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW := kernel32.NewProc("Process32FirstW")
	procProcess32NextW := kernel32.NewProc("Process32NextW")

	snap, _, _ := procCreateToolhelp32Snapshot.Call(TH32CS_SNAPPROCESS, 0)
	if snap != 0 && snap != ^uintptr(0) {
		defer kernel32.NewProc("CloseHandle").Call(snap)
		var entry PROCESSENTRY32
		entry.DwSize = uint32(unsafe.Sizeof(entry))

		r1, _, _ := procProcess32FirstW.Call(snap, uintptr(unsafe.Pointer(&entry)))
		for r1 != 0 {
			exeName := syscall.UTF16ToString(entry.SzExeFile[:])
			lowerExe := strings.ToLower(exeName)
			if lowerExe == "telegramdl.exe" || lowerExe == "tgdown.exe" {
				return true
			}
			r1, _, _ = procProcess32NextW.Call(snap, uintptr(unsafe.Pointer(&entry)))
		}
	}
	return false
}
