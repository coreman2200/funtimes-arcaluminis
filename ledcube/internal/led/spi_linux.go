//go:build linux

package led

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

/*
Minimal spidev ioctl bindings (no external deps).
If you prefer, swap this file to use periph.io/x/conn/spi for cleaner setup.
*/

const (
	spiIOCWriteMode        = 0x40016b01
	spiIOCWriteBitsPerWord = 0x40016b03
	spiIOCWriteMaxSpeedHz  = 0x40046b04
)

type SPI struct {
	mu       sync.Mutex
	f        *os.File
	count    int
	colorOrd [3]byte
	resetUs  int
	// Precomputed LUT: byte -> 24-bit encoded (3 bytes) using 0b100 (0) / 0b110 (1)
	lut [256][3]byte
}

// NewSPI opens spidev (e.g. "/dev/spidev0.0") and prepares an encoder for WS2812-over-SPI.
// speedHz in the 2_400_000–3_200_000 range works well with this 3x expand scheme.
// colorOrder like "GRB" or "RGB". resetUs is the latch (usually >= 280µs; 300–400 is safe).
func NewSPI(spiDev string, count int, colorOrder string, speedHz int, resetUs int) (*SPI, error) {
	if count <= 0 {
		return nil, fmt.Errorf("invalid LED count: %d", count)
	}
	if speedHz <= 0 {
		speedHz = 2400000
	}
	if resetUs <= 0 {
		resetUs = 300
	}
	f, err := os.OpenFile(spiDev, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("open spidev: %w", err)
	}
	// mode 0
	mode := byte(0)
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), spiIOCWriteMode, uintptr(unsafe.Pointer(&mode))); e != 0 {
		_ = f.Close()
		return nil, fmt.Errorf("SPI set mode: %v", e)
	}
	// 8 bits per word
	bpw := byte(8)
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), spiIOCWriteBitsPerWord, uintptr(unsafe.Pointer(&bpw))); e != 0 {
		_ = f.Close()
		return nil, fmt.Errorf("SPI set bits-per-word: %v", e)
	}
	// max speed
	if _, _, e := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), spiIOCWriteMaxSpeedHz, uintptr(unsafe.Pointer(&speedHz))); e != 0 {
		_ = f.Close()
		return nil, fmt.Errorf("SPI set speed: %v", e)
	}

	s := &SPI{
		f:        f,
		count:    count,
		resetUs:  resetUs,
		colorOrd: [3]byte{byte('G'), byte('R'), byte('B')}, // default GRB
	}
	if len(colorOrder) == 3 {
		s.colorOrd = [3]byte{colorOrder[0], colorOrder[1], colorOrder[2]}
	}

	// Build LUT: for each input byte, expand each bit MSB->LSB to 3 SPI bits:
	// bit=1 -> '110' (high longer), bit=0 -> '100' (high shorter).
	// Pack 8*(3) = 24 bits into 3 bytes.
	for v := 0; v < 256; v++ {
		var b0, b1, b2 byte
		out := uint32(0)
		for i := 7; i >= 0; i-- {
			bit := (v >> i) & 1
			var tri uint32
			if bit == 1 {
				tri = 0b110
			} else {
				tri = 0b100
			}
			out = (out << 3) | tri
		}
		b0 = byte((out >> 16) & 0xFF)
		b1 = byte((out >> 8) & 0xFF)
		b2 = byte(out & 0xFF)
		s.lut[v][0], s.lut[v][1], s.lut[v][2] = b0, b1, b2
	}

	return s, nil
}

func (s *SPI) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.f != nil {
		err := s.f.Close()
		s.f = nil
		return err
	}
	return nil
}

func (s *SPI) encodePixel(r, g, b byte, dst []byte) {
	order := s.colorOrd
	var v [3]byte
	for i := 0; i < 3; i++ {
		switch order[i] {
		case 'R':
			v[i] = r
		case 'G':
			v[i] = g
		case 'B':
			v[i] = b
		default:
			v[i] = g // fallback
		}
	}
	off := 0
	for i := 0; i < 3; i++ {
		dst[off+0] = s.lut[v[i]][0]
		dst[off+1] = s.lut[v[i]][1]
		dst[off+2] = s.lut[v[i]][2]
		off += 3
	}
}

// Write takes len(rgb)==3*count. We expand to 9 bytes/pixel (3 per color) plus reset tail.
func (s *SPI) Write(rgb []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.f == nil {
		return fmt.Errorf("SPI closed")
	}
	if len(rgb) != s.count*3 {
		return fmt.Errorf("rgb length %d does not match count %d", len(rgb), s.count)
	}

	// Frame buffer: 9 bytes per pixel (24 encoded bits per byte => 3 bytes per color)
	enc := make([]byte, s.count*9)

	for i := 0; i < s.count; i++ {
		r := rgb[i*3+0]
		g := rgb[i*3+1]
		b := rgb[i*3+2]
		s.encodePixel(r, g, b, enc[i*9:i*9+9])
	}

	// Write encoded stream
	if _, err := s.f.Write(enc); err != nil {
		return fmt.Errorf("spi write: %w", err)
	}

	// Reset (latch): drive low for resetUs. With SPI we approximate by sending zeros.
	// At 2.4MHz, 1 byte ~ 3.33us. For 300us, send ~90 bytes. Round up to be safe.
	resetBytes := (s.resetUs + 3) / 3 * 1 // rough; we’ll just send 128 zeros.
	if resetBytes < 128 {
		resetBytes = 128
	}
	zeros := make([]byte, resetBytes)
	if _, err := s.f.Write(zeros); err != nil {
		return fmt.Errorf("spi latch: %w", err)
	}
	return nil
}
