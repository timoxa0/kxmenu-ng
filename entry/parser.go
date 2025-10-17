package entry

import (
	"bufio"
	"os"
	"strings"
)

type BootEntry struct {
	Title      string
	Version    string
	Linux      string
	Initrd     string
	Devicetree string
	Options    string
	Filename   string
}

func ParseEntry(entry_file *os.File) (*BootEntry, error) {
	entry := &BootEntry{
		Filename: entry_file.Name(),
	}
	scanner := bufio.NewScanner(entry_file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue // Skip empty lines and comments
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) < 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		// Assign values based on key
		switch key {
		case "title":
			entry.Title = value
		case "version":
			entry.Version = value
		case "linux":
			entry.Linux = value
		case "initrd":
			entry.Initrd = value
		case "devicetree":
			entry.Devicetree = value
		case "options":
			entry.Options = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entry, nil
}

func (e *BootEntry) CleanupEntry() {
	e.Initrd = strings.ReplaceAll(e.Initrd, " $tuned_initrd", "")
	e.Options = strings.ReplaceAll(e.Options, " $tuned_params", "")
}
