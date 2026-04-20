package loader

import (
	"bufio"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"
)

const loadTimeout = 2 * time.Minute

func Load(urls ...string) ([]string, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("urls are empty")
	}
	slog.Info("starting urls load", "sources", len(urls))

	client := &http.Client{Timeout: loadTimeout}
	result := make([]string, 0)

	for _, sourceURL := range urls {
		slog.Debug("loading source", "url", sourceURL)
		resp, err := client.Get(sourceURL)
		if err != nil {
			slog.Error("failed to load source", "url", sourceURL, "error", err)
			return result, fmt.Errorf("failed to load %s: %w", sourceURL, err)
		}

		err = func() error {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to load %s: status %s", sourceURL, resp.Status)
			}

			scanner := bufio.NewScanner(resp.Body)
			scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}

				if slices.Contains(result, line) {
					continue
				}

				result = append(result, line)
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("failed to read %s: %w", sourceURL, err)
			}

			return nil
		}()
		if err != nil {
			slog.Error("failed to parse source", "url", sourceURL, "error", err)
			return result, err
		}
		slog.Info("source loaded", "url", sourceURL, "unique_total", len(result))
	}
	slog.Info("finished urls load", "unique_total", len(result))

	return result, nil
}

func Save(out string, urls []string) error {
	slog.Info("saving", "out", out)
	return save(out, urls)
}

func save(out string, urls []string) error {
	if out == "" {
		return fmt.Errorf("output file is empty")
	}

	lines, err := Load(urls...)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open output file %s: %w", out, err)
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			slog.Error("failed to write line", "out", out, "error", err)
			return fmt.Errorf("failed to write output file %s: %w", out, err)
		}
	}
	slog.Info("save completed", "out", out, "lines", len(lines))

	return nil
}
