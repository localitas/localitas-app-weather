# Weather

Weather forecasts and current conditions.

Part of the [Localitas](https://github.com/localitas) platform — a self-hosted, privacy-first personal computing system.

## Features

- Current conditions by zip code or city
- Multi-day forecasts
- UV index, wind, humidity, and feels-like temperature
- No API key required (uses open weather data)

## Installation

### Development (via Localitas core)

```bash
# Clone the repo
git clone https://github.com/localitas/localitas-app-weather.git ~/localitas-app-weather

# Start with the Localitas dev cluster (builds and runs in Docker automatically)
cd ~/localitas && make dev-core
```

### Standalone

```bash
cd ~/localitas-app-weather

# Build and run locally
make build
./bin/weather-server serve --listen :8000

# Or via launchd (macOS)
make start

# Or via Docker
make start-docker
```

## Exposing to the Internet

Localitas apps are accessible remotely through Localitas's built-in tunnel service, powered by FRP. No port forwarding or dynamic DNS required.

1. Sign up at [localitas.com](https://localitas.com) and connect your local Localitas core
2. The tunnel automatically exposes your core (and all apps) at `https://{your-subdomain}.localitas.com`
3. This app is available at `https://{your-subdomain}.localitas.com/apps/ext/weather/`

All traffic is encrypted end-to-end. Authentication is handled by the Localitas core — only authorized users can access your apps.

## License

MIT
