package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/example/ledcube-wails/internal/layout"
	"github.com/example/ledcube-wails/internal/led"
	"github.com/example/ledcube-wails/internal/ws"
)

func main() {
	// Flags
	var (
		addr       = flag.String("addr", ":8080", "listen address (host:port)")
		uiDir      = flag.String("ui-dir", "./web/dist", "path to built frontend")
		simOnly    = flag.Bool("sim-only", false, "run without GPIO/LED hardware")
		x          = flag.Int("x", 5, "strips per panel (X)")
		y          = flag.Int("y", 26, "leds per strip (Y)")
		z          = flag.Int("z", 5, "panels (Z)")
		pitchMM    = flag.Float64("pitch-mm", 17.6, "LED pitch in mm (along Y)")
		gapMM      = flag.Float64("panel-gap-mm", 25, "panel gap in mm (along Z)")
		fps        = flag.Int("fps", 60, "target frames per second")
		brightness = flag.Float64("brightness", 0.8, "global brightness 0..1")
		driver     = flag.String("driver", "sim", "driver: pwm | sim")
		gpio       = flag.Int("gpio", 18, "data pin (BCM number)")
		color      = flag.String("color", "GRB", "LED color order")
		configPath = flag.String("config", "config.yaml", "path to config.yaml")
	)
	flag.Parse()

	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})

	l := layout.Layout{
		Dim:        layout.Dim{X: *x, Y: *y, Z: *z},
		Order:      layout.Serpentine{XFlipEveryRow: true, YFlipEveryPanel: false},
		PanelGapMM: *gapMM,
		PitchMM:    *pitchMM,
	}

	state := ws.NewState(l, *fps, *brightness, *simOnly)
	state.ConfigPath = *configPath

	// Select driver
	if *driver == "pwm" && !*simOnly {
		if drv, err := led.NewPWM(*gpio, l.Count(), *color, *brightness); err != nil {
			log.Warn().Err(err).Msg("PWM init failed; falling back to SIM")
			state.Driver = led.NewSim()
		} else {
			state.Driver = drv
		}
	} else {
		state.Driver = led.NewSim()
	}

	mux := http.NewServeMux()
	// WS + API
	mux.HandleFunc("/ws/frames", state.HandleFramesWS)
	mux.HandleFunc("/ws/control", state.HandleControlWS)
	mux.HandleFunc("/ws/diag", state.HandleDiagWS)
	mux.HandleFunc("/api/health", state.HandleHealth)

	// Static UI from disk (no go:embed)
	if _, err := os.Stat(*uiDir); err != nil {
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`<html><body>
<h3>UI not found</h3>
<p>Build it with <code>cd web && npm i && npm run build</code> and rerun, or pass <code>-ui-dir</code>.</p>
</body></html>`))
		})
	} else {
		mux.Handle("/", http.FileServer(http.Dir(*uiDir)))
	}

	srv := &http.Server{Addr: *addr, Handler: withCORS(mux)}

	// Start server
	go func() {
		log.Info().Str("addr", *addr).Msg("HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("server error")
		}
	}()

	// Start render loop
	go state.RunRenderLoop()

	// Graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info().Msg("shutting down")
	_ = srv.Close()
	if state.Driver != nil {
		_ = state.Driver.Close()
	}
}

func withCORS(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}
		h.ServeHTTP(w, r)
	})
}
