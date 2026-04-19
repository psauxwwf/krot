package loader

import (
	"bufio"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"
)

const loadTimeout = 2 * time.Minute

var (
	Vless = []string{
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
	}

	VlessSmall = []string{
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
	}

	Mtproto = []string{
		"https://raw.githubusercontent.com/SoliSpirit/mtproto/master/all_proxies.txt",
	}
)

func Load(urls ...string) ([]string, error) {
	if len(urls) == 0 {
		urls = VlessSmall
	}
	slog.Info("starting urls load", "sources", len(urls))

	client := &http.Client{Timeout: loadTimeout}
	result := make([]string, 0)

	for _, sourceURL := range urls {
		slog.Debug("loading source", "url", sourceURL)
		resp, err := client.Get(sourceURL)
		if err != nil {
			slog.Error("failed to load source", "url", sourceURL, "error", err)
			return result, fmt.Errorf("failed to load %s: %w", sourceURL, err)
		}

		err = func() error {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to load %s: status %s", sourceURL, resp.Status)
			}

			scanner := bufio.NewScanner(resp.Body)
			scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}

				if slices.Contains(result, line) {
					continue
				}

				result = append(result, line)
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("failed to read %s: %w", sourceURL, err)
			}

			return nil
		}()
		if err != nil {
			slog.Error("failed to parse source", "url", sourceURL, "error", err)
			return result, err
		}
		fmt.Fprintf(os.Stderr, "loaded url: %s\n", sourceURL)
		slog.Info("source loaded", "url", sourceURL, "unique_total", len(result))
	}
	slog.Info("finished urls load", "unique_total", len(result))

	return result, nil
}

func SaveVless(out string) error {
	slog.Info("saving vless", "out", out)
	return save(out, Vless)
}

func SaveVlessSmall(out string) error {
	slog.Info("saving vless_small", "out", out)
	return save(out, VlessSmall)
}

func SaveMtproto(out string, urls []string) error {
	slog.Info("saving mtproto", "out", out)
	return save(out, Mtproto)
}

func save(out string, urls []string) error {
	if out == "" {
		return fmt.Errorf("output file is empty")
	}

	lines, err := Load(urls...)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open output file %s: %w", out, err)
	}
	defer f.Close()

	for _, line := range lines {
		if _, err := f.WriteString(line + "\n"); err != nil {
			slog.Error("failed to write line", "out", out, "error", err)
			return fmt.Errorf("failed to write output file %s: %w", out, err)
		}
	}
	slog.Info("save completed", "out", out, "lines", len(lines))

	return nil
}
