package checker

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"

	checkermodels "github.com/kutovoys/xray-checker/models"
	checkersubscription "github.com/kutovoys/xray-checker/subscription"
	checkerxray "github.com/kutovoys/xray-checker/xray"

	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"
)

const (
	probeURL1     = "https://cp.cloudflare.com/generate_204"
	probeURL2     = "https://www.gstatic.com/generate_204"
	localBindAddr = "127.0.0.1"
)

type config struct {
	proxy *checkermodels.ProxyConfig
}

func Check(rawURI string, timeout time.Duration, parseOnly bool) error {
	if timeout <= 0 {
		return fmt.Errorf("timeout must be > 0")
	}

	config, err := parse(rawURI)
	if err != nil {
		return err
	}

	if parseOnly {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	port, err := getFreeTCPPort()
	if err != nil {
		return fmt.Errorf("reserve local socks port: %w", err)
	}

	xrayInstance, err := startXray(ctx, config, port)
	if err != nil {
		return err
	}
	defer xrayInstance.Close()

	if err := waitForSOCKS(ctx, port); err != nil {
		return err
	}

	if err := probeThroughSOCKS(ctx, timeout, port); err != nil {
		return err
	}

	return nil
}

func startXray(ctx context.Context, cfg config, socksPort int) (*core.Instance, error) {
	configBytes, err := buildXrayConfig(cfg, socksPort)
	if err != nil {
		return nil, err
	}

	xrayConfig, err := serial.DecodeJSONConfig(bytes.NewReader(configBytes))
	if err != nil {
		return nil, fmt.Errorf("decode xray config: %w", err)
	}

	coreConfig, err := xrayConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("build xray config: %w", err)
	}

	instance, err := core.New(coreConfig)
	if err != nil {
		return nil, fmt.Errorf("create xray instance: %w", err)
	}

	started := make(chan error, 1)
	go func() {
		started <- instance.Start()
	}()

	select {
	case err := <-started:
		if err != nil {
			_ = instance.Close()
			return nil, fmt.Errorf("start xray instance: %w", err)
		}
		return instance, nil
	case <-ctx.Done():
		_ = instance.Close()
		return nil, fmt.Errorf("start xray instance: %w", ctx.Err())
	}
}

func buildXrayConfig(cfg config, socksPort int) ([]byte, error) {
	proxyConfig := *cfg.proxy
	proxyConfig.Index = 0

	generator := checkerxray.NewConfigGenerator()
	configBytes, err := generator.GenerateConfig([]*checkermodels.ProxyConfig{&proxyConfig}, socksPort, "none")
	if err != nil {
		return nil, fmt.Errorf("generate xray config: %w", err)
	}

	return configBytes, nil
}

func probeThroughSOCKS(ctx context.Context, timeout time.Duration, socksPort int) error {
	dialer, err := proxy.SOCKS5("tcp", net.JoinHostPort(localBindAddr, strconv.Itoa(socksPort)), nil, &net.Dialer{Timeout: timeout})
	if err != nil {
		return fmt.Errorf("create socks dialer: %w", err)
	}

	httpTransport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			type contextDialer interface {
				DialContext(context.Context, string, string) (net.Conn, error)
			}
			if d, ok := dialer.(contextDialer); ok {
				return d.DialContext(ctx, network, addr)
			}
			return dialer.Dial(network, addr)
		},
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     true,
		TLSHandshakeTimeout:   timeout,
		ResponseHeaderTimeout: timeout,
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: httpTransport,
	}

	var errs []string
	for _, probeURL := range []string{probeURL1, probeURL2} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
		if err != nil {
			return fmt.Errorf("build probe request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", probeURL, err))
			if ctx.Err() != nil {
				break
			}
			continue
		}
		_ = resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 500 {
			return nil
		}

		errs = append(errs, fmt.Sprintf("%s: unexpected status %d", probeURL, resp.StatusCode))
	}

	if ctx.Err() != nil {
		return fmt.Errorf("probe through xray: %w", ctx.Err())
	}

	return fmt.Errorf("probe through xray failed: %s", strings.Join(errs, "; "))
}

func waitForSOCKS(ctx context.Context, port int) error {
	addr := net.JoinHostPort(localBindAddr, strconv.Itoa(port))
	var lastErr error

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		lastErr = err

		select {
		case <-ctx.Done():
			return fmt.Errorf("wait for local socks listener %s: %w (last error: %v)", addr, ctx.Err(), lastErr)
		case <-ticker.C:
		}
	}
}

func getFreeTCPPort() (int, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort(localBindAddr, "0"))
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("unexpected listener address type %T", listener.Addr())
	}

	return addr.Port, nil
}

func parse(rawURI string) (config, error) {
	parser := checkersubscription.NewParser()

	result, err := parser.Parse(strings.TrimSpace(rawURI))
	if err != nil {
		return config{}, fmt.Errorf("parse proxy uri: %w", err)
	}
	if result == nil || len(result.Configs) == 0 {
		return config{}, fmt.Errorf("parse proxy uri: no configs found")
	}

	proxyConfig := result.Configs[0]
	if proxyConfig == nil {
		return config{}, fmt.Errorf("parse proxy uri: empty config")
	}
	if err := proxyConfig.Validate(); err != nil {
		return config{}, fmt.Errorf("invalid parsed proxy config: %w", err)
	}

	return config{proxy: proxyConfig}, nil
}
