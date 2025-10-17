package menu

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"sort"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"

	"github.com/timoxa0/kxmenu-ng/entry"
	"github.com/timoxa0/kxmenu-ng/input"
)

type MenuItem struct {
	Entry       *entry.BootEntry
	DisplayName string
	Description string
}

type BootMenu struct {
	Items         []MenuItem
	SelectedIndex int
	Terminal      *Terminal
	Title         string
	Timeout       int // seconds, 0 = no timeout
	InputManager  *input.InputManager
}

func NewBootMenu(title string, timeout int, entries []*entry.BootEntry, inputMgr *input.InputManager) *BootMenu {
	menu := &BootMenu{
		Items:         make([]MenuItem, len(entries)),
		SelectedIndex: 0,
		Terminal:      NewTerminal(),
		Title:         title,
		Timeout:       timeout,
		InputManager:  inputMgr,
	}

	for i, e := range entries {
		displayName := strings.TrimSpace(e.Title)
		if displayName == "" {
			displayName = fmt.Sprintf("Boot entry %d", i+1)
		}

		description := ""
		if e.Version != "" {
			description = fmt.Sprintf("Version: %s", e.Version)
		}
		if e.Linux != "" {
			if description != "" {
				description += " | "
			}
			description += fmt.Sprintf("Kernel: %s", e.Linux)
		}
		if e.Devicetree != "" {
			if description != "" {
				description += " | "
			}
			description += fmt.Sprintf("DTB: %s", e.Devicetree)
		}

		menu.Items[i] = MenuItem{
			Entry:       e,
			DisplayName: displayName,
			Description: description,
		}
	}

	c := collate.New(language.English, collate.IgnoreCase, collate.Numeric)

	sort.Slice(menu.Items, func(i, j int) bool {
		return c.CompareString(strings.ToLower(menu.Items[i].Entry.Filename), strings.ToLower(menu.Items[j].Entry.Filename)) < 0
	})

	return menu
}

func (m *BootMenu) Show() (*entry.BootEntry, error) {
	if !m.Terminal.IsTTY {
		return m.showSimpleMenu()
	}

	if err := m.setupTerminal(); err != nil {
		return m.showSimpleMenu()
	}
	defer m.restoreTerminal()

	return m.showInteractiveMenu()
}

func (m *BootMenu) setupTerminal() error {
	cmd := exec.Command("stty", "-echo", "cbreak")
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return err
	}
	fmt.Print(HideCursor + ClearScreen)
	return nil
}

func (m *BootMenu) restoreTerminal() {
	fmt.Print(ShowCursor + ResetColor)
	cmd := exec.Command("stty", "echo", "-cbreak")
	cmd.Stdin = os.Stdin
	cmd.Run()
}

func (m *BootMenu) showSimpleMenu() (*entry.BootEntry, error) {
	fmt.Printf("\n%s\n", m.Title)
	fmt.Println(strings.Repeat("=", len(m.Title)))

	for i, item := range m.Items {
		fmt.Printf("%d. %s\n", i+1, item.DisplayName)
		if item.Description != "" {
			fmt.Printf("   %s\n", item.Description)
		}
	}

	fmt.Printf("\nSelect entry (1-%d) [default: 1]: ", len(m.Items))

	var input string
	fmt.Scanln(&input)

	if input == "" {
		return m.Items[0].Entry, nil
	}

	selection, err := strconv.Atoi(input)
	if err != nil || selection < 1 || selection > len(m.Items) {
		return nil, fmt.Errorf("invalid selection: %s", input)
	}

	return m.Items[selection-1].Entry, nil
}

func (m *BootMenu) showInteractiveMenu() (*entry.BootEntry, error) {
	timeoutCh := make(chan bool, 1)
	timeoutStopCh := make(chan bool, 1)
	inputCh := make(chan input.KeyCode, 1)

	if m.Timeout > 0 {
		go func() {
			for {
				select {
				case <-timeoutStopCh:
					m.Timeout = 0
					return
				default:
					time.Sleep(time.Second)
					m.Timeout -= 1
					if m.Timeout == 0 {
						timeoutCh <- true
						return
					}
					timeoutCh <- false

				}
			}
		}()
	}

	go func() {
		for {
			if event, status := m.InputManager.GetEventNonBlocking(); status {
				inputCh <- event.Code
			}
		}
	}()

	for {
		m.drawMenu()

		select {
		case done := <-timeoutCh:
			if done {
				return m.Items[m.SelectedIndex].Entry, nil
			}
		case key := <-inputCh:
			if m.Timeout != 0 {
				timeoutStopCh <- true
			}
			switch key {
			case input.KeySelect:
				return m.Items[m.SelectedIndex].Entry, nil
			case input.KeyQuit:
				return nil, fmt.Errorf("menu cancelled by user")

			case input.KeyDown:
				if m.SelectedIndex < len(m.Items)-1 {
					m.SelectedIndex++
				}

			case input.KeyUp:
				if m.SelectedIndex > 0 {
					m.SelectedIndex--
				}
			}
		}
	}
}

func (m *BootMenu) drawMenu() {
	// Calculate menu dimensions
	menuItemsHeight := len(m.Items)
	titleHeight := 3      // title + separator + blank line
	bottomInfoHeight := 4 // info panel + controls
	totalMenuHeight := titleHeight + menuItemsHeight + bottomInfoHeight

	// Calculate vertical centering
	startRow := max(1, (m.Terminal.Height-totalMenuHeight)/2)

	// Clear screen and position cursor
	fmt.Print(ClearScreen)
	fmt.Print(EscSeq + fmt.Sprintf("%d;1H", startRow))

	// Draw title (centered)
	titlePadding := max(0, (m.Terminal.Width-len(m.Title))/2)
	fmt.Print(strings.Repeat(" ", titlePadding) + BoldText + CyanText + m.Title + ResetColor + "\n\n")

	// Calculate menu item centering
	maxItemWidth := 0
	for _, item := range m.Items {
		maxItemWidth = max(len(item.DisplayName), maxItemWidth)
	}
	maxItemWidth += 2 // Add padding
	itemPadding := max(0, (m.Terminal.Width-maxItemWidth)/2)

	// Draw menu items (centered)
	for i, item := range m.Items {
		prefix := ""
		suffix := ""

		if i == m.SelectedIndex {
			prefix = BoldText + ReverseVideo
			suffix = ResetColor
		}

		// Truncate long names to fit terminal width
		displayName := item.DisplayName
		maxNameWidth := m.Terminal.Width - itemPadding - 2
		if len(displayName) > maxNameWidth {
			displayName = displayName[:maxNameWidth-3] + "..."
		}

		// Center the menu item
		fmt.Print(strings.Repeat(" ", itemPadding))
		fmt.Printf("%s%s%s\n", prefix, displayName, suffix)
	}

	// Move to bottom area for entry info and controls
	bottomStartRow := m.Terminal.Height - bottomInfoHeight + 1
	fmt.Print(EscSeq + fmt.Sprintf("%d;1H", bottomStartRow))

	// Draw separator line above bottom info
	bottomSeparator := strings.Repeat("─", m.Terminal.Width-2)
	fmt.Printf(" %s \n", bottomSeparator)

	// Show detailed info for selected item
	if m.SelectedIndex < len(m.Items) {
		selectedEntry := m.Items[m.SelectedIndex].Entry

		// Kernel info
		if selectedEntry.Linux != "" {
			fmt.Printf(" %sKernel:%s %s\n", BoldText, ResetColor, selectedEntry.Linux)
		}

		// Ramdisk info
		if selectedEntry.Initrd != "" {
			fmt.Printf(" %sRamdisk:%s %s\n", BoldText, ResetColor, selectedEntry.Initrd)
		}

		// Devicetree info
		if selectedEntry.Devicetree != "" {
			fmt.Printf(" %sDevicetree:%s %s\n", BoldText, ResetColor, selectedEntry.Devicetree)
		}
	}

	// Draw controls footer (centered)
	fmt.Print(EscSeq + fmt.Sprintf("%d;1H", m.Terminal.Height))
	footer := "Use Arrows/Volume keys to select, Enter/Power to confirm"
	if m.Timeout > 0 {
		footer += fmt.Sprintf(" (timeout: %ds)", m.Timeout)
	}

	footerPadding := max(0, (m.Terminal.Width-len(footer))/2)
	fmt.Print(strings.Repeat(" ", footerPadding) + WhiteText + footer + ResetColor)
}
