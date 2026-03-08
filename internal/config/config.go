package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Interval       time.Duration `yaml:"interval"`
	RunOnce        bool          `yaml:"run_once"`
	StateFile      string        `yaml:"state_file"`
	GoBGPBin       string        `yaml:"gobgp_bin"`
	BlackholeComm  string        `yaml:"blackhole_community"`
	Sources        []Source      `yaml:"sources"`
	AllowlistCIDRs []string      `yaml:"allowlist_cidrs"`
	MinPrefixV4    int           `yaml:"min_prefix_v4"`
	MinPrefixV6    int           `yaml:"min_prefix_v6"`
}

type Source struct {
	Name string `yaml:"name"`
	URL  string `yaml:"url"`
}

func Load(path string) (Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{
		Interval:      30 * time.Minute,
		StateFile:     "/var/lib/nullroute/state.txt",
		GoBGPBin:      "gobgp",
		BlackholeComm: "65535:666",
		MinPrefixV4:   24,
		MinPrefixV6:   48,
	}

	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.Interval <= 0 {
		return Config{}, fmt.Errorf("interval must be > 0")
	}
	if len(cfg.Sources) == 0 {
		return Config{}, fmt.Errorf("at least one source is required")
	}
	for _, s := range cfg.Sources {
		if s.URL == "" {
			return Config{}, fmt.Errorf("source url cannot be empty")
		}
	}
	if cfg.MinPrefixV4 < 8 || cfg.MinPrefixV4 > 32 {
		return Config{}, fmt.Errorf("min_prefix_v4 out of range")
	}
	if cfg.MinPrefixV6 < 16 || cfg.MinPrefixV6 > 128 {
		return Config{}, fmt.Errorf("min_prefix_v6 out of range")
	}

	return cfg, nil
}
