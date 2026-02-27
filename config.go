package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type AudioConfig struct {
	SampleRate int `yaml:"sample_rate"`
	BufferSize int `yaml:"buffer_size"`
}

type VisualConfig struct {
	FPS           int     `yaml:"fps"`
	BarWidth      int     `yaml:"bar_width"`
	BarGap        int     `yaml:"bar_gap"`
	Smoothing     float64 `yaml:"smoothing"`
	Sensitivity   float64 `yaml:"sensitivity"`
	PeakFallSpeed float64 `yaml:"peak_fall_speed"`
	ShowPeaks     bool    `yaml:"show_peaks"`
	Mirror        bool    `yaml:"mirror"`
	ShowStatus    bool    `yaml:"show_status"`
}

type Config struct {
	Style       string       `yaml:"style"`
	ColorScheme string       `yaml:"color_scheme"`
	Audio       AudioConfig  `yaml:"audio"`
	Visual      VisualConfig `yaml:"visual"`
	DemoMode    bool         `yaml:"-"`
}

func DefaultConfig() *Config {
	return &Config{
		Style:       "bars",
		ColorScheme: "rainbow",
		Audio: AudioConfig{
			SampleRate: 44100,
			BufferSize: 4096,
		},
		Visual: VisualConfig{
			FPS:           60,
			BarWidth:      2,
			BarGap:        1,
			Smoothing:     0.65,
			Sensitivity:   1.0,
			PeakFallSpeed: 0.03,
			ShowPeaks:     true,
			Mirror:        false,
			ShowStatus:    true,
		},
	}
}

func (c *Config) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, c)
}

func (c *Config) TryLoadDefault() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	paths := []string{
		filepath.Join(home, ".config", "audiovis", "config.yaml"),
		filepath.Join(home, ".config", "audiovis", "config.yml"),
		filepath.Join(home, ".audiovis.yaml"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			_ = c.LoadFromFile(p)
			return
		}
	}
}
