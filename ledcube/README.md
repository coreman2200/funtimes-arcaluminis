
# LED Cube Debugger (Wails + Web)

Runs with or without hardware (use `-sim-only`). Adjustable topology (X/Y/Z, pitch, gap). Built-in test modes.

## Build on macOS
```bash
cd web && npm i && npm run build && cd ..
go build -o bin/ledcube ./cmd/ledcube
./bin/ledcube -sim-only -x 5 -y 26 -z 5 -panel-gap-mm 50 -pitch-mm 17.6
# open http://localhost:8080
```

## Wails desktop (macOS / Pi)
```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
wails build -platform darwin/arm64   # macOS app in build/bin/LEDCube.app
# On Raspberry Pi (Linux/arm64), install GTK/WebKit deps then:
# sudo apt install -y libgtk-3-dev libwebkit2gtk-4.1-dev build-essential pkg-config libglib2.0-dev
# wails build -platform linux/arm64
```

## Endpoints
- `GET /`            — serves the embedded UI (after `web` build)
- `WS /ws/frames`    — topology on connect + frame stream `{rgb: []byte}`
- `WS /ws/control`   — send control JSON, e.g. `{"dim":{"x":5,"y":26,"z":5}}` or `{"runTest":"index_sweep"}`
- `WS /ws/diag`      — diagnostics stream (UI can subscribe)
- `GET /api/health`  — simple JSON health

## Next planned (not yet implemented)
- PWM driver for Pi 5 (GPIO18, rpi_ws281x) + first-run wizard
- Power estimator + limiter + diagnostics panel UI
- Systemd install script for headless boot
- Sensors (INA3221/INA219) stubs & telemetry


## Headless install on Raspberry Pi (service)
```bash
# Build artifacts first (web + server)
cd web && npm i && npm run build && cd ..
GOOS=linux GOARCH=arm64 go build -o bin/ledcube ./cmd/ledcube

# Copy to Pi (example)
# scp -r bin web README.md config.yaml.sample docs packaging pi@raspberrypi:/home/pi/ledcube-wails
# On the Pi:
cd ~/ledcube-wails && sudo ./packaging/install.sh
sudo systemctl enable --now ledcube
```

## Config
See `config.yaml.sample`. On first run you can copy it to `/opt/ledcube/config.yaml` and adjust.


## PWM on Raspberry Pi 5 (driver flag)
Run the server with PWM output on GPIO18 (level-shift to 5V):
```bash
./bin/ledcube -driver=pwm -gpio 18 -x 5 -y 26 -z 5
```
Use `-sim-only` on laptops or if no hardware is connected.

## First‑run wizard
On first launch, the UI shows a small setup modal to pick driver and confirm dimensions.
You can also edit `config.yaml` (to be added) to persist these across reboots.

## Raspberry Pi 5: rpi_ws281x setup
```bash
sudo apt update
sudo apt install -y build-essential git cmake
git clone https://github.com/jgarff/rpi_ws281x.git
cd rpi_ws281x && mkdir build && cd build
cmake ..
make -j4
sudo make install   # installs libws2811
```

Then build the server (Linux/arm64) and run with PWM:
```bash
GOOS=linux GOARCH=arm64 go build -o bin/ledcube ./cmd/ledcube
./bin/ledcube -driver=pwm -gpio 18
```
