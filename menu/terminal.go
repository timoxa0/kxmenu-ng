package menu

import (
	"os"
	"syscall"
	"unsafe"
)

type Terminal struct {
	Width  int
	Height int
	IsTTY  bool
}

func isatty(fd int) bool {
	var termios syscall.Termios
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, // TCGETS
		uintptr(fd), 0x5401,
		uintptr(unsafe.Pointer(&termios)),
		0, 0, 0)
	return err == 0
}

func getTerminalSize() (int, int) {
	type winsize struct {
		Row    uint16
		Col    uint16
		Xpixel uint16
		Ypixel uint16
	}

	ws := &winsize{}
	ret, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(0x5413), // TIOCGWINSZ
		uintptr(unsafe.Pointer(ws)))
	if int(ret) == -1 { // syscall failed
		return 0, 0
	}
	return int(ws.Col), int(ws.Row)
}

func NewTerminal() *Terminal {
	term := &Terminal{
		Width:  80, // default fallback
		Height: 24, // default fallback
		IsTTY:  false,
	}

	if isatty(int(os.Stdout.Fd())) {
		term.IsTTY = true
		if width, height := getTerminalSize(); width > 0 && height > 0 {
			term.Width = width
			term.Height = height
		}
	}

	return term
}
