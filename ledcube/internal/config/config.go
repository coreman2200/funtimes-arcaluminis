package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type PowerCfg struct {
	LimitAmps   float64 `yaml:"limit_amps"`
	WhiteCap    float64 `yaml:"white_cap"`
	SoftStartMs int     `yaml:"soft_start_ms"`
}

type Dim struct {
	X int `yaml:"x"`
	Y int `yaml:"y"`
	Z int `yaml:"z"`
}

type SPI struct {
	Dev     string `yaml:"dev"`      // e.g. /dev/spidev0.0
	SpeedHz int    `yaml:"speed_hz"` // e.g. 2400000
	ResetUs int    `yaml:"reset_us"` // e.g. 300
}

type Config struct {
	Driver     string  `yaml:"driver"` // "pwm" | "sim"
	GPIO       int     `yaml:"gpio"`
	ColorOrder string  `yaml:"color_order"`
	Brightness float64 `yaml:"brightness"`
	FPS        int     `yaml:"fps"`

	Dim             Dim     `yaml:"dim"`
	PitchMM         float64 `yaml:"pitch_mm"`
	PanelGapMM      float64 `yaml:"panel_gap_mm"`
	XFlipEveryRow   bool    `yaml:"x_flip_every_row"`
	YFlipEveryPanel bool    `yaml:"y_flip_every_panel"`

	Power PowerCfg `yaml:"power"`
	SPI   SPI      `yaml:"spi,omitempty"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func Save(path string, c *Config) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
