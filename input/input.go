package input

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

const EVIOCGNAME = 0x80ff4506

type KeyCode int

const (
	KeyUnknown KeyCode = iota
	KeyUp
	KeyDown
	KeySelect
	KeyEscape
	KeyQuit
)

type EventType int

const (
	KeyRelease EventType = iota
	KeyPress
)

type KeyEvent struct {
	Code KeyCode
	Type EventType
}

type InputDevice struct {
	Name      string
	Path      string
	File      *os.File
	keyStates map[uint16]bool
}

type InputManager struct {
	devices   []InputDevice
	eventChan chan KeyEvent
	stopChan  chan bool
}

// input_event from <linux/input.h>
type inputEvent struct {
	Time  syscall.Timeval
	Type  uint16
	Code  uint16
	Value int32
}

const (
	EV_SYN = 0x00
	EV_KEY = 0x01
)

const (
	KEY_ESC        = 1
	KEY_Q          = 16
	KEY_ENTER      = 28
	KEY_UP         = 103
	KEY_DOWN       = 108
	KEY_VOLUMEDOWN = 114
	KEY_VOLUMEUP   = 115
	KEY_POWER      = 116
)

func NewInputManager() *InputManager {
	return &InputManager{
		devices:   make([]InputDevice, 0),
		eventChan: make(chan KeyEvent, 10),
		stopChan:  make(chan bool, 1),
	}
}

func (im *InputManager) DiscoverDevices() error {
	// Look for input devices in /dev/input/
	inputDir := "/dev/input"
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return fmt.Errorf("failed to read input directory: %v", err)
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "event") {
			devicePath := filepath.Join(inputDir, entry.Name())

			// Try to open the device
			file, err := os.OpenFile(devicePath, os.O_RDONLY, 0)
			if err != nil {
				continue // Skip devices we can't open
			}

			// Get device name
			name, err := getDeviceName(file)
			if err != nil {
				name = entry.Name() // Use filename as fallback
			}

			// Add all devices that can be opened (skip capability check for now)
			device := InputDevice{
				Name:      name,
				Path:      devicePath,
				File:      file,
				keyStates: make(map[uint16]bool),
			}
			im.devices = append(im.devices, device)
			fmt.Printf("Found input device: %s (%s)\n", name, devicePath)
		}
	}

	return nil
}

func getDeviceName(file *os.File) (string, error) {
	name := make([]byte, 256)
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		file.Fd(),
		uintptr(EVIOCGNAME),
		uintptr(unsafe.Pointer(&name[0])))

	if errno != 0 {
		return "", fmt.Errorf("failed to get device name")
	}

	// Find null terminator and convert to string
	end := 0
	for i, b := range name {
		if b == 0 {
			end = i
			break
		}
	}

	return string(name[:end]), nil
}

func (im *InputManager) StartListening() {
	for i := range im.devices {
		go im.listenDevice(&im.devices[i])
	}
}

func (im *InputManager) listenDevice(device *InputDevice) {
	eventSize := int(unsafe.Sizeof(inputEvent{}))
	buf := make([]byte, eventSize)

	for {
		select {
		case <-im.stopChan:
			return
		default:
			n, err := device.File.Read(buf)
			if err != nil {
				log.Fatalf("Error reading from device %s: %v\n", device.Path, err)
				return
			}

			if n != eventSize {
				// Incomplete read, skip this event
				continue
			}

			event := (*inputEvent)(unsafe.Pointer(&buf[0]))

			if event.Type == EV_KEY {
				im.handleKeyEvent(device, event.Code, event.Value)
			}
		}
	}
}

func (im *InputManager) handleKeyEvent(device *InputDevice, keyCode uint16, value int32) {
	switch value {
	case 0: // Key release
		if !device.keyStates[keyCode] {
			return
		}
		device.keyStates[keyCode] = false
		keyEvent := im.translateKeyCode(keyCode, 0)
		if keyEvent.Code == KeyUnknown {
			return
		}
		select {
		case im.eventChan <- keyEvent:
		default: // Channel full -> drop event
		}
	case 1: // Key press
		device.keyStates[keyCode] = true
		keyEvent := im.translateKeyCode(keyCode, 1)
		if keyEvent.Code == KeyUnknown {
			return
		}
		select {
		case im.eventChan <- keyEvent:
		default: // Channel full -> drop event
		}
	case 2: // Key repeat - ignore

	}
}

func (im *InputManager) translateKeyCode(linuxCode uint16, value uint32) KeyEvent {
	var keyEvent KeyEvent
	keyEvent.Type = EventType(value)

	switch linuxCode {
	case KEY_VOLUMEUP, KEY_UP:
		keyEvent.Code = KeyUp
	case KEY_VOLUMEDOWN, KEY_DOWN:
		keyEvent.Code = KeyDown
	case KEY_POWER, KEY_ENTER:
		keyEvent.Code = KeySelect
	case KEY_ESC:
		keyEvent.Code = KeyEscape
	case KEY_Q:
		keyEvent.Code = KeyQuit
	default:
		keyEvent.Code = KeyUnknown
	}

	return keyEvent
}

func (im *InputManager) GetEvent() KeyEvent {
	return <-im.eventChan
}

func (im *InputManager) GetEventNonBlocking() (KeyEvent, bool) {
	select {
	case event := <-im.eventChan:
		return event, true
	default:
		return KeyEvent{}, false
	}
}

func (im *InputManager) Stop() {
	close(im.stopChan)

	// Close device files
	for _, device := range im.devices {
		device.File.Close()
	}
}
