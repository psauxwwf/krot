# krot

`krot` is a concurrent proxy checker.

It reads proxy URLs from input files, validates them, and saves only working entries.

Supported formats:

- MTProto links: `tg://proxy?...`, `https://t.me/proxy?...`, `https://www.t.me/proxy?...`
- Xray-compatible URI schemes: `vless://`, `vmess://`, `trojan://`, `ss://`

## Sub

```ini
https://github.com/psauxwwf/krot/releases/latest/download/vless_small.txt
```

```ini
https://github.com/psauxwwf/krot/releases/latest/download/vless.txt
```

## What It Does

- MTProto checks use a real Telegram API call (`help.getNearestDc`)
- `vless/vmess/trojan/ss` checks run local Xray and probe connectivity via local SOCKS5
- Supports high concurrency with worker pool
- Skips empty lines and `#` comments
- Always shuffles input lines before checking
- Optional parse-only mode (URI parse/validate only, without network checks)
- Logs are written to `krot.json` by default; logger output does not go to terminal

## Build

```bash
task build:linux
```

Termux/Android ARM64 build:

```bash
task build:termux
```

## Commands

Available commands:

- `krot` - check proxies from one input file
- `krot parse` - parse/validate proxies from one input file without network checks
- `krot load` - download source lists from `urls.yaml` and then parse/validate them
- `krot pipeline` - run built-in checks for `mtproto.txt`, `vless.txt`, `vless_small.txt`
- `krot save` - save default URL lists to `urls.yaml`

Common flags:

- `--urls` (default: `urls.yaml`) - path to YAML file with source URL lists
- `--in` (default: empty) - input file
- `--out` (default: empty) - output file; if empty, auto-generated as `<dd.mm.yyyy_hh:mm>_<basename(in)>`
- `--log-path` (default: `krot.json`) - path to JSON log file
- `--log-level` (default: `info`) - log level: `debug|info|warn|error`
- `--timeout` (default: `6s`) - timeout for one proxy check (`10s`, `1m`, etc.)
- `--workers` (default: `runtime.NumCPU()*3`) - number of concurrent workers
- `--chars` (default: `4096`) - max characters allowed in one input line

## Modes

`krot` currently works in four practical modes.

### 1) Normal mode (single file check)

Default mode for checking one file with real connectivity tests.

```bash
./bin/krot --in vless.txt --out ok.txt --workers 24 --timeout 8s
```

`--in` must be set in this mode.

If `--out` is not set, output filename is generated automatically.

### 2) Parse-only mode

Validates URI syntax from one file without real connectivity checks:

```bash
./bin/krot parse --in in.txt
```

In parse-only flow, worker count is internally multiplied for faster parsing throughput.

### 3) Pipeline mode

Runs checks for predefined files in one run:

- `mtproto.txt`
- `vless.txt`
- `vless_small.txt`

```bash
./bin/krot pipeline --workers 24 --timeout 8s
```

### 4) Load mode

Downloads and merges remote source lists from `urls.yaml` (or the file passed via `--urls`) into:

- `vless.txt`
- `vless_small.txt`
- `mtproto.txt`

Then runs parse-only validation on these files.

```bash
./bin/krot load --workers 24
```

Generate default `urls.yaml`:

```bash
./bin/krot save
```

## Input Rules

- One proxy URI per line
- Empty lines are ignored
- Lines starting with `#` are ignored
- Lines longer than `--chars` are skipped

Example:

```text
# MTProto
tg://proxy?server=example.com&port=443&secret=abcdef1234
https://t.me/proxy?server=example.com&port=443&secret=abcdef1234

# Xray-compatible URIs
vless://uuid@example.com:443?encryption=none&type=tcp&security=tls&sni=example.com
vmess://...
trojan://...
ss://...
```

## Output and Logs

- Output file contains only successful entries
- Order is not guaranteed (concurrent processing)
- Progress is printed to `stderr`
- Logger output is written to `krot.json` in JSON format by default
- `--log-path <path>` overrides the default log file path

## Exit Codes

- `0` - success
- `1` - initialization error (flag validation, logger setup)
- `2` - runtime or unknown fatal error

## Disclaimer

Use responsibly and in compliance with local laws and service/provider policies.

---

## Links

- [WG Tunnel](https://github.com/wgtunnel/android/releases/download/4.3.1/wgtunnel-standalone-v4.3.1.apk)
- [Happ](https://github.com/Happ-proxy/happ-android/releases/download/3.20.4/Happ.apk)
- [Exclave](https://github.com/dyhkwong/Exclave/releases/download/0.17.39/Exclave-0.17.39-arm64-v8a.apk)
- [ByeByeDPI](https://github.com/romanvht/ByeByeDPI/releases/download/v.1.7.5/ByeByeDPI-v1.7.5-arm64-v8a-release.apk)
- [Dnstt.xyz](https://github.com/dnstt-xyz/dnstt_xyz_app/releases/download/v2.2.0/DNSTT-Client-v2.2.0-Android-arm64-v8a.apk)
- [Karing](https://github.com/KaringX/karing/releases/download/v1.2.19.2202/karing_1.2.19.2202_android_arm64-v8a.apk)
- [Olcbox](https://github.com/alananisimov/olcbox/releases/download/v0.1.0-alpha/Olcbox-android-release.apk)
- [Olcng](https://github.com/openlibrecommunity/olcng/releases/download/v4.1.1/app-fdroid-arm64-v8a-release.apk)
- [Incy](https://github.com/INCY-DEV/incy-platforms/releases/download/desktop-v3.0.25/Incy.apk)
- [OpenConnect](https://f-droid.org/packages/net.openconnect_vpn.android)
- [V2BOX](https://play.google.com/store/apps/details?id=dev.hexasoftware.v2box)
