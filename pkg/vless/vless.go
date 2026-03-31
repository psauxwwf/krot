package vless

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/proxy"

	checkermodels "github.com/kutovoys/xray-checker/models"
	checkerxray "github.com/kutovoys/xray-checker/xray"

	xuuid "github.com/xtls/xray-core/common/uuid"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	_ "github.com/xtls/xray-core/main/distro/all"
	xvless "github.com/xtls/xray-core/proxy/vless"
)

const (
	probeURL1     = "https://cp.cloudflare.com/generate_204"
	probeURL2     = "https://www.gstatic.com/generate_204"
	localBindAddr = "127.0.0.1"
)

type config struct {
	proxy *checkermodels.ProxyConfig
}

func CheckWithTimeout(rawURI string, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("timeout must be > 0")
	}

	cfg, err := parse(rawURI)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	port, err := getFreeTCPPort()
	if err != nil {
		return fmt.Errorf("reserve local socks port: %w", err)
	}

	xrayInstance, err := startXray(ctx, cfg, port)
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
	u, err := url.Parse(strings.TrimSpace(rawURI))
	if err != nil {
		return config{}, fmt.Errorf("invalid vless uri: %w", err)
	}
	if !strings.EqualFold(u.Scheme, "vless") {
		return config{}, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if u.User == nil {
		return config{}, fmt.Errorf("vless uuid is required")
	}

	userID := strings.TrimSpace(u.User.Username())
	if _, err := xuuid.ParseString(userID); err != nil {
		return config{}, fmt.Errorf("invalid vless uuid: %w", err)
	}

	host := strings.TrimSpace(u.Hostname())
	if host == "" {
		return config{}, fmt.Errorf("vless host is required")
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return config{}, fmt.Errorf("invalid vless port")
	}

	query := u.Query()
	encryption := strings.ToLower(strings.TrimSpace(query.Get("encryption")))
	if encryption == "" {
		encryption = "none"
	}
	if encryption != "none" {
		return config{}, fmt.Errorf("unsupported vless encryption %q", encryption)
	}

	network := strings.ToLower(strings.TrimSpace(firstNonEmpty(query.Get("type"), query.Get("net"))))
	if network == "" {
		network = "tcp"
	}
	if network == "raw" {
		network = "tcp"
	}
	if !isSupportedNetwork(network) {
		return config{}, fmt.Errorf("unsupported network type %q", network)
	}

	security := strings.ToLower(strings.TrimSpace(query.Get("security")))
	if security == "" {
		security = "none"
	}
	if !isSupportedSecurity(security) {
		return config{}, fmt.Errorf("unsupported security mode %q", security)
	}

	sni := firstNonEmpty(query.Get("sni"), query.Get("serverName"), query.Get("peer"))
	if sni == "" && security != "none" {
		sni = host
	}

	flow := strings.TrimSpace(query.Get("flow"))
	if flow != "" && flow != xvless.XRV {
		return config{}, fmt.Errorf("unsupported flow %q", flow)
	}

	fingerprint := firstNonEmpty(query.Get("fp"), query.Get("fingerprint"), query.Get("client-fingerprint"))
	if security == "reality" && fingerprint == "" {
		fingerprint = "chrome"
	}

	publicKey := firstNonEmpty(query.Get("pbk"), query.Get("publicKey"))
	if security == "reality" && publicKey == "" {
		return config{}, fmt.Errorf("public key (pbk) is required for reality")
	}

	shortID := firstNonEmpty(query.Get("sid"), query.Get("shortId"))
	serviceName := firstNonEmpty(query.Get("serviceName"), query.Get("service_name"))
	headerHost := firstNonEmpty(query.Get("host"), query.Get("authority"))
	headerType := strings.ToLower(strings.TrimSpace(query.Get("headerType")))
	path := strings.TrimSpace(query.Get("path"))

	proxyConfig := &checkermodels.ProxyConfig{
		Protocol:      "vless",
		Server:        host,
		Port:          port,
		Name:          u.Fragment,
		Security:      security,
		Type:          network,
		UUID:          userID,
		Flow:          flow,
		Encryption:    encryption,
		HeaderType:    headerType,
		Path:          path,
		Host:          headerHost,
		SNI:           sni,
		Fingerprint:   fingerprint,
		PublicKey:     publicKey,
		ShortID:       shortID,
		Mode:          strings.TrimSpace(query.Get("mode")),
		ServiceName:   serviceName,
		AllowInsecure: parseBool(query.Get("allowInsecure")),
		ALPN:          splitCSV(query.Get("alpn")),
		MultiMode:     parseBool(query.Get("multiMode")),
	}

	if network == "splithttp" || network == "xhttp" {
		proxyConfig.Path = normalizedPath(path)
	}
	if network == "ws" || network == "httpupgrade" {
		proxyConfig.Path = normalizedPath(path)
	}

	if err := validateTransport(proxyConfig); err != nil {
		return config{}, err
	}
	if err := proxyConfig.Validate(); err != nil {
		return config{}, fmt.Errorf("invalid parsed vless config: %w", err)
	}

	return config{proxy: proxyConfig}, nil
}

func validateTransport(cfg *checkermodels.ProxyConfig) error {
	if cfg.Security == "reality" && cfg.PublicKey == "" {
		return fmt.Errorf("public key (pbk) is required for reality")
	}

	switch cfg.Type {
	case "grpc":
		if cfg.ServiceName == "" {
			return fmt.Errorf("grpc serviceName is required")
		}
	case "tcp":
		if cfg.HeaderType != "" && cfg.HeaderType != "none" && cfg.HeaderType != "http" {
			return fmt.Errorf("unsupported tcp header type %q", cfg.HeaderType)
		}
	}

	return nil
}

func isSupportedNetwork(network string) bool {
	switch network {
	case "tcp", "ws", "grpc", "http", "h2", "httpupgrade", "splithttp", "xhttp":
		return true
	default:
		return false
	}
}

func isSupportedSecurity(security string) bool {
	switch security {
	case "none", "tls", "reality":
		return true
	default:
		return false
	}
}

func parseBool(v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	return v == "1" || v == "true"
}

func splitCSV(v string) []string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizedPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
