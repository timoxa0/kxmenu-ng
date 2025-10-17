package entry

import (
	"bufio"
	"fmt"
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

func FindEntries(dir string) ([]*BootEntry, error) {
	var entries []*BootEntry

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory %s does not exist", dir)
	}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if isEntryFile(info.Name()) {
			file, err := os.Open(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to open %s: %v\n", path, err)
				return nil
			}
			defer file.Close()
			entry, err := ParseEntry(file)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v\n", path, err)
				return nil
			}
			entries = append(entries, entry)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error scanning directory %s: %v", dir, err)
	}

	return entries, nil
}

func isEntryFile(filename string) bool {
	if strings.HasSuffix(filename, ".conf") {
		return true
	}

	return false
}

func (e *BootEntry) CleanupEntry() {
	e.Initrd = strings.ReplaceAll(e.Initrd, " $tuned_initrd", "")
	e.Options = strings.ReplaceAll(e.Options, " $tuned_params", "")
}
