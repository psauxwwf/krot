# krot

`krot` is a fast, concurrent proxy checker for two formats:

- Telegram MTProto proxy links (`tg://proxy...` and `https://t.me/proxy...`)
- VLESS URIs (`vless://...`)

It reads proxy URLs from a text file, verifies connectivity, and writes only working entries to an output file.

## Features

- Validates MTProto proxy links by performing a real Telegram API reachability check
- Validates VLESS links for `tcp` and `ws` transport
- Supports VLESS security modes: `none`, `tls`, and `reality` (for TCP)
- Handles large lists using configurable worker-based concurrency
- Supports configurable timeout per proxy check
- Skips blank lines and `#` comments in input files
- Writes structured logs to `krot.json` and human-readable logs to stderr

## Requirements

- Go `1.26+` (if you run from source)
- Outbound network access to tested proxy endpoints

## Usage

Run `krot` and provide input/output files:

```bash
./bin/krot -in in.txt -out out.txt -timeout 30s -workers 8 -level info
```

## Command-line options

- `-in` (default: `in.txt`): input file with proxy URLs
- `-out` (default: `out.txt`): output file for working proxies
- `-timeout` (default: `30s`): timeout per proxy check (for example: `10s`, `45s`, `1m`)
- `-workers` (default: number of CPU cores): number of concurrent workers
- `-level` (default: `info`): log level (`debug`, `info`, `warn`, `error`)

## Input format

- One proxy URL per line
- Empty lines are ignored
- Lines starting with `#` are treated as comments

Example:

```text
# MTProto
tg://proxy?server=example.com&port=443&secret=abcdef1234
https://t.me/proxy?server=example.com&port=443&secret=abcdef1234

# VLESS
vless://uuid@example.com:443?encryption=none&type=tcp&security=tls&sni=example.com
```

## Supported formats

### MTProto

- `tg://proxy?server=...&port=...&secret=...`
- `https://t.me/proxy?server=...&port=...&secret=...`

`secret` can be provided in hex or base64/base64url form.

### VLESS

- Scheme: `vless://`
- Required: UUID, host, port
- Supported transports: `type=tcp`, `type=ws`
- Supported security:
  - `security=none`
  - `security=tls`
  - `security=reality` (TCP only, requires `pbk`; optional `sid`, `fp`)
- Supported flow: `flow=xtls-rprx-vision`

## Output

- `out.txt` (or your `-out` path) contains only proxies that passed checks
- `krot.json` contains structured JSON logs for debugging/auditing

## Public list sources (examples)

- MTProto lists: <https://raw.githubusercontent.com/SoliSpirit/mtproto/master/all_proxies.txt>
- VLESS lists: <https://raw.githubusercontent.com/zieng2/wl/main/vless_universal.txt>

## Disclaimer

Use this project responsibly and in compliance with local laws, provider terms, and platform rules.
