package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// ProviderConfig is a generic, untyped bag so new providers need no core changes.
type ProviderConfig struct {
	Name    string                 `mapstructure:"name"`
	Type    string                 `mapstructure:"type"` // e.g. "torrentio", "jackett"
	Enabled bool                   `mapstructure:"enabled"`
	Options map[string]interface{} `mapstructure:"options"` // endpoint, api_key, etc.
}

// ResolverConfig controls how magnet/torrent links become playable HTTP URLs.
type ResolverConfig struct {
	Type    string   `mapstructure:"type"`    // "peerflix" (default) or "none"
	Bin     string   `mapstructure:"bin"`     // bridge binary; defaults to "peerflix"
	Args    []string `mapstructure:"args"`    // extra CLI args for the bridge
	Timeout int      `mapstructure:"timeout"` // seconds to wait for the bridge server (default 90)
}

type Config struct {
	Player     string           `mapstructure:"player"`
	PlayerArgs []string         `mapstructure:"player_args"` // extra flags passed to the player
	Providers  []ProviderConfig `mapstructure:"providers"`
	Resolver   ResolverConfig   `mapstructure:"resolver"`
}

func Load() (*Config, error) {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".config", "cine")

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(dir)
	v.SetDefault("player", "mpv")
	v.SetDefault("resolver.type", "peerflix")
	v.SetDefault("resolver.timeout", 90)
	v.SetEnvPrefix("CINE")
	v.AutomaticEnv() // env vars like CINE_PLAYER override the file

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &c, nil
}
