package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"

	"krot/config/config"
	"krot/internal/krot"

	"krot/pkg/loader"
)

const (
	_ int = iota
	initCode
	fatalCode
	inputCode
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
	defaults := config.Default()

	var (
		confPath = "krot.yaml"
		_config  *config.Config
	)

	rootCmd := &cobra.Command{
		Use:           "krot",
		Short:         "Concurrent proxy checker",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			var err error
			_config, err = config.FromCobra(confPath, cmd)
			if err != nil {
				return newExitError(initCode, err)
			}

			return newExitError(initCode, configureLogger(_config.Runtime.Level, _config.Runtime.Log))
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := validateConfig(_config); err != nil {
				return newExitError(initCode, err)
			}

			slog.Info("starting proxy checker",
				"input", _config.Runtime.In,
				"out", _config.Runtime.Out,
				"level", _config.Runtime.Level,
				"timeout", _config.Runtime.Timeout.String(),
				"workers", _config.Runtime.Workers,
			)

			checker := krot.New(_config.Runtime.Timeout, _config.Runtime.Parse, _config.Runtime.Chars, _config.Runtime.Shuf)
			if _config.Runtime.Load {
				loadFiles := make(map[string][]string, len(_config.Urls))
				for key, urls := range _config.Urls {
					loadFiles[key+".txt"] = urls
				}

				saveErrs := make([]error, 0, len(loadFiles))
				for filename, urls := range loadFiles {
					saveErrs = append(saveErrs, loader.Save(filename, urls))
				}
				if err := errors.Join(saveErrs...); err != nil {
					slog.Error("failed to save one or more url files", "error", err)
				}

				parseChecker := krot.New(_config.Runtime.Timeout, true, _config.Runtime.Chars, _config.Runtime.Shuf)
				parseErrs := make([]error, 0, len(loadFiles))
				for filename := range loadFiles {
					parseErrs = append(parseErrs, parseChecker.Run(filename, filename, _config.Runtime.Workers*3))
				}
				if err := errors.Join(parseErrs...); err != nil {
					slog.Error("failed to parse one or more url files", "error", err)
				}

				return nil
			}

			if _config.Runtime.In == "" {
				return newExitError(inputCode, fmt.Errorf("source file not set: use runtime.in or --in"))
			}

			if _config.Runtime.Pipeline {
				return newExitError(fatalCode, checker.Pipeline(_config.Runtime.Workers, _config.Urls))
			}

			out := _config.Runtime.Out
			if out == "" {
				out = krot.ToOutname(_config.Runtime.In)
			}

			return newExitError(fatalCode, checker.Run(_config.Runtime.In, out, _config.Runtime.Workers))
		},
	}

	rootCmd.Flags().StringVar(&confPath, "config", "krot.yaml", "path to config file")
	rootCmd.Flags().String("in", defaults.Runtime.In, "input file")
	rootCmd.Flags().String("out", defaults.Runtime.Out, "output file")
	rootCmd.Flags().String("log", defaults.Runtime.Log, "log file path")
	rootCmd.Flags().String("level", defaults.Runtime.Level, "log level: debug|info|warn|error")
	rootCmd.Flags().Duration("timeout", defaults.Runtime.Timeout, "proxy check timeout (e.g. 10s, 1m)")
	rootCmd.Flags().Int("workers", defaults.Runtime.Workers, "number of concurrent workers")
	rootCmd.Flags().Bool("pipeline", defaults.Runtime.Pipeline, "start all checks")
	rootCmd.Flags().Bool("shuf", defaults.Runtime.Shuf, "shuffle input lines")
	rootCmd.Flags().Bool("parse", defaults.Runtime.Parse, "parse only url don't send requests")
	rootCmd.Flags().Int("chars", defaults.Runtime.Chars, "max chars in one line")
	rootCmd.Flags().Bool("load", defaults.Runtime.Load, "download source files")

	return rootCmd
}

func validateConfig(_config *config.Config) error {
	if _config.Runtime.Timeout <= 0 {
		return fmt.Errorf("invalid timeout %q: must be > 0", _config.Runtime.Timeout.String())
	}
	if _config.Runtime.Workers <= 0 {
		return fmt.Errorf("invalid workers %d: must be > 0", _config.Runtime.Workers)
	}
	if _config.Runtime.Load {
		if len(_config.Urls["vless"]) == 0 {
			return fmt.Errorf("urls.vless is empty")
		}
		if len(_config.Urls["vless_small"]) == 0 {
			return fmt.Errorf("urls.vless_small is empty")
		}
		if len(_config.Urls["mtproto"]) == 0 {
			return fmt.Errorf("urls.mtproto is empty")
		}
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
