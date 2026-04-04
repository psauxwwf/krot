package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"krot/pkg/mtproto"
	"krot/pkg/vless"
)

var (
	in       = flag.String("in", "vless.txt", "input file")
	out      = flag.String("out", "", "output file")
	level    = flag.String("level", "info", "log level: debug|info|warn|error")
	timeout  = flag.Duration("timeout", 10*time.Second, "proxy check timeout (e.g. 10s, 1m)")
	workers  = flag.Int("workers", runtime.NumCPU(), "number of concurrent workers")
	pipeline = flag.Bool("pipeline", false, "start all checks")
	shuf     = flag.Bool("shuf", true, "shuffle input lines")
	parse    = flag.Bool("parse", false, "parse only url don't send requests")
	chars    = flag.Int("chars", 8192, "max chars in one line")
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

func toOutname(in string) string {
	return fmt.Sprintf("%s_%s", time.Now().Format("02.01.2006_15:04"), filepath.Base(in))
}

func readJobs(in string, maxChars int) ([]job, error) {
	_in, err := os.Open(in)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file %s: %w", in, err)
	}
	defer _in.Close()

	reader := bufio.NewReader(_in)
	line := 0
	jobs := make([]job, 0)

	for {
		line++
		rawLine, err := reader.ReadString('\n')
		isEOF := errors.Is(err, io.EOF)
		if err != nil && !isEOF && !errors.Is(err, bufio.ErrBufferFull) {
			return nil, fmt.Errorf("failed to read input %s in line %d: %w", in, line, err)
		}

		if errors.Is(err, bufio.ErrBufferFull) {
			for errors.Is(err, bufio.ErrBufferFull) {
				_, err = reader.ReadString('\n')
			}
			if err != nil && !errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("failed to skip too long line %d in input %s: %w", line, in, err)
			}
			slog.Warn("skipping too long line", "line", line)
			if errors.Is(err, io.EOF) {
				break
			}
			continue
		}

		uri := strings.TrimSpace(rawLine)
		skip := false
		switch {
		case uri == "":
			slog.Debug("skipping empty line", "line", line)
			skip = true
		case strings.HasPrefix(uri, "#"):
			slog.Debug("skipping comment line", "line", line)
			skip = true
		case utf8.RuneCountInString(uri) > maxChars:
			slog.Debug("skipping so long line", "line", line)
			skip = true
		}
		if skip {
			if isEOF {
				break
			}
			continue
		}

		jobs = append(jobs, job{line: line, uri: uri})
		if isEOF {
			break
		}
	}

	return jobs, nil
}

func _main(
	in string,
	out string,
	workers int,
) error {
	jobs, err := readJobs(in, *chars)
	if err != nil {
		return err
	}
	total := len(jobs)

	var ok, fail int

	_out, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open output file %s: %w", out, err)
	}
	defer _out.Close()

	if *shuf {
		rand.New(rand.NewSource(time.Now().UnixNano())).Shuffle(len(jobs), func(i, j int) {
			jobs[i], jobs[j] = jobs[j], jobs[i]
		})
	}

	jobsch := make(chan job)
	resch := make(chan result)

	var wg sync.WaitGroup
	for range workers {
		wg.Go(func() {
			for j := range jobsch {
				slog.Debug("checking proxy", "line", j.line)
				err := check(j.uri, *timeout, *parse)
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
		if _, err := _out.WriteString(r.uri + "\n"); err != nil {
			return fmt.Errorf("failed to write output %s in line %d: %w", out, r.line, err)
		}
		if err := _out.Sync(); err != nil {
			return fmt.Errorf("failed to sync output %s in line %d: %w", out, r.line, err)
		}
	}

	slog.Info("proxy checking finished",
		"total", total,
		"ok", ok,
		"failed", fail,
	)

	return nil
}

func main() {
	flag.Parse()

	var parsedLevel slog.Level
	if err := parsedLevel.UnmarshalText([]byte(*level)); err != nil {
		fmt.Fprintf(os.Stderr, "invalid log level %q: %v\n", *level, err)
		os.Exit(1)
	}
	if *timeout <= 0 {
		fmt.Fprintf(os.Stderr, "invalid timeout %q: must be > 0\n", timeout.String())
		os.Exit(2)
	}
	if *workers <= 0 {
		fmt.Fprintf(os.Stderr, "invalid workers %d: must be > 0\n", *workers)
		os.Exit(3)
	}

	logFile, err := os.OpenFile("krot.json", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		os.Exit(4)
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

	if *pipeline {
		if err := errors.Join(
			_main("mtproto.txt", toOutname("mtproto.txt"), *workers*10),
			_main("vless.txt", toOutname("vless.txt"), *workers*10),
		); err != nil {
			slog.Error("fatal error", "error", err)
			os.Exit(5)
		}
		return
	}

	if *out == "" {
		*out = toOutname(*in)
	}
	if err := _main(*in, *out, *workers); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(5)
	}
}

func check(rawURI string, timeout time.Duration, parseOnly bool) error {
	u, err := url.Parse(rawURI)
	if err != nil {
		return fmt.Errorf("invalid uri: %w", err)
	}

	switch {
	case strings.EqualFold(u.Scheme, "vless"):
		return vless.Check(rawURI, timeout, parseOnly)
	case strings.EqualFold(u.Scheme, "tg") && strings.EqualFold(u.Host, "proxy"):
		return mtproto.Check(rawURI, timeout, parseOnly)
	case (strings.EqualFold(u.Scheme, "http") || strings.EqualFold(u.Scheme, "https")) &&
		(strings.EqualFold(u.Host, "t.me") || strings.EqualFold(u.Host, "www.t.me")) &&
		u.Path == "/proxy":
		return mtproto.Check(rawURI, timeout, parseOnly)
	default:
		return fmt.Errorf("unsupported proxy url format: %q", rawURI)
	}
}
