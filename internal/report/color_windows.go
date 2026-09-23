//go:build windows

package report

import (
	"io"
	"os"
	"syscall"
	"unsafe"
)

// enableVT switches the given console to virtual-terminal processing so
// ANSI escape sequences render on Windows 10+. Pure stdlib (syscall);
// failures are ignored — colors simply degrade.
func enableVT(out io.Writer) {
	f, ok := out.(*os.File)
	if !ok {
		return
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")
	const enableVirtualTerminalProcessing = 0x0004

	var mode uint32
	if rc, _, _ := getMode.Call(f.Fd(), uintptr(unsafe.Pointer(&mode))); rc == 0 {
		return
	}
	_, _, _ = setMode.Call(f.Fd(), uintptr(mode|enableVirtualTerminalProcessing))
}
