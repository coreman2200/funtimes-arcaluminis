
package led

// Driver abstracts an LED output sink.
type Driver interface {
	// Write pushes an RGB frame to hardware. len(rgb) must be 3*N.
	Write(rgb []byte) error
	// Close releases resources.
	Close() error
}
