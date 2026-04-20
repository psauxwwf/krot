package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"krot/internal/krot"

	"krot/pkg/loader"
)

type options struct {
	in       string
	out      string
	log      string
	level    string
	timeout  time.Duration
	workers  int
	pipeline bool
	shuf     bool
	parse    bool
	chars    int
	load     bool
}

const (
	_ int = iota
	initCode
	fatalCode
)

type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string {
	if e == nil || e.err == nil {
		return ""
	}

	return e.err.Error()
}

func (e *exitError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.err
}

func newExitError(code int, err error) error {
	if err == nil {
		return nil
	}

	return &exitError{code: code, err: err}
}

func main() {
	if err := fang.Execute(context.Background(), rootCmd(), fang.WithoutVersion()); err != nil {
		if err, ok := errors.AsType[*exitError](err); ok {
			fmt.Fprintln(os.Stderr, err.err)
			os.Exit(err.code)
		}

		fmt.Fprintln(os.Stderr, err)
		os.Exit(fatalCode)
	}
}

func rootCmd() *cobra.Command {
	opts := options{}

	rootCmd := &cobra.Command{
		Use:           "krot",
		Short:         "Concurrent proxy checker",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return newExitError(initCode, configureLogger(opts.level, opts.log))
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validateOptions(opts); err != nil {
				return newExitError(initCode, err)
			}

			slog.Info("starting proxy checker",
				"input", opts.in,
				"out", opts.out,
				"level", opts.level,
				"timeout", opts.timeout.String(),
				"workers", opts.workers,
			)

			checker := krot.New(opts.timeout, opts.parse, opts.chars, opts.shuf)
			if opts.load {
				if err := errors.Join(
					loader.SaveVless("vless.txt"),
					loader.SaveVlessSmall("vless_small.txt"),
					loader.SaveMtproto("mtproto.txt", nil),
				); err != nil {
					return newExitError(fatalCode, err)
				}

				parseChecker := krot.New(opts.timeout, true, opts.chars, opts.shuf)
				if err := errors.Join(
					parseChecker.Run("vless.txt", "vless.txt", opts.workers*3),
					parseChecker.Run("vless_small.txt", "vless_small.txt", opts.workers*3),
					parseChecker.Run("mtproto.txt", "mtproto.txt", opts.workers*3),
				); err != nil {
					return newExitError(fatalCode, err)
				}

				return nil
			}

			if opts.pipeline {
				return newExitError(fatalCode, checker.Pipeline(opts.workers))
			}

			out := opts.out
			if out == "" {
				out = krot.ToOutname(opts.in)
			}

			return newExitError(fatalCode, checker.Run(opts.in, out, opts.workers))
		},
	}

	rootCmd.Flags().StringVar(&opts.in, "in", "", "input file")
	rootCmd.Flags().StringVar(&opts.out, "out", "", "output file")
	rootCmd.Flags().StringVar(&opts.log, "log", "", "log file path")
	rootCmd.Flags().StringVar(&opts.level, "level", "info", "log level: debug|info|warn|error")
	rootCmd.Flags().DurationVar(&opts.timeout, "timeout", 6*time.Second, "proxy check timeout (e.g. 10s, 1m)")
	rootCmd.Flags().IntVar(&opts.workers, "workers", runtime.NumCPU()*3, "number of concurrent workers")
	rootCmd.Flags().BoolVar(&opts.pipeline, "pipeline", false, "start all checks")
	rootCmd.Flags().BoolVar(&opts.shuf, "shuf", true, "shuffle input lines")
	rootCmd.Flags().BoolVar(&opts.parse, "parse", false, "parse only url don't send requests")
	rootCmd.Flags().IntVar(&opts.chars, "chars", 4096, "max chars in one line")
	rootCmd.Flags().BoolVar(&opts.load, "load", false, "download source files")

	return rootCmd
}

func validateOptions(opts options) error {
	if opts.timeout <= 0 {
		return fmt.Errorf("invalid timeout %q: must be > 0", opts.timeout.String())
	}
	if opts.workers <= 0 {
		return fmt.Errorf("invalid workers %d: must be > 0", opts.workers)
	}

	return nil
}

func configureLogger(levelText, logPath string) error {
	var parsedLevel slog.Level
	if err := parsedLevel.UnmarshalText([]byte(levelText)); err != nil {
		return fmt.Errorf("invalid log level %q: %w", levelText, err)
	}

	stdoutHandler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: parsedLevel,
	})

	if strings.TrimSpace(logPath) == "" {
		slog.SetDefault(slog.New(stdoutHandler))
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return fmt.Errorf("failed to create log dir for %q: %w", logPath, err)
	}

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file %q: %w", logPath, err)
	}

	slog.SetDefault(slog.New(slog.NewMultiHandler(
		stdoutHandler,
		slog.NewJSONHandler(logFile, &slog.HandlerOptions{
			AddSource: true,
			Level:     parsedLevel,
		}),
	)))

	return nil
}
