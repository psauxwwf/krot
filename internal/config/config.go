package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	defaultRuntime = Runtime{
		Out:     "",
		Log:     "krot.json",
		Level:   "info",
		Timeout: 6 * time.Second,
		Workers: runtime.NumCPU() * 3,
		Parse:   false,
		Chars:   4096,
	}
	defaultURLs = Config{
		URLs: URLs{
			"vless": []string{
				"https://raw.githubusercontent.com/zieng2/wl/main/vless_universal.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/1.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/2.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/3.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/4.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/5.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/6.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/7.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/8.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/9.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/10.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/11.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/12.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/13.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/14.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/15.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/16.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/17.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/18.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/19.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/20.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/21.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/22.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/23.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/24.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/25.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/26.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/BLACK_VLESS_RUS_mobile.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/BLACK_SS+All_RUS.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/Vless-Reality-White-Lists-Rus-Mobile.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/Vless-Reality-White-Lists-Rus-Mobile-2.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/BLACK_VLESS_RUS.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/WHITE-CIDR-RU-all.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/WHITE-CIDR-RU-checked.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/WHITE-SNI-RU-all.txt",
				"https://raw.githubusercontent.com/V2RayRoot/V2RayConfig/refs/heads/main/Config/vmess.txt",
				"https://raw.githubusercontent.com/V2RayRoot/V2RayConfig/refs/heads/main/Config/shadowsocks.txt",
				// "https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/default/1.txt",
				// "https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/default/6.txt",
				// "https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/default/22.txt",
				// "https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/default/23.txt",
				// "https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/default/24.txt",
				// "https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/default/25.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-all.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/raw/bypass-all-raw.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-1.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-2.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-3.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-4.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-5.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-6.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-7.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-8.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-9.txt",
				"https://github.com/Epodonios/v2ray-configs/raw/refs/heads/main/All_Configs_Sub.txt",
			},
			"vless_small": []string{
				"https://raw.githubusercontent.com/zieng2/wl/main/vless_universal.txt",
				"https://github.com/AvenCores/goida-vpn-configs/raw/refs/heads/main/githubmirror/26.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/BLACK_VLESS_RUS_mobile.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/BLACK_SS+All_RUS.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/Vless-Reality-White-Lists-Rus-Mobile.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/Vless-Reality-White-Lists-Rus-Mobile-2.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/BLACK_VLESS_RUS.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/WHITE-CIDR-RU-all.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/WHITE-CIDR-RU-checked.txt",
				"https://raw.githubusercontent.com/igareck/vpn-configs-for-russia/refs/heads/main/WHITE-SNI-RU-all.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-all.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/raw/bypass-all-raw.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-1.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-2.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-3.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-4.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-5.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-6.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-7.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-8.txt",
				"https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/bypass/bypass-9.txt",
				"https://github.com/Epodonios/v2ray-configs/raw/refs/heads/main/All_Configs_Sub.txt",
			},
			"mtproto": []string{
				"https://raw.githubusercontent.com/SoliSpirit/mtproto/master/all_proxies.txt",
				"https://raw.githubusercontent.com/DepMSK37/proxy-list/refs/heads/main/verified/proxy_all_verified.txt",
				"https://raw.githubusercontent.com/devho3ein/tg-proxy/refs/heads/main/proxys/All_Proxys.txt",
				"https://raw.githubusercontent.com/V2RayRoot/V2RayConfig/refs/heads/main/Config/proxies.txt",
				"https://raw.githubusercontent.com/FLAT447/v2ray-lists/refs/heads/main/blacklist.txt",
				"https://raw.githubusercontent.com/FLAT447/v2ray-lists/refs/heads/main/whitelist.txt",
				"https://raw.githubusercontent.com/Argh94/Proxy-List/refs/heads/main/HTTPS.txt",
				"https://raw.githubusercontent.com/Argh94/Proxy-List/refs/heads/main/MTProto.txt",
				"https://raw.githubusercontent.com/kort0881/telegram-proxy-collector/refs/heads/main/proxy_all.txt",
				// "https://raw.githubusercontent.com/whoahaow/rjsxrd/refs/heads/main/githubmirror/tg-proxy/MTProto.txt",
			},
		},
	}
	ErrNotExists = fmt.Errorf("urls config not found: %w", os.ErrNotExist)
)

type Config struct {
	URLs URLs `yaml:"urls"`
}

type Runtime struct {
	Out     string        `yaml:"out"`
	Log     string        `yaml:"log"`
	Level   string        `yaml:"level"`
	Timeout time.Duration `yaml:"timeout"`
	Workers int           `yaml:"workers"`
	Parse   bool          `yaml:"parse"`
	Chars   int           `yaml:"chars"`
}

type URLs map[string][]string

func Default() Config {
	dst := Config{}
	if defaultURLs.URLs != nil {
		dst.URLs = make(URLs, len(defaultURLs.URLs))
		for key, values := range defaultURLs.URLs {
			dst.URLs[key] = append([]string(nil), values...)
		}
	}

	return dst
}

func DefaultRuntime() Runtime {
	return defaultRuntime
}

func Save(filename string) error {
	return save(Default(), filename)
}

func SaveConfig(filename string, urlsConfig *Config) error {
	if urlsConfig == nil {
		return fmt.Errorf("urls config is nil")
	}

	return save(urlsConfig, filename)
}

func New(filename string) (*Config, error) {
	_config := Default()

	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &_config, ErrNotExists
		}
		return nil, fmt.Errorf("failed to find urls config: %w", err)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read urls config: %w", err)
	}

	if err := yaml.Unmarshal(data, &_config); err != nil {
		return nil, fmt.Errorf("failed to load urls config: %w", err)
	}

	return &_config, nil
}

func save(config any, path string) error {
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal urls config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to save urls config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to save urls config: %w", err)
	}

	return nil
}
