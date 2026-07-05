package lineio

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

type Line struct {
	Number int
	Text   string
}

type Options struct {
	SkipComments bool
	MaxChars     int
	Seen         map[string]struct{}
}

func Read(r io.Reader, source string, opts Options) ([]Line, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	lines := make([]Line, 0)
	lineNo := 0

	for scanner.Scan() {
		lineNo++

		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}
		if opts.SkipComments && strings.HasPrefix(text, "#") {
			continue
		}
		if opts.MaxChars > 0 && utf8.RuneCountInString(text) > opts.MaxChars {
			continue
		}
		if opts.Seen != nil {
			if _, ok := opts.Seen[text]; ok {
				continue
			}
			opts.Seen[text] = struct{}{}
		}

		lines = append(lines, Line{Number: lineNo, Text: text})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read %s in line %d: %w", source, lineNo, err)
	}

	return lines, nil
}

func MergeFiles(inputs []string, opts Options) (string, error) {
	merged, err := os.CreateTemp("", "krot-input-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp input: %w", err)
	}
	defer func() {
		_ = merged.Close()
	}()
	defer func() {
		if err != nil {
			_ = os.Remove(merged.Name())
		}
	}()

	writer := bufio.NewWriterSize(merged, 256*1024)
	seen := opts.Seen
	if seen == nil {
		seen = make(map[string]struct{})
	}
	readOpts := opts
	readOpts.Seen = seen

	for _, input := range inputs {
		if err := appendFile(writer, input, readOpts); err != nil {
			return "", err
		}
	}

	if err := writer.Flush(); err != nil {
		return "", fmt.Errorf("failed to flush temp input %q: %w", merged.Name(), err)
	}

	return merged.Name(), nil
}

func appendFile(writer *bufio.Writer, input string, opts Options) error {
	f, err := os.Open(input)
	if err != nil {
		return fmt.Errorf("failed to open input file %q: %w", input, err)
	}
	defer f.Close()

	lines, err := Read(f, input, opts)
	if err != nil {
		return err
	}

	for _, line := range lines {
		if _, err := writer.WriteString(line.Text + "\n"); err != nil {
			return fmt.Errorf("failed to write merged input from %q: %w", input, err)
		}
	}

	return nil
}
