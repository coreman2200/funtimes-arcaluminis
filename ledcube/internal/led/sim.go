
package led

type Sim struct{}

func NewSim() *Sim { return &Sim{} }
func (s *Sim) Write(rgb []byte) error { return nil }
func (s *Sim) Close() error { return nil }
