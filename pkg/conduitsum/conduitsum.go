package conduitsum

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
)

const Filename = "conduit.sum"

// Parse parses the contents of a conduit.sum file.
// Each line contains a single schema hash representing the expected state
// of the database after the corresponding migration is applied.
func Parse(data []byte) ([]string, error) {
	var hashes []string

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		if strings.ContainsAny(line, " \t") {
			return nil, fmt.Errorf("conduitsum: invalid line: %q", line)
		}

		hashes = append(hashes, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("conduitsum: failed to read: %w", err)
	}

	return hashes, nil
}

// Format serializes schema hashes into the conduit.sum file format.
func Format(hashes []string) []byte {
	var buf bytes.Buffer

	for _, h := range hashes {
		fmt.Fprintf(&buf, "%s\n", h)
	}

	return buf.Bytes()
}
