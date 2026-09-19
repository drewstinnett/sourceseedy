//go:build windows

package cmd

import (
	"io"
	"os"
	"syscall"
)

var setConsoleMode = syscall.NewLazyDLL("kernel32.dll").NewProc("SetConsoleMode")

const enableVirtualTerminalProcessing = 0x0004

// ansiSupported turns on escape sequence handling for the console w writes to,
// which a Windows console doesn't do until it is asked, and reports whether it
// is on. Without it the color codes would print as literal text
func ansiSupported(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	h := syscall.Handle(f.Fd())
	var mode uint32
	if err := syscall.GetConsoleMode(h, &mode); err != nil {
		return false
	}
	if mode&enableVirtualTerminalProcessing != 0 {
		return true
	}
	r, _, _ := setConsoleMode.Call(uintptr(h), uintptr(mode|enableVirtualTerminalProcessing))
	return r != 0
}
