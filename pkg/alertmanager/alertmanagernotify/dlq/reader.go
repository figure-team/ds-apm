package dlq

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ReadEntries reads all dead-letter entries persisted under path, including
// rotated siblings (path.<timestamp>) written by JSONLDeadLetterSink. Entries
// are returned in chronological order: older rotated files first, then the
// active primary file, each in write order.
//
// A missing primary file with no rotated siblings reads as an empty slice
// (not an error) so a replay over a never-written DLQ is a clean no-op.
// Blank lines are skipped; a malformed line is a hard error so a replay never
// silently drops a failed delivery.
func ReadEntries(path string) ([]*Entry, error) {
	// JSONLDeadLetterSink rotates the primary to "<path>.<timestamp>" siblings,
	// so every persisted file matches the "<path>*" glob.
	matches, err := filepath.Glob(path + "*")
	if err != nil {
		return nil, fmt.Errorf("dlq read: glob: %w", err)
	}

	// Read rotated siblings (older) before the active primary (newest). The
	// timestamp suffix sorts lexically in chronological order.
	var rotated []string
	primaryExists := false
	for _, m := range matches {
		if m == path {
			primaryExists = true
			continue
		}
		rotated = append(rotated, m)
	}
	sort.Strings(rotated)
	files := rotated
	if primaryExists {
		files = append(files, path)
	}

	entries := make([]*Entry, 0)
	for _, f := range files {
		fileEntries, err := readEntriesFromFile(f)
		if err != nil {
			return nil, err
		}
		entries = append(entries, fileEntries...)
	}
	return entries, nil
}

func readEntriesFromFile(path string) ([]*Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("dlq read: open %s: %w", path, err)
	}
	defer f.Close() //nolint:errcheck

	var entries []*Entry
	scanner := bufio.NewScanner(f)
	// Match the sink's per-line ceiling so a large payload is not truncated.
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		b := scanner.Bytes()
		if len(b) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(b, &e); err != nil {
			return nil, fmt.Errorf("dlq read: %s line %d: %w", path, line, err)
		}
		entries = append(entries, &e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("dlq read: scan %s: %w", path, err)
	}
	return entries, nil
}
