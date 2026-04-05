package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"krot/internal/krot"
)

var (
	in       = flag.String("in", "vless.txt", "input file")
	out      = flag.String("out", "", "output file")
	level    = flag.String("level", "info", "log level: debug|info|warn|error")
	timeout  = flag.Duration("timeout", 10*time.Second, "proxy check timeout (e.g. 10s, 1m)")
	workers  = flag.Int("workers", runtime.NumCPU()*3, "number of concurrent workers")
	pipeline = flag.Bool("pipeline", false, "start all checks")
	shuf     = flag.Bool("shuf", false, "shuffle input lines")
	parse    = flag.Bool("parse", false, "parse only url don't send requests")
	chars    = flag.Int("chars", 8192, "max chars in one line")
)

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
			slog.NewJSONHandler(logFile, &slog.HandlerOptions{
				AddSource: true,
				Level:     parsedLevel,
			}),
		),
	)
	slog.SetDefault(log)
	slog.Info("starting proxy checker",
		"input", *in, "out", *out, "level", parsedLevel.String(), "timeout", timeout.String(), "workers", *workers)

	_krot := krot.New(*timeout, *parse, *chars, *shuf)

	if *pipeline {
		if err := _krot.Pipeline(*workers); err != nil {
			slog.Error("fatal error", "error", err)
			os.Exit(5)
		}
		return
	}

	if *out == "" {
		*out = krot.ToOutname(*in)
	}
	if err := _krot.Run(*in, *out, *workers); err != nil {
		slog.Error("fatal error", "error", err)
		os.Exit(5)
	}
}
