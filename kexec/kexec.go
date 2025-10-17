//go:build linux && arm64

package kexec

import (
	"fmt"
	"golang.org/x/sys/unix"
	"syscall"
	"unsafe"
)

const SYS_KEXEC_LOAD_FILE = uintptr(294)
const RB_KEXEC = 0x45584543

func Boot() error {
	return unix.Reboot(RB_KEXEC)
}

func LoadFile(kernel_fd, initrd_fd uintptr, cmdline string) error {
	var cmdlineBytes []byte
	if cmdline != "" {
		cmdlineBytes = append([]byte(cmdline), 0)
	} else {
		cmdlineBytes = []byte{0}
	}

	var cmdPtr uintptr
	if len(cmdlineBytes) > 0 {
		cmdPtr = uintptr(unsafe.Pointer(&cmdlineBytes[0]))
	} else {
		cmdPtr = 0
	}

	// syscall: int kexec_file_load(int kernel_fd, int initrd_fd, unsigned long cmdline_len,
	//                             const char *cmdline, unsigned long flags);

	_, _, errno := syscall.Syscall6(
		SYS_KEXEC_LOAD_FILE,        // syscall no
		uintptr(kernel_fd),         // kernel_fd
		uintptr(initrd_fd),         // initrd_fd (0 if none)
		uintptr(len(cmdlineBytes)), // cmdline_len
		cmdPtr,                     // cmdline pointer
		uintptr(0),                 // no flags
		0,                          // unused
	)

	if errno == 0 {
		return nil
	} else {
		return fmt.Errorf("%v\n", errno)
	}
}
