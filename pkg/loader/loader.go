package loader

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"krot/internal/lineio"
	"krot/pkg/env"
)

const (
	loadTimeout   = 1 * time.Minute
	loadUserAgent = "Mozilla/5.0 (X11; Linux x86_64; rv:150.0) Gecko/20100101 Firefox/150.0"
)

func Load(urls ...string) ([]string, error) {
	if len(urls) == 0 {
		return nil, fmt.Errorf("urls are empty")
	}
	slog.Info("starting urls load", "sources", len(urls))

	var (
		client     = &http.Client{Timeout: loadTimeout}
		seen       = make(map[string]struct{})
		result     = make([]string, 0)
		failedURLs = make([]string, 0)
		ok         int
		fail       int
	)
	total := len(urls)
	showProgress := !env.IsGitHubActions()

	printProgress := func() {
		if !showProgress {
			return
		}
		processed := ok + fail
		fmt.Fprintf(os.Stderr, "\r%d/%d | ok %d | failed %d | unique %d", processed, total, ok, fail, len(result))
	}

	for _, sourceURL := range urls {
		lines, err := loadSource(client, sourceURL, seen)
		if err != nil {
			fail++
			failedURLs = append(failedURLs, sourceURL)
			printProgress()
			slog.Error("failed to process source", "url", sourceURL, "error", err)
			continue
		}

		for _, line := range lines {
			result = append(result, line.Text)
		}
		ok++
		printProgress()

		slog.Info("source loaded", "url", sourceURL, "unique_total", len(result))
	}
	if showProgress && total > 0 {
		fmt.Fprintln(os.Stderr)
	}
	if len(failedURLs) > 0 {
		fmt.Fprintln(os.Stderr, "failed source urls:")
		for _, failedURL := range failedURLs {
			fmt.Fprintln(os.Stderr, failedURL)
		}
	}
	slog.Info("finished urls load", "unique_total", len(result))

	return result, nil
}

func loadSource(client *http.Client, sourceURL string, seen map[string]struct{}) ([]lineio.Line, error) {
	slog.Debug("loading source", "url", sourceURL)

	req, err := http.NewRequest(http.MethodGet, sourceURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build request for %s: %w", sourceURL, err)
	}
	req.Header.Set("User-Agent", loadUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to load source %s: %w", sourceURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to load %s: status %s", sourceURL, resp.Status)
	}

	return lineio.Read(resp.Body, sourceURL, lineio.Options{Seen: seen})
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
