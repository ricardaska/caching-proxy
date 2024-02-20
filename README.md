# Caching Proxy

Basic caching HTTP proxy for APIs, created to cache Mojang API responses for multiple clients (game servers).

Responses are cached on route level.

## Build

Project is normally built using `make build` command which creates statically-linked binary, it requires `musl-gcc` installed.

## Example config.toml

```toml
log_level = "info" # Options: debug, info, warn, error

## Control server

[control_server]
enabled = false
network = "unix" # Options: unix, tcp, tcp4, tcp6
bind = "/tmp/proxy_ctrl.sock"

## HTTP request forwarding client config

[http_client]
timeout_tcp = "3s"
timeout_tls = "5s"
timeout_headers = "5s"

max_idle_conns = 3
idle_timeout = "10s"

## HTTP server and route config

# API endpoint

[[servers]]
bind = "localhost:4001"

[[servers.routes]]
target = "https://api.mojang.com"
path = "/users/profiles/minecraft/"
keep_headers = ['Content-Type'] # Omit keep_headers to keep everything, or use drop_headers to drop specific ones
time_to_live = "15m"

# Session server endpoint

[[servers]]
bind = "localhost:4002"

[[servers.routes]]
target = "https://sessionserver.mojang.com"
path = "/session/minecraft/profile/"
keep_headers = ['Content-Type']
time_to_live = "15m"

```

## Control server

### Commands

Drop cached values:

-   `drop <url path>`
-   `drop_prefix <url path>`

Adjust log level:

-   `log_level [debug|info|warn|error]`

### Usage example with netcat

```bash
echo 'drop /users/profiles/minecraft/User' | nc -w1 -U /tmp/proxy_ctrl.sock
```

# Libraries

-   TOML file format [go-toml](https://github.com/pelletier/go-toml)
