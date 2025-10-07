package ws

import (
	"encoding/json"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/config"
	diag "github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/diagnostics"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/layout"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/led"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/tests"
)

type State struct {
	mu         sync.RWMutex
	Layout     layout.Layout
	FPS        int
	Brightness float64
	SimOnly    bool

	ConfigPath string
	Driver     led.Driver

	rgb         []byte
	frameID     uint64
	startTime   time.Time
	clients     map[*websocket.Conn]bool
	diagClients map[*websocket.Conn]bool

	testRunner    *tests.Runner
	CurrentDriver string
}

func NewState(l layout.Layout, fps int, brightness float64, simOnly bool) *State {
	return &State{
		Layout:      l,
		FPS:         fps,
		Brightness:  brightness,
		SimOnly:     simOnly,
		rgb:         make([]byte, l.Count()*3),
		startTime:   time.Now(),
		clients:     map[*websocket.Conn]bool{},
		diagClients: map[*websocket.Conn]bool{},
	}
}

func (s *State) RunRenderLoop() {
	ticker := time.NewTicker(time.Second / time.Duration(max(1, s.FPS)))
	defer ticker.Stop()
	phase := 0.0
	for range ticker.C {
		s.mu.Lock()
		n := s.Layout.Count()

		if s.testRunner != nil {
			done := !s.testRunner.Step(s.Layout, s.rgb)
			if done {
				s.testRunner = nil
				s.pushDiag(diag.Diagnostic{Severity: diag.Info, Code: "TEST.DONE", Summary: "Test complete"})
			} else {
				// fallthrough to frame send
			}
		} else {
			// Demo effect: rotating rainbow
			for i := 0; i < n; i++ {
				perPanel := s.Layout.Dim.X * s.Layout.Dim.Y
				z := i / perPanel
				rem := i % perPanel
				y := rem / s.Layout.Dim.X
				x := rem % s.Layout.Dim.X
				u := float64(x) / float64(max(1, s.Layout.Dim.X-1))
				v := float64(y) / float64(max(1, s.Layout.Dim.Y-1))
				w := float64(z) / float64(max(1, s.Layout.Dim.Z-1))
				h := math.Mod(u+v+w+phase, 1.0)
				r, g, b := hsvToRGB(h, 1.0, s.Brightness)
				s.rgb[i*3+0] = byte(r * 255)
				s.rgb[i*3+1] = byte(g * 255)
				s.rgb[i*3+2] = byte(b * 255)
			}
			phase += 0.01
		}

		// Apply simple white-cap limiter before sending
		applyWhiteCap(s.rgb, 0.85)

		s.frameID++
		buf := append([]byte{}, s.rgb...)
		drv := s.Driver
		s.mu.Unlock()

		// Write to hardware if driver present
		if drv != nil {
			_ = drv.Write(buf)
		}
		s.broadcastFrame(buf)
	}
}

func (s *State) HandleFramesWS(w http.ResponseWriter, r *http.Request) {
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.clients[conn] = true
	s.mu.Unlock()
	s.sendTopology(conn)

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.clients, conn)
			s.mu.Unlock()
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (s *State) HandleDiagWS(w http.ResponseWriter, r *http.Request) {
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	s.mu.Lock()
	s.diagClients[conn] = true
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.diagClients, conn)
			s.mu.Unlock()
			conn.Close()
		}()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (s *State) HandleControlWS(w http.ResponseWriter, r *http.Request) {
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := up.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg map[string]any
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		s.applyControl(msg)
		s.sendTopology(conn)
	}
}

func (s *State) HandleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resp := map[string]any{
		"frame_id":   s.frameID,
		"uptime_s":   time.Since(s.startTime).Seconds(),
		"count":      s.Layout.Count(),
		"fps":        s.FPS,
		"brightness": s.Brightness,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *State) applyControl(msg map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := msg["dim"].(map[string]any); ok {
		if x, ok2 := v["x"].(float64); ok2 {
			s.Layout.Dim.X = int(x)
		}
		if y, ok2 := v["y"].(float64); ok2 {
			s.Layout.Dim.Y = int(y)
		}
		if z, ok2 := v["z"].(float64); ok2 {
			s.Layout.Dim.Z = int(z)
		}
		s.rgb = make([]byte, s.Layout.Count()*3)
	}
	if v, ok := msg["panelGapMM"].(float64); ok {
		s.Layout.PanelGapMM = v
	}
	if v, ok := msg["pitchMM"].(float64); ok {
		s.Layout.PitchMM = v
	}
	if v, ok := msg["fps"].(float64); ok {
		s.FPS = int(v)
	}
	if v, ok := msg["brightness"].(float64); ok {
		s.Brightness = clamp(v, 0, 1)
	}
	if v, ok := msg["runTest"].(string); ok {
		s.pushDiag(diag.Diagnostic{Severity: diag.Info, Code: "TEST.RUNNING", Summary: "Running test", Detail: v})
		switch v {
		case string(tests.IndexSweep):
			s.testRunner = tests.NewRunner(tests.Plan{Kind: tests.IndexSweep})
		case string(tests.RGBTest):
			s.testRunner = tests.NewRunner(tests.Plan{Kind: tests.RGBTest})
		case string(tests.PlaneZ):
			s.testRunner = tests.NewRunner(tests.Plan{Kind: tests.PlaneZ})
		default:
			s.pushDiag(diag.Diagnostic{
				Severity: diag.Warn, Code: "TEST.UNKNOWN", Summary: "Unknown test name",
				Evidence: map[string]any{"name": v},
			})
		}
	}

	// Persist config after any change
	s.saveConfig()
}

func (s *State) saveConfig() {
	if s.ConfigPath == "" {
		return
	}
	cfg := &config.Config{
		Driver: func() string {
			if s.SimOnly {
				return "sim"
			}
			return "spi"
		}(),
		GPIO:            18,
		ColorOrder:      "GRB",
		Brightness:      s.Brightness,
		FPS:             s.FPS,
		Dim:             config.Dim{X: s.Layout.Dim.X, Y: s.Layout.Dim.Y, Z: s.Layout.Dim.Z},
		PitchMM:         s.Layout.PitchMM,
		PanelGapMM:      s.Layout.PanelGapMM,
		XFlipEveryRow:   s.Layout.Order.XFlipEveryRow,
		YFlipEveryPanel: s.Layout.Order.YFlipEveryPanel,
		Power: config.PowerCfg{
			LimitAmps:   35.0,
			WhiteCap:    0.85,
			SoftStartMs: 800,
		},
	}
	_ = config.Save(s.ConfigPath, cfg)
}

func (s *State) sendTopology(conn *websocket.Conn) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	top := map[string]any{
		"dim":        map[string]int{"x": s.Layout.Dim.X, "y": s.Layout.Dim.Y, "z": s.Layout.Dim.Z},
		"order":      map[string]bool{"xFlipEveryRow": s.Layout.Order.XFlipEveryRow, "yFlipEveryPanel": s.Layout.Order.YFlipEveryPanel},
		"panelGapMM": s.Layout.PanelGapMM,
		"pitchMM":    s.Layout.PitchMM,
		"driver":     s.CurrentDriver,
	}
	b, _ := json.Marshal(top)
	_ = conn.WriteMessage(websocket.TextMessage, b)
}

func (s *State) broadcastFrame(rgb []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	type frame struct {
		T       int64  `json:"t"`
		FrameID uint64 `json:"frame_id"`
		RGB     []byte `json:"rgb"`
	}
	b, _ := json.Marshal(frame{T: time.Now().UnixNano(), FrameID: s.frameID, RGB: rgb})
	for c := range s.clients {
		c.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
		if err := c.WriteMessage(websocket.TextMessage, b); err != nil {
			log.Debug().Err(err).Msg("write frame")
		}
	}
}

func (s *State) pushDiag(d diag.Diagnostic) {
	b, _ := json.Marshal(d)
	for c := range s.diagClients {
		c.SetWriteDeadline(time.Now().Add(200 * time.Millisecond))
		_ = c.WriteMessage(websocket.TextMessage, b)
	}
}

func hsvToRGB(h, s, v float64) (float64, float64, float64) {
	i := int(h * 6.0)
	f := h*6.0 - float64(i)
	p := v * (1.0 - s)
	q := v * (1.0 - f*s)
	t := v * (1.0 - (1.0-f)*s)
	switch i % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// estimateCurrent returns estimated amps given rgb frame (20mA/chan full-scale)
func estimateCurrent(rgb []byte) float64 {
	var sum float64
	for i := 0; i+2 < len(rgb); i += 3 {
		sum += float64(rgb[i]) + float64(rgb[i+1]) + float64(rgb[i+2])
	}
	// Approx: each color up to 20mA; scale 0..255
	return sum / 255.0 * 0.020
}

// applyWhiteCap clamps per-LED RGB so r+g+b <= whiteCap*3*255
func applyWhiteCap(rgb []byte, whiteCap float64) {
	if whiteCap <= 0 || whiteCap >= 1 {
		return
	}
	limit := whiteCap * 3.0 * 255.0
	for i := 0; i+2 < len(rgb); i += 3 {
		s := float64(rgb[i]) + float64(rgb[i+1]) + float64(rgb[i+2])
		if s > limit && s > 0 {
			scale := limit / s
			r := float64(rgb[i]) * scale
			g := float64(rgb[i+1]) * scale
			b := float64(rgb[i+2]) * scale
			rb := byte(math.Round(r))
			gb := byte(math.Round(g))
			bb := byte(math.Round(b))
			rgb[i], rgb[i+1], rgb[i+2] = rb, gb, bb
		}
	}
}
