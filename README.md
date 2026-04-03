# krot

`krot` is a concurrent proxy checker for two formats:

- Telegram MTProto proxy links: `tg://proxy...`, `https://t.me/proxy...`
- VLESS URIs: `vless://...`

The tool reads proxy URLs from a text file, checks them, and writes only working entries to the output file.

## Features

- Checks MTProto proxies through a real Telegram API request (`help.getNearestDc`)
- Checks VLESS proxies by starting a local Xray instance and probing HTTP connectivity through SOCKS
- Supports configurable timeout and worker concurrency
- Skips empty lines and `#` comments in the input file
- Writes text logs to stderr and JSON logs to `krot.json`

## Requirements

- Go `1.26.1+` for building from source
- Network access to the tested proxy endpoints
- For VLESS checks: access to probe URLs such as `https://cp.cloudflare.com/generate_204` and `https://www.gstatic.com/generate_204`

## Build

Or use the provided `Taskfile`:

```bash
task build:linux
```

There is also a Termux/Android ARM64 target:

```bash
task build:termux
```

## Usage

```bash
./bin/krot -in in.txt -out out.txt -timeout 30s -workers 8 -level info
```

## Command-Line Flags

- `-in` default `in.txt`: input file with proxy URLs
- `-out` default `out.txt`: output file for working proxies
- `-timeout` default `30s`: timeout per proxy check
- `-workers` default `runtime.NumCPU()`: number of concurrent workers
- `-level` default `info`: log level: `debug`, `info`, `warn`, `error`

Invalid `timeout` and `workers` values are rejected at startup.

## Input Format

- One proxy URL per line
- Empty lines are ignored
- Lines starting with `#` are ignored as comments

Example:

```text
# MTProto
tg://proxy?server=example.com&port=443&secret=abcdef1234
https://t.me/proxy?server=example.com&port=443&secret=abcdef1234

# VLESS
vless://uuid@example.com:443?encryption=none&type=tcp&security=tls&sni=example.com
vless://uuid@example.com:443?encryption=none&type=grpc&security=tls&serviceName=my-service&sni=example.com
```

## Output

- The output file contains only proxies that passed the check
- Entries are written as checks finish, so output order is not guaranteed to match input order
- `krot.json` is append-only JSON logging in the current working directory

## Disclaimer

Use the project responsibly and in compliance with local laws, provider terms, and platform rules.
