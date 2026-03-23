package vless

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	xuuid "github.com/xtls/xray-core/common/uuid"
	xvless "github.com/xtls/xray-core/proxy/vless"
	"github.com/xtls/xray-core/proxy/vless/encoding"
	xreality "github.com/xtls/xray-core/transport/internet/reality"
	xraytls "github.com/xtls/xray-core/transport/internet/tls"
)

const defaultTimeout = 30 * time.Second

type config struct {
	uuid      xuuid.UUID
	host      string
	port      int
	network   string
	wsPath    string
	wsHost    string
	security  string
	sni       string
	flow      string
	fp        string
	publicKey string
	shortID   string
}

func Check(uri string) error {
	return CheckWithTimeout(uri, defaultTimeout)
}

func CheckWithTimeout(rawURI string, timeout time.Duration) error {
	if timeout <= 0 {
		return fmt.Errorf("timeout must be > 0")
	}

	cfg, err := parse(rawURI)
	if err != nil {
		return err
	}

	if cfg.network == "ws" {
		return checkWS(cfg, timeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	dialer := &net.Dialer{Timeout: timeout}
	addr := net.JoinHostPort(cfg.host, strconv.Itoa(cfg.port))
	baseConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("dial %s failed: %w", addr, err)
	}
	defer baseConn.Close()

	_ = baseConn.SetDeadline(time.Now().Add(timeout))

	securedConn, err := secureConn(ctx, baseConn, cfg)
	if err != nil {
		return err
	}

	if err := sendVLESSProbe(securedConn, cfg); err != nil {
		return err
	}

	return nil
}

func secureConn(ctx context.Context, conn net.Conn, cfg config) (net.Conn, error) {
	switch cfg.security {
	case "", "none":
		return conn, nil
	case "tls":
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName:         cfg.sni,
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			return nil, fmt.Errorf("tls handshake failed: %w", err)
		}
		return tlsConn, nil
	case "reality":
		if cfg.publicKey == "" {
			return nil, fmt.Errorf("public key (pbk) is required for reality")
		}

		fingerprint := strings.ToLower(strings.TrimSpace(cfg.fp))
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		if xraytls.GetFingerprint(fingerprint) == nil {
			return nil, fmt.Errorf("unknown reality fingerprint %q", cfg.fp)
		}

		publicKey, err := base64.RawURLEncoding.DecodeString(cfg.publicKey)
		if err != nil || len(publicKey) != 32 {
			return nil, fmt.Errorf("invalid reality public key")
		}

		if len(cfg.shortID) > 16 {
			return nil, fmt.Errorf("invalid reality short id: too long")
		}
		shortID := make([]byte, 8)
		if _, err := hex.Decode(shortID, []byte(cfg.shortID)); err != nil {
			return nil, fmt.Errorf("invalid reality short id: %w", err)
		}

		port, err := xnet.PortFromInt(uint32(cfg.port))
		if err != nil {
			return nil, fmt.Errorf("invalid port: %w", err)
		}

		realityConn, err := xreality.UClient(conn, &xreality.Config{
			Fingerprint: fingerprint,
			ServerName:  cfg.sni,
			PublicKey:   publicKey,
			ShortId:     shortID,
		}, ctx, xnet.TCPDestination(xnet.ParseAddress(cfg.host), port))
		if err != nil {
			return nil, fmt.Errorf("reality handshake failed: %w", err)
		}

		return realityConn, nil
	default:
		return nil, fmt.Errorf("unsupported security mode %q", cfg.security)
	}
}

func sendVLESSProbe(conn net.Conn, cfg config) error {
	request, req, err := buildVLESSRequest(cfg)
	if err != nil {
		return err
	}

	if _, err := conn.Write(request); err != nil {
		return fmt.Errorf("send vless request failed: %w", err)
	}

	if _, err := encoding.DecodeResponseHeader(conn, req); err != nil {
		return fmt.Errorf("read vless response failed: %w", err)
	}

	return nil
}

func checkWS(cfg config, timeout time.Duration) error {
	wsScheme := "ws"
	if cfg.security == "tls" {
		wsScheme = "wss"
	}
	if cfg.security != "none" && cfg.security != "tls" {
		return fmt.Errorf("unsupported ws security mode %q", cfg.security)
	}

	path := cfg.wsPath
	if path == "" {
		path = "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	u := url.URL{
		Scheme: wsScheme,
		Host:   net.JoinHostPort(cfg.host, strconv.Itoa(cfg.port)),
		Path:   path,
	}

	headers := http.Header{}
	if cfg.wsHost != "" {
		headers.Set("Host", cfg.wsHost)
	}

	dialer := websocket.Dialer{HandshakeTimeout: timeout}
	if wsScheme == "wss" {
		dialer.TLSClientConfig = &tls.Config{
			ServerName:         cfg.sni,
			InsecureSkipVerify: true,
			MinVersion:         tls.VersionTLS12,
		}
	}

	conn, _, err := dialer.Dial(u.String(), headers)
	if err != nil {
		return fmt.Errorf("websocket dial failed: %w", err)
	}
	defer conn.Close()

	request, req, err := buildVLESSRequest(cfg)
	if err != nil {
		return err
	}

	if err := conn.SetWriteDeadline(time.Now().Add(timeout)); err != nil {
		return fmt.Errorf("set ws write deadline failed: %w", err)
	}
	if err := conn.WriteMessage(websocket.BinaryMessage, request); err != nil {
		return fmt.Errorf("send vless over websocket failed: %w", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return fmt.Errorf("set ws read deadline failed: %w", err)
	}
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		return fmt.Errorf("read websocket response failed: %w", err)
	}
	if msgType != websocket.BinaryMessage {
		return fmt.Errorf("unexpected websocket message type %d", msgType)
	}

	if _, err := encoding.DecodeResponseHeader(bytes.NewReader(data), req); err != nil {
		return fmt.Errorf("read vless response failed: %w", err)
	}

	return nil
}

func buildVLESSRequest(cfg config) ([]byte, *protocol.RequestHeader, error) {
	account := &xvless.MemoryAccount{
		ID:         protocol.NewID(cfg.uuid),
		Flow:       cfg.flow,
		Encryption: "none",
	}

	targetHost := cfg.sni
	if targetHost == "" {
		targetHost = "www.cloudflare.com"
	}

	targetPort, err := xnet.PortFromInt(443)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid probe port: %w", err)
	}

	req := requestHeader(cfg)
	req.Address = xnet.ParseAddress(targetHost)
	req.Port = targetPort
	req.User = &protocol.MemoryUser{Account: account}

	addons := &encoding.Addons{}
	if cfg.flow == xvless.XRV {
		addons.Flow = xvless.XRV
	}

	var buf bytes.Buffer
	if err := encoding.EncodeRequestHeader(&buf, req, addons); err != nil {
		return nil, nil, fmt.Errorf("encode vless request failed: %w", err)
	}

	return buf.Bytes(), req, nil
}

func requestHeader(_ config) *protocol.RequestHeader {
	return &protocol.RequestHeader{
		Version: encoding.Version,
		Command: protocol.RequestCommandTCP,
	}
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

	userID, err := xuuid.ParseString(strings.TrimSpace(u.User.Username()))
	if err != nil {
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

	q := u.Query()
	if encryption := strings.TrimSpace(q.Get("encryption")); encryption != "" && encryption != "none" {
		return config{}, fmt.Errorf("unsupported vless encryption %q", encryption)
	}

	network := strings.ToLower(strings.TrimSpace(q.Get("type")))
	if network == "" {
		network = "tcp"
	}
	if network != "tcp" && network != "ws" {
		return config{}, fmt.Errorf("unsupported network type %q", network)
	}

	security := strings.ToLower(strings.TrimSpace(q.Get("security")))
	if security == "" {
		security = "none"
	}

	sni := strings.TrimSpace(q.Get("sni"))
	if sni == "" {
		sni = host
	}

	flow := strings.TrimSpace(q.Get("flow"))
	if flow != "" && flow != xvless.XRV {
		return config{}, fmt.Errorf("unsupported flow %q", flow)
	}

	return config{
		uuid:      userID,
		host:      host,
		port:      port,
		network:   network,
		wsPath:    strings.TrimSpace(q.Get("path")),
		wsHost:    strings.TrimSpace(q.Get("host")),
		security:  security,
		sni:       sni,
		flow:      flow,
		fp:        strings.TrimSpace(q.Get("fp")),
		publicKey: strings.TrimSpace(q.Get("pbk")),
		shortID:   strings.TrimSpace(q.Get("sid")),
	}, nil
}
