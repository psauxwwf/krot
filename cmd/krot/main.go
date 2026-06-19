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

	"krot/internal/config"
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
	_config := config.DefaultRuntime()
	urlsPath := "urls.yaml"

	rootCmd := &cobra.Command{
		Use:           "krot",
		Short:         "Concurrent proxy checker",
		SilenceUsage:  true,
		SilenceErrors: true,
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			return runCheck(_config)
		},
	}

	rootCmd.PersistentFlags().StringVar(&urlsPath, "urls", urlsPath, "path to urls config file")
	rootCmd.PersistentFlags().StringVar(&_config.In, "in", _config.In, "input file")
	rootCmd.PersistentFlags().StringVar(&_config.Out, "out", _config.Out, "output file")
	rootCmd.PersistentFlags().StringVar(&_config.Log, "log-path", _config.Log, "log file path")
	rootCmd.PersistentFlags().StringVar(&_config.Level, "log-level", _config.Level, "log level: debug|info|warn|error")
	rootCmd.PersistentFlags().DurationVar(&_config.Timeout, "timeout", _config.Timeout, "proxy check timeout (e.g. 10s, 1m)")
	rootCmd.PersistentFlags().IntVar(&_config.Workers, "workers", _config.Workers, "number of concurrent workers")
	rootCmd.PersistentFlags().IntVar(&_config.Chars, "chars", _config.Chars, "max chars in one line")

	rootCmd.AddCommand(
		newSaveCmd(&_config, &urlsPath),
		newLoadCmd(&_config, &urlsPath),
		newParseCmd(&_config),
		newPipelineCmd(&_config, &urlsPath),
	)

	return rootCmd
}

func newSaveCmd(_config *config.Runtime, urlsPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "save",
		Short: "Save default URL lists to urls config",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := configureLogger(_config.Level, _config.Log); err != nil {
				return newExitError(initCode, err)
			}
			if err := config.Save(*urlsPath); err != nil {
				return newExitError(fatalCode, fmt.Errorf("failed to save urls config %q: %w", *urlsPath, err))
			}

			slog.Info("urls config saved", "path", *urlsPath)
			return nil
		},
	}
}

func newLoadCmd(_config *config.Runtime, urlsPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "load",
		Short: "Load source files from urls config",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runLoad(*_config, *urlsPath)
		},
	}
}

func newParseCmd(_config *config.Runtime) *cobra.Command {
	return &cobra.Command{
		Use:   "parse",
		Short: "Parse and validate proxies from input file",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			parseCfg := *_config
			parseCfg.Parse = true
			return runCheck(parseCfg)
		},
	}
}

func newPipelineCmd(_config *config.Runtime, urlsPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "pipeline",
		Short: "Run built-in checks for predefined files",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runPipeline(*_config, *urlsPath)
		},
	}
}

func validateRuntime(_config config.Runtime) error {
	if _config.Timeout <= 0 {
		return fmt.Errorf("invalid timeout %q: must be > 0", _config.Timeout.String())
	}
	if _config.Workers <= 0 {
		return fmt.Errorf("invalid workers %d: must be > 0", _config.Workers)
	}

	return nil
}

func validateURLs(urls config.Urls) error {
	if len(urls["vless"]) == 0 {
		return fmt.Errorf("urls.vless is empty")
	}
	if len(urls["vless_small"]) == 0 {
		return fmt.Errorf("urls.vless_small is empty")
	}
	if len(urls["mtproto"]) == 0 {
		return fmt.Errorf("urls.mtproto is empty")
	}

	return nil
}

func runCheck(_config config.Runtime) error {
	if err := configureLogger(_config.Level, _config.Log); err != nil {
		return newExitError(initCode, err)
	}
	if err := validateRuntime(_config); err != nil {
		return newExitError(initCode, err)
	}
	if _config.In == "" {
		return newExitError(inputCode, fmt.Errorf("source file not set: use --in"))
	}

	slog.Info("starting proxy checker",
		"input", _config.In,
		"out", _config.Out,
		"level", _config.Level,
		"timeout", _config.Timeout.String(),
		"workers", _config.Workers,
		"parse", _config.Parse,
	)

	out := _config.Out
	if out == "" {
		out = krot.ToOutname(_config.In)
	}

	checker := krot.New(_config.Timeout, _config.Parse, _config.Chars)
	return newExitError(fatalCode, checker.Run(_config.In, out, _config.Workers))
}

func runLoad(_config config.Runtime, urlsPath string) error {
	if err := configureLogger(_config.Level, _config.Log); err != nil {
		return newExitError(initCode, err)
	}
	if err := validateRuntime(_config); err != nil {
		return newExitError(initCode, err)
	}

	urlsCfg, err := loadURLsConfig(urlsPath)
	if err != nil {
		return newExitError(initCode, err)
	}
	if err := validateURLs(urlsCfg.Urls); err != nil {
		return newExitError(initCode, err)
	}

	loadFiles := make(map[string][]string, len(urlsCfg.Urls))
	for key, urls := range urlsCfg.Urls {
		loadFiles[key+".txt"] = urls
	}

	saveErrs := make([]error, 0, len(loadFiles))
	for filename, urls := range loadFiles {
		saveErrs = append(saveErrs, loader.Save(filename, urls))
	}
	if err := errors.Join(saveErrs...); err != nil {
		slog.Error("failed to save one or more url files", "error", err)
	}

	parseChecker := krot.New(_config.Timeout, true, _config.Chars)
	parseErrs := make([]error, 0, len(loadFiles))
	for filename := range loadFiles {
		parseErrs = append(parseErrs, parseChecker.Run(filename, filename, _config.Workers*3))
	}
	if err := errors.Join(parseErrs...); err != nil {
		slog.Error("failed to parse one or more url files", "error", err)
	}

	return nil
}

func runPipeline(_config config.Runtime, urlsPath string) error {
	if err := configureLogger(_config.Level, _config.Log); err != nil {
		return newExitError(initCode, err)
	}
	if err := validateRuntime(_config); err != nil {
		return newExitError(initCode, err)
	}

	urlsCfg, err := loadURLsConfig(urlsPath)
	if err != nil {
		return newExitError(initCode, err)
	}
	if err := validateURLs(urlsCfg.Urls); err != nil {
		return newExitError(initCode, err)
	}

	slog.Info("starting pipeline",
		"level", _config.Level,
		"timeout", _config.Timeout.String(),
		"workers", _config.Workers,
	)

	checker := krot.New(_config.Timeout, false, _config.Chars)
	return newExitError(fatalCode, checker.Pipeline(_config.Workers, urlsCfg.Urls))
}

func loadURLsConfig(path string) (*config.Config, error) {
	if err := ensureURLsConfig(path); err != nil {
		return nil, err
	}

	return config.New(path)
}

func ensureURLsConfig(path string) error {
	_, err := config.New(path)
	if err == nil {
		return nil
	}
	if !errors.Is(err, config.ErrNotExists) {
		return err
	}
	if err := config.Save(path); err != nil {
		return fmt.Errorf("failed to create default urls config %q: %w", path, err)
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
