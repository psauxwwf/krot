package krot

import (
	"bufio"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"krot/pkg/checker"
	"krot/pkg/mtproto"
)

type Krot struct {
	timeout   time.Duration
	parseOnly bool
	maxChars  int
	shuffle   bool
}

type job struct {
	line int
	uri  string
}

type result struct {
	line int
	uri  string
	err  error
}

func New(_timeout time.Duration, _parseOnly bool, _maxChars int, _shuffle bool) *Krot {
	return &Krot{
		timeout:   _timeout,
		parseOnly: _parseOnly,
		maxChars:  _maxChars,
		shuffle:   _shuffle,
	}
}

func ToOutname(in string) string {
	return fmt.Sprintf("%s_%s", time.Now().Format("02.01.2006_15_04"), filepath.Base(in))
}

func (k *Krot) Pipeline(workers int, urls map[string][]string) error {
	loadFiles := make(map[string][]string, len(urls))
	for key, list := range urls {
		loadFiles[key+".txt"] = list
	}

	errList := make([]error, 0, len(loadFiles))
	for in := range loadFiles {
		errList = append(errList, k.Run(in, ToOutname(in), workers))
	}

	return errors.Join(errList...)
}

func (k *Krot) Run(in, out string, workers int) error {
	if k.parseOnly {
		workers = workers * 10
	}
	jobs, err := readJobs(in, k.maxChars)
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

	if k.shuffle {
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
				err := k.check(j.uri)
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

	_print := func() {
		processed := ok + fail
		fmt.Fprintf(os.Stderr, "\r%d/%d | ok %d | failed %d", processed, total, ok, fail)
	}

	for r := range resch {
		if r.err != nil {
			fail++
			_print()
			slog.Debug("proxy check failed", "line", r.line, "uri", r.uri, "error", r.err)
			continue
		}

		ok++
		_print()
		if !k.parseOnly {
			slog.Info("proxy check ok", "line", r.line, "uri", r.uri)
		}
		if _, err := _out.WriteString(r.uri + "\n"); err != nil {
			return fmt.Errorf("failed to write output %s in line %d: %w", out, r.line, err)
		}
		if err := _out.Sync(); err != nil {
			return fmt.Errorf("failed to sync output %s in line %d: %w", out, r.line, err)
		}
	}
	if total > 0 {
		fmt.Fprintln(os.Stderr)
	}

	slog.Info("proxy checking finished",
		"total", total,
		"ok", ok,
		"failed", fail,
	)

	return nil
}

func readJobs(in string, maxChars int) ([]job, error) {
	_in, err := os.Open(in)
	if err != nil {
		return nil, fmt.Errorf("failed to open input file %s: %w", in, err)
	}
	defer _in.Close()

	scanner := bufio.NewScanner(_in)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	line := 0
	jobs := make([]job, 0)

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
		if utf8.RuneCountInString(uri) > maxChars {
			slog.Debug("skipping so long line", "line", line)
			continue
		}

		jobs = append(jobs, job{line: line, uri: uri})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read input %s in line %d: %w", in, line, err)
	}

	return jobs, nil
}

func (k *Krot) check(rawURI string) error {
	u, err := url.Parse(rawURI)
	if err != nil {
		return fmt.Errorf("invalid uri: %w", err)
	}

	var (
		scheme = strings.ToLower(u.Scheme)
		host   = strings.ToLower(u.Host)
	)

	if isCheckerScheme(scheme) {
		return checker.Check(rawURI, k.timeout, k.parseOnly)
	}

	if isMTProtoURI(scheme, host, u.Path) {
		return mtproto.Check(rawURI, k.timeout, k.parseOnly)
	}

	return fmt.Errorf("unsupported proxy url format: %q", rawURI)
}

func isCheckerScheme(scheme string) bool {
	switch scheme {
	case "vless", "vmess", "trojan", "ss":
		return true
	default:
		return false
	}
}

func isMTProtoURI(scheme, host, path string) bool {
	if scheme == "tg" && host == "proxy" {
		return true
	}

	if (scheme == "http" || scheme == "https") && (host == "t.me" || host == "www.t.me") && path == "/proxy" {
		return true
	}

	return false
}
