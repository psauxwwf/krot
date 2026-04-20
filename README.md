# krot

`krot` is a concurrent proxy checker.

It reads proxy URLs from input files, validates them, and saves only working entries.

Supported formats:

- MTProto links: `tg://proxy?...`, `https://t.me/proxy?...`, `https://www.t.me/proxy?...`
- Xray-compatible URI schemes: `vless://`, `vmess://`, `trojan://`, `ss://`

## What It Does

- MTProto checks use a real Telegram API call (`help.getNearestDc`)
- `vless/vmess/trojan/ss` checks run local Xray and probe connectivity via local SOCKS5
- Supports high concurrency with worker pool
- Skips empty lines and `#` comments
- Optional shuffle for input lines before checking
- Optional parse-only mode (URI parse/validate only, without network checks)
- Logs to `stdout` by default; optionally duplicates JSON logs to a file via `--log`

## Build

```bash
task build:linux
```

Termux/Android ARM64 build:

```bash
task build:termux
```

## Flags

All available CLI flags from `cmd/krot/main.go`:

- `--in` (default: empty) - input file (required for normal mode)
- `--out` (default: empty) - output file; if empty, auto-generated as `<dd.mm.yyyy_hh:mm>_<basename(in)>`
- `--log` (default: empty) - optional path for JSON log file (logs still go to `stdout`)
- `--level` (default: `info`) - log level: `debug|info|warn|error`
- `--timeout` (default: `6s`) - timeout for one proxy check (`10s`, `1m`, etc.)
- `--workers` (default: `runtime.NumCPU()*3`) - number of concurrent workers
- `--pipeline` (default: `false`) - run built-in pipeline checks for `mtproto.txt`, `vless.txt`, `vless_small.txt`
- `--shuf` (default: `true`) - shuffle input lines before processing
- `--parse` (default: `false`) - parse/validate only, without outbound requests
- `--chars` (default: `4096`) - max characters allowed in one input line
- `--load` (default: `false`) - download source lists into local files and then run parse validation on them

## Modes

`krot` currently works in three practical modes.

### 1) Normal mode (single file check)

Default mode when `--load=false` and `--pipeline=false`.

```bash
./bin/krot --in vless.txt --out ok.txt --workers 24 --timeout 8s
```

`--in` must be set in this mode.

If `--out` is not set, output filename is generated automatically.

### 2) Pipeline mode

Runs checks for predefined files in one run:

- `mtproto.txt`
- `vless.txt`
- `vless_small.txt`

```bash
./bin/krot --pipeline --workers 24 --timeout 8s
```

### 3) Load mode

Downloads and merges remote source lists into:

- `vless.txt`
- `vless_small.txt`
- `mtproto.txt`

Then runs parse-only validation on these files.

```bash
./bin/krot --load --workers 24
```

## Parse-Only Mode

`--parse` can be used in normal or pipeline mode to quickly validate URI syntax without real connectivity checks:

```bash
./bin/krot --in in.txt --parse
```

In parse-only flow, worker count is internally multiplied for faster parsing throughput.

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
- Logs are printed to `stdout` in text format
- If `--log <path>` is passed, JSON logs are also appended to that file

## Exit Codes

- `0` - success
- `1` - initialization error (flag validation, logger setup)
- `2` - runtime or unknown fatal error

## Disclaimer

Use responsibly and in compliance with local laws and service/provider policies.
