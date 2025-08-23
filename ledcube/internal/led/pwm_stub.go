//go:build !linux

package led

import "fmt"

type PWM struct{}

func NewPWM(gpio int, count int, colorOrder string, brightness float64) (*PWM, error) {
	return nil, fmt.Errorf("pwm driver not supported on this platform")
}
func (p *PWM) Write(rgb []byte) error { return nil }
func (p *PWM) Close() error { return nil }
