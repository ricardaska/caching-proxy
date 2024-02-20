package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Duration time.Duration

func (d *Duration) UnmarshalText(text []byte) error {
	duration, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	*d = Duration(duration)
	return nil
}

func (d *Duration) Duration() time.Duration {
	return time.Duration(*d)
}

type HTTPClientConfig struct {
	TCPTimeout    Duration `toml:"timeout_tcp"`
	TLSTimeout    Duration `toml:"timeout_tls"`
	HeaderTimeout Duration `toml:"timeout_headers"`
	IdleTimeout   Duration `toml:"idle_timeout"`
	MaxIdleConns  int      `toml:"max_idle_conns"`
}

type ServerConfig struct {
	BindAddr string         `toml:"bind"`
	Routes   []*RouteConfig `toml:"routes"`
}

type ControlServerConfig struct {
	Enabled  bool   `toml:"enabled"`
	Network  string `toml:"network"`
	BindAddr string `toml:"bind"`
}

type RouteConfig struct {
	Target      string   `toml:"target"`
	Path        string   `toml:"path"`
	KeepHeaders []string `toml:"keep_headers"`
	DropHeaders []string `toml:"drop_headers"`
	TTL         Duration `toml:"time_to_live"`
}

type Config struct {
	LogLevel      slog.Level           `toml:"log_level"`
	ControlServer *ControlServerConfig `toml:"control_server"`
	HTTPClient    *HTTPClientConfig    `toml:"http_client"`
	Servers       []*ServerConfig      `toml:"servers"`
}

func ReadConfig(path string) (*Config, error) {
	var config Config

	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := toml.Unmarshal(contents, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
