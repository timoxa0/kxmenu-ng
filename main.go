package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/timoxa0/kxmenu-ng/entry"
	"github.com/timoxa0/kxmenu-ng/input"
	"github.com/timoxa0/kxmenu-ng/kexec"
	"github.com/timoxa0/kxmenu-ng/menu"
)

func main() {
	bootdir := "/mnt"
	if len(os.Args) > 1 {
		bootdir = os.Args[1]
	}

	entries, err := entry.FindEntries(bootdir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error scanning directory: %v\n", err)
		os.Exit(1)
	}

	if len(entries) == 0 {
		fmt.Printf("No boot entries found in %s\n", bootdir)
		os.Exit(1)
	}

	inputMgr := input.NewInputManager()
	if err := inputMgr.DiscoverDevices(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to discover input devices: %v\n", err)
		os.Exit(1)
	}
	inputMgr.StartListening()
	defer inputMgr.Stop()

	bootMenu := menu.NewBootMenu("Boot menu", 10, entries, inputMgr)
	fmt.Println()

	selectedEntry, err := bootMenu.Show()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Menu error: %v\n", err)
		os.Exit(1)
	}

	kernelFile, err := os.Open(bootdir + selectedEntry.Linux)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open kernel: %v\n", err)
		os.Exit(1)
	}
	if strings.HasPrefix(filepath.Base(selectedEntry.Linux), "vmlinuz") {
		gzipReader, err := gzip.NewReader(kernelFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to decompress kernel: %v\n", err)
			os.Exit(1)
		}
		decompressedKernelFile, err := os.CreateTemp("/tmp", "kernel-*")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create uncompressed kernel image: %v\n", err)
			os.Exit(1)
		}
		io.Copy(decompressedKernelFile, gzipReader)
		kernelFile.Close()
		decompressedKernelFile.Close()
		kernelFile, err = os.Open(decompressedKernelFile.Name())
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open uncompressed kernel: %v\n", err)
			os.Exit(1)
		}
	}

	ramdiskFile, err := os.Open(bootdir + selectedEntry.Initrd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open ramdisk: %v\n", err)
		os.Exit(1)
	}

	err = kexec.LoadFile(kernelFile.Fd(), ramdiskFile.Fd(), selectedEntry.Options)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed load new kernel: %v\n", err)
		os.Exit(1)
	}

	kernelFile.Close()
	ramdiskFile.Close()

	if err := kexec.Boot(); err != nil {
		fmt.Fprintf(os.Stderr, "reboot(RB_KEXEC) failed: %v\n", err)
		os.Exit(1)
	}
}
