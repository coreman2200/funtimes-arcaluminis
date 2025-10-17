//go:build !linux

package led

import "fmt"

type SPI struct{}

func NewSPI(spiDev string, count int, colorOrder string, speedHz int, resetUs int) (*SPI, error) {
	return nil, fmt.Errorf("spi driver not supported on this platform")
}

func (s *SPI) Write(rgb []byte) error { return fmt.Errorf("spi driver not supported on this platform") }

func (s *SPI) Close() error { return nil }
