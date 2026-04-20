package mtproto

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
)

func Check(uri string, timeout time.Duration, parseOnly bool) error {
	slog.Debug("mtproto: parse proxy URI")

	proxyAddr, secret, err := parseProxyURI(uri)
	if err != nil {
		slog.Warn("mtproto: invalid proxy URI", "error", err)
		return err
	}
	slog.Debug("mtproto: parsed proxy URI", "addr", proxyAddr, "secret_len", len(secret))

	if parseOnly {
		return nil
	}

	dialer := &net.Dialer{Timeout: timeout}
	slog.Debug("mtproto: creating resolver", "addr", proxyAddr)
	resolver, err := dcs.MTProxy(proxyAddr, secret, dcs.MTProxyOptions{Dial: dialer.DialContext})
	if err != nil {
		slog.Warn("mtproto: resolver creation failed", "addr", proxyAddr, "error", err)
		return fmt.Errorf("invalid proxy config: %w", err)
	}

	client := telegram.NewClient(telegram.TestAppID, telegram.TestAppHash, telegram.Options{
		Resolver:        resolver,
		DialTimeout:     timeout,
		ExchangeTimeout: timeout,
		// MaxRetries:      1,
		// RetryInterval:   100 * time.Millisecond,
	})
	slog.Debug("mtproto: starting nearest DC request", "addr", proxyAddr)

	runErrCh := make(chan error, 1)
	go func() {
		runErrCh <- client.Run(context.Background(), func(ctx context.Context) error {
			_, err := client.API().HelpGetNearestDC(ctx)
			return err
		})
	}()

	select {
	case err := <-runErrCh:
		if err != nil {
			slog.Warn("mtproto: proxy check failed", "addr", proxyAddr, "error", err)
			return fmt.Errorf("proxy check failed: %w", err)
		}
	case <-time.After(timeout):
		slog.Warn("mtproto: proxy check timeout", "addr", proxyAddr, "timeout", timeout)
		return fmt.Errorf("proxy check timeout after %s", timeout)
	}
	slog.Info("mtproto: proxy verified", "addr", proxyAddr)

	return nil
}

func parseProxyURI(rawURI string) (string, []byte, error) {
	u, err := url.Parse(rawURI)
	if err != nil {
		return "", nil, fmt.Errorf("invalid proxy URL: %w", err)
	}

	switch {
	case u.Scheme == "tg" && strings.EqualFold(u.Host, "proxy"):
	case (u.Scheme == "http" || u.Scheme == "https") &&
		(strings.EqualFold(u.Host, "t.me") || strings.EqualFold(u.Host, "www.t.me")) &&
		u.Path == "/proxy":
	default:
		return "", nil, fmt.Errorf("unsupported proxy URL format: %q", rawURI)
	}

	query := u.Query()
	server := strings.TrimSpace(query.Get("server"))
	if server == "" {
		return "", nil, fmt.Errorf("server is required")
	}

	port, err := strconv.Atoi(query.Get("port"))
	if err != nil || port < 1 || port > 65535 {
		return "", nil, fmt.Errorf("invalid port")
	}

	secretRaw := strings.TrimSpace(query.Get("secret"))
	if secretRaw == "" {
		return "", nil, fmt.Errorf("secret is required")
	}

	secret, err := decodeSecret(secretRaw)
	if err != nil {
		return "", nil, err
	}

	return net.JoinHostPort(server, strconv.Itoa(port)), secret, nil
}

func decodeSecret(secret string) ([]byte, error) {
	if isHex(secret) {
		if len(secret)%2 != 0 {
			return nil, fmt.Errorf("invalid secret: odd hex length")
		}

		b, err := hex.DecodeString(secret)
		if err != nil {
			return nil, fmt.Errorf("invalid secret: %w", err)
		}

		return b, nil
	}

	normalized := strings.NewReplacer("-", "+", "_", "/").Replace(secret)
	if rem := len(normalized) % 4; rem != 0 {
		normalized += strings.Repeat("=", 4-rem)
	}

	b, err := base64.StdEncoding.DecodeString(normalized)
	if err != nil {
		return nil, fmt.Errorf("invalid secret: %w", err)
	}
	if len(b) == 0 {
		return nil, fmt.Errorf("invalid secret: empty")
	}

	return b, nil
}

func isHex(s string) bool {
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return len(s) > 0
}
