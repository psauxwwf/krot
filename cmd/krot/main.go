package main

import (
	"bufio"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"krot/pkg/mtproto"
	"krot/pkg/vless"
)

var (
	in      = flag.String("in", "in.txt", "input file")
	out     = flag.String("out", "out.txt", "output file for working proxies")
	level   = flag.String("level", "info", "log level: debug|info|warn|error")
	timeout = flag.Duration("timeout", 30*time.Second, "proxy check timeout (e.g. 30s, 1m)")
	workers = flag.Int("workers", runtime.NumCPU(), "number of concurrent workers")
)

type job struct {
	line int
	uri  string
}

type result struct {
	line int
	uri  string
	err  error
}

func main() {
	flag.Parse()

	var parsedLevel slog.Level
	if err := parsedLevel.UnmarshalText([]byte(*level)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid log level %q: %v\n", *level, err)
		os.Exit(2)
	}
	if *timeout <= 0 {
		fmt.Fprintf(os.Stderr, "invalid timeout %q: must be > 0\n", timeout.String())
		os.Exit(2)
	}
	if *workers <= 0 {
		fmt.Fprintf(os.Stderr, "invalid workers %d: must be > 0\n", *workers)
		os.Exit(2)
	}

	logFile, err := os.OpenFile("krot.json", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer logFile.Close()

	log := slog.New(
		slog.NewMultiHandler(
			slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
				AddSource: true,
				Level:     parsedLevel,
			}),
			slog.NewJSONHandler(logFile, &slog.HandlerOptions{
				AddSource: true,
				Level:     parsedLevel,
			}),
		),
	)
	slog.SetDefault(log)
	slog.Info("starting proxy checker",
		"input", *in, "out", *out, "level", parsedLevel.String(), "timeout", timeout.String(), "workers", *workers)

	file, err := os.Open(*in)
	if err != nil {
		slog.Error("failed to open input file", "path", *in, "error", err)
		os.Exit(1)
	}
	defer file.Close()

	outFile, err := os.OpenFile(*out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		slog.Error("failed to open output file", "path", *out, "error", err)
		os.Exit(1)
	}
	defer outFile.Close()

	scanner := bufio.NewScanner(file)
	var (
		line, total, ok, fail int
		jobs                  []job
	)
	for scanner.Scan() {
		line++
		uri := strings.TrimSpace(scanner.Text())
		if uri == "" {
			slog.Debug("skipping empty line", "line", line)
			continue
		}
		if strings.HasPrefix(uri, "#") {
			slog.Debug("skipping comment line", "line", line)
			continue
		}
		total++
		jobs = append(jobs, job{line: line, uri: uri})
	}

	if err := scanner.Err(); err != nil {
		slog.Error("failed to read input", "path", *in, "error", err)
		os.Exit(1)
	}

	jobsch := make(chan job)
	resch := make(chan result)

	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Go(func() {
			for j := range jobsch {
				slog.Debug("checking proxy", "line", j.line)
				err := check(j.uri, *timeout)
				resch <- result{line: j.line, uri: j.uri, err: err}
			}
		})
	}

	go func() {
		for _, j := range jobs {
			jobsch <- j
		}
		close(jobsch)
		wg.Wait()
		close(resch)
	}()

	for r := range resch {
		if r.err != nil {
			fail++
			slog.Warn("proxy check failed", "line", r.line, "uri", r.uri, "error", r.err)
			continue
		}

		ok++
		slog.Info("proxy check ok", "line", r.line, "uri", r.uri)
		if _, err := outFile.WriteString(r.uri + "\n"); err != nil {
			slog.Error("failed to write output", "path", *out, "line", r.line, "error", err)
			os.Exit(1)
		}
		if err := outFile.Sync(); err != nil {
			slog.Error("failed to sync output", "path", *out, "line", r.line, "error", err)
			os.Exit(1)
		}
	}

	slog.Info("proxy checking finished",
		"total", total,
		"ok", ok,
		"failed", fail,
	)
}

func check(rawURI string, timeout time.Duration) error {
	u, err := url.Parse(rawURI)
	if err != nil {
		return fmt.Errorf("invalid uri: %w", err)
	}

	switch {
	case strings.EqualFold(u.Scheme, "vless"):
		return vless.CheckWithTimeout(rawURI, timeout)
	case strings.EqualFold(u.Scheme, "tg") && strings.EqualFold(u.Host, "proxy"):
		return mtproto.Check(rawURI, timeout)
	case (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) &&
		(strings.EqualFold(u.Host, "t.me") || strings.EqualFold(u.Host, "www.t.me")) &&
		u.Path == "/proxy":
		return mtproto.Check(rawURI, timeout)
	default:
		return fmt.Errorf("unsupported proxy url format: %q", rawURI)
	}
}
