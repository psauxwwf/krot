package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

var (
	_default = Config{
		Runtime: Runtime{
			In:       "",
			Out:      "",
			Log:      "",
			Level:    "info",
			Timeout:  6 * time.Second,
			Workers:  runtime.NumCPU() * 3,
			Pipeline: false,
			Shuf:     true,
			Parse:    false,
			Chars:    4096,
			Load:     false,
		},
		Urls: Urls{
			"vless": []string{
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
			},
		},
	}
	ErrNotExists = fmt.Errorf("config not found: %w", os.ErrNotExist)
)

type Config struct {
	Runtime Runtime `yaml:"runtime"`
	Urls    Urls    `yaml:"urls"`
}

type Runtime struct {
	In       string        `yaml:"in"`
	Out      string        `yaml:"out"`
	Log      string        `yaml:"log"`
	Level    string        `yaml:"level"`
	Timeout  time.Duration `yaml:"timeout"`
	Workers  int           `yaml:"workers"`
	Pipeline bool          `yaml:"pipeline"`
	Shuf     bool          `yaml:"shuf"`
	Parse    bool          `yaml:"parse"`
	Chars    int           `yaml:"chars"`
	Load     bool          `yaml:"load"`
}

type Urls map[string][]string

func Default() Config {
	dst := _default
	if _default.Urls != nil {
		dst.Urls = make(Urls, len(_default.Urls))
		for key, values := range _default.Urls {
			dst.Urls[key] = append([]string(nil), values...)
		}
	}

	return dst
}

func Save(filename string) error {
	return save(Default(), filename)
}

func New(filename string) (*Config, error) {
	_config := Default()

	if _, err := os.Stat(filename); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &_config, ErrNotExists
		}
		return nil, fmt.Errorf("failed to find file: %w", err)
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	if err := yaml.Unmarshal(data, &_config); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	return &_config, nil
}

func FromCobra(path string, cmd *cobra.Command) (*Config, error) {
	if _, err := os.Stat(path); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to stat config %q: %w", path, err)
		}
		if err := Save(path); err != nil {
			return nil, fmt.Errorf("failed to create default config %q: %w", path, err)
		}
	}

	v := viper.New()
	v.SetConfigFile(path)

	binding := map[string]string{
		"runtime.in":       "in",
		"runtime.out":      "out",
		"runtime.log":      "log",
		"runtime.level":    "level",
		"runtime.timeout":  "timeout",
		"runtime.workers":  "workers",
		"runtime.pipeline": "pipeline",
		"runtime.shuf":     "shuf",
		"runtime.parse":    "parse",
		"runtime.chars":    "chars",
		"runtime.load":     "load",
	}
	for key, flag := range binding {
		if err := v.BindPFlag(key, cmd.Flags().Lookup(flag)); err != nil {
			return nil, fmt.Errorf("failed to bind flag %q: %w", flag, err)
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config %q: %w", path, err)
	}

	_config := Default()
	if err := v.Unmarshal(&_config); err != nil {
		return nil, fmt.Errorf("failed to parse config %q: %w", path, err)
	}

	return &_config, nil
}

func save(config any, path string) error {
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to save default config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("failed to save default config: %w", err)
	}

	return nil
}
