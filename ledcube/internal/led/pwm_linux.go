//go:build linux

package led

/*
#cgo LDFLAGS: -lws2811
#include <stdlib.h>
#include <stdint.h>
#include <ws2811/ws2811.h>
*/
import "C"
import (
	"fmt"
	"sync"
	"unsafe"
)

type PWM struct{
	gpio int
	count int
	colorOrder string
	brightness float64

	mu sync.Mutex
	dev *C.ws2811_t
	buf unsafe.Pointer
}

func NewPWM(gpio int, count int, colorOrder string, brightness float64) (*PWM, error) {
	p := &PWM{gpio: gpio, count: count, colorOrder: colorOrder, brightness: brightness}

	// Allocate ws2811_t
	p.dev = (*C.ws2811_t)(C.calloc(1, C.size_t(unsafe.Sizeof(*p.dev))))
	if p.dev == nil { return nil, fmt.Errorf("calloc ws2811_t failed") }

	// Initialize structure with common defaults
	p.dev.freq = 800000
	p.dev.dmanum = 10
	// Channel 0
	ch := &p.dev.channel[0]
	ch.gpionum = C.int(gpio)
	ch.count = C.int(count)
	ch.invert = 0
	// Map color order to strip type (basic set)
	switch colorOrder {
	case "RGB": ch.strip_type = C.WS2811_STRIP_RGB
	case "BRG": ch.strip_type = C.WS2811_STRIP_BRG
	case "GRB": fallthrough
	default:    ch.strip_type = C.WS2811_STRIP_GRB
	}
	ch.brightness = C.uint8_t(int(brightness*255) & 0xFF)

	// Init device
	if st := C.ws2811_init(p.dev); st != C.WS2811_SUCCESS {
		C.free(unsafe.Pointer(p.dev))
		return nil, fmt.Errorf("ws2811_init failed: %d", int(st))
	}

	// Grab pointer to LED buffer
	p.buf = unsafe.Pointer(ch.leds)
	return p, nil
}

func (p *PWM) Write(rgb []byte) error {
	p.mu.Lock(); defer p.mu.Unlock()
	if p.dev == nil { return fmt.Errorf("pwm not initialized") }
	// Map RGB to 32-bit GRB packed format expected by ws2811 (0x00RRGGBB order varies by strip_type)
	// We'll pack as 0x00RRGGBB and rely on strip_type for channel ordering.
	leds := (*[1 << 26]C.ws2811_led_t)(p.buf)[:p.count:p.count]
	for i := 0; i < p.count && i*3+2 < len(rgb); i++ {
		r := uint32(rgb[i*3+0])
		g := uint32(rgb[i*3+1])
		b := uint32(rgb[i*3+2])
		val := (r << 16) | (g << 8) | b
		leds[i] = C.ws2811_led_t(val)
	}
	if st := C.ws2811_render(p.dev); st != C.WS2811_SUCCESS {
		return fmt.Errorf("ws2811_render failed: %d", int(st))
	}
	return nil
}

func (p *PWM) Close() error {
	p.mu.Lock(); defer p.mu.Unlock()
	if p.dev != nil {
		C.ws2811_fini(p.dev)
		C.free(unsafe.Pointer(p.dev))
		p.dev = nil
	}
	return nil
}
