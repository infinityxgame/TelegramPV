//go:build windows

package server

import (
	"fmt"
	"syscall"
	"unsafe"
)

func openURL(targetURL string) error {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	shellExecute := shell32.NewProc("ShellExecuteW")
	verbPtr, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	urlPtr, err := syscall.UTF16PtrFromString(targetURL)
	if err != nil {
		return err
	}
	// 1 = SW_SHOWNORMAL
	ret, _, _ := shellExecute.Call(0, uintptr(unsafe.Pointer(verbPtr)), uintptr(unsafe.Pointer(urlPtr)), 0, 0, 1)
	if ret <= 32 {
		return fmt.Errorf("ShellExecuteW error code %d", ret)
	}
	return nil
}
