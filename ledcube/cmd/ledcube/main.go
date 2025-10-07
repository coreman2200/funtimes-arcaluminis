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

	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/config"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/layout"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/led"
	"github.com/coreman2200/funtimes-arcaluminis/ledcube/internal/ws"
)

func main() {
	// ---- Flags (remain usable; config.yaml can override most) ----
	var (
		x          = flag.Int("x", 5, "LEDs per row (X)")
		y          = flag.Int("y", 26, "LED rows per panel (Y)")
		z          = flag.Int("z", 5, "Panels/depth (Z)")
		xFlip      = flag.Bool("x-flip-every-row", true, "serpentine: flip every row along X")
		yFlip      = flag.Bool("y-flip-every-panel", true, "serpentine: flip every panel along Y")
		pitchMM    = flag.Float64("pitch-mm", 10, "LED pitch (mm)")
		panelGapMM = flag.Float64("panel-gap-mm", 50, "panel gap (mm) along Z")
		fps        = flag.Int("fps", 60, "target frames per second")
		brightness = flag.Float64("brightness", 0.8, "global brightness 0..1")
		driver     = flag.String("driver", "sim", "driver: spi | pwm | sim")
		gpio       = flag.Int("gpio", 18, "PWM data pin (BCM number) for rpi_ws281x")
		colorOrder = flag.String("color", "GRB", "LED color order (e.g. GRB, RGB)")
		addr       = flag.String("addr", ":8080", "HTTP listen address")
		configPath = flag.String("config", "config.yaml", "path to config.yaml")
		simOnly    = flag.Bool("sim-only", false, "force simulation (no hardware output)")
	)
	flag.Parse()

	// ---- Logging ----
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.Kitchen})

	// ---- Load config.yaml (optional) ----
	var cfg *config.Config
	if c, err := config.Load(*configPath); err != nil {
		log.Warn().Err(err).Str("path", *configPath).Msg("config load failed; proceeding with flags")
	} else {
		cfg = c
	}

	// ---- Effective params (config overrides flags where available) ----
	eX, eY, eZ := *x, *y, *z
	eXFlip, eYFlip := *xFlip, *yFlip
	ePitch, eGap := *pitchMM, *panelGapMM
	eFPS, eBright := *fps, *brightness
	eColor := *colorOrder

	if cfg != nil {
		// Dimensions
		if cfg.Dim.X > 0 {
			eX = cfg.Dim.X
		}
		if cfg.Dim.Y > 0 {
			eY = cfg.Dim.Y
		}
		if cfg.Dim.Z > 0 {
			eZ = cfg.Dim.Z
		}
		// Geometry
		ePitch = firstNonZeroFloat(cfg.PitchMM, ePitch)
		eGap = firstNonZeroFloat(cfg.PanelGapMM, eGap)

		// Visuals
		if cfg.FPS > 0 {
			eFPS = cfg.FPS
		}
		if cfg.Brightness > 0 {
			eBright = cfg.Brightness
		}
		// Color order
		if cfg.ColorOrder != "" {
			eColor = cfg.ColorOrder
		}
	}

	// ---- Build layout ----
	l := layout.Layout{
		Dim:        layout.Dim{X: eX, Y: eY, Z: eZ},
		Order:      layout.Serpentine{XFlipEveryRow: eXFlip, YFlipEveryPanel: eYFlip},
		PanelGapMM: eGap,
		PitchMM:    ePitch,
	}

	// ---- State ----
	state := ws.NewState(l, eFPS, eBright, *simOnly)
	state.ConfigPath = *configPath

	// ---- Driver selection: -sim-only overrides; otherwise config.driver then -driver ----
	selected := *driver
	if cfg != nil && cfg.Driver != "" {
		selected = cfg.Driver
	}
	if *simOnly {
		selected = "sim"
	}

	switch selected {
	case "sim":
		state.Driver = led.NewSim()

	case "spi":
		// Defaults if YAML not filled
		spiDev := "/dev/spidev0.0"
		speedHz := 2400000
		resetUs := 300
		if cfg != nil {
			if cfg.SPI.Dev != "" {
				spiDev = cfg.SPI.Dev
			}
			if cfg.SPI.SpeedHz != 0 {
				speedHz = cfg.SPI.SpeedHz
			}
			if cfg.SPI.ResetUs != 0 {
				resetUs = cfg.SPI.ResetUs
			}
		}
		drv, err := led.NewSPI(spiDev, l.Count(), eColor, speedHz, resetUs)
		if err != nil {
			log.Warn().Err(err).
				Str("driver", "spi").
				Str("dev", spiDev).
				Int("speed_hz", speedHz).
				Msg("SPI init failed; falling back to SIM")
			state.Driver = led.NewSim()
		} else {
			state.Driver = drv
		}

	case "pwm":
		// Avoid compile errors if PWM isn't built in this snapshot.
		_ = gpio // keep flag in place without unused warning
		log.Warn().Msg("driver=pwm requested, but PWM is not compiled in this build; using SIM instead")
		state.Driver = led.NewSim()

	default:
		log.Warn().Str("driver", selected).Msg("unknown driver; using SIM")
		state.Driver = led.NewSim()
	}
	state.CurrentDriver = selected

	// ---- HTTP routes ----
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", state.HandleFramesWS)
	mux.HandleFunc("/diag", state.HandleDiagWS)
	mux.HandleFunc("/control", state.HandleControlWS)
	mux.HandleFunc("/health", state.HandleHealth)

	srv := &http.Server{
		Addr:         *addr,
		Handler:      withCORS(mux),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ---- Run render loop & server ----
	go state.RunRenderLoop()
	go func() {
		log.Info().Str("addr", *addr).Str("driver", selected).Msg("HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("http server crashed")
		}
	}()

	// ---- Graceful shutdown ----
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	s := <-ch
	log.Info().Str("signal", s.String()).Msg("shutting down")

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

func firstNonZeroFloat(v, fallback float64) float64 {
	if v != 0 {
		return v
	}
	return fallback
}
