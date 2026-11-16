---
title: Weather
description: Weather forecasts and current conditions
---

# Weather

Get current weather conditions and forecasts for any location worldwide.

Full API spec: swagger.json

## API Endpoints

Method | Path | Description
GET | /api/weather?q={location} | Current conditions + forecast

## Weather Lookup

Provide a zip code, city name, or address to get current conditions and multi-day forecast. The app geocodes the location using Nominatim and fetches weather data from Open-Meteo.

    GET /api/weather?q=San+Francisco
    GET /api/weather?q=94102

## Response Data

Each weather response includes:
- Location name and coordinates
- Current temperature, humidity, and wind speed
- Weather condition description
- Multi-day forecast with daily highs and lows

## Data Sources

- **Nominatim (OpenStreetMap)** - Converts location names to geographic coordinates
- **Open-Meteo** - Provides weather data without requiring an API key

## Web Interface

The browser UI provides a search box for location lookup. Enter a city or address to see current conditions and a forecast overview rendered server-side.

## Units

Temperature values are returned in the units configured by Open-Meteo (Celsius by default). Wind speed is reported in km/h.

## Build & Deploy

Run these in the app's repository, `~/localitas-app-weather`.

### Version

```bash
bin/weather-server --version
```

### Build from source

```bash
# Native binary (bin/weather-server)
make build

# Linux amd64 binary the Docker image is built from (weather-server-linux-amd64)
make build-linux
```

Both run `make lint` (gofmt and go vet) first; `make test` runs the lint step and then the tests.

### Run natively

```bash
make start   # builds bin/weather-server, writes the launchd service com.localitas.app.weather for it and loads it on port 9206
make stop
make logs    # follows ~/.localitas/logs/weather/stdout.log
```

The app announces itself to core when it starts, and core serves it at `/apps/ext/weather/`.

### Docker

```bash
make build-docker   # builds the Linux binary and the weather:latest image
make start-docker   # runs container "weather" on port 9206, labelled localitas.app=true
make stop-docker
make logs-docker
```

Core registers containers labelled `localitas.app=true` once Docker reports them healthy.

### Publish

```bash
# Run the tests, build the image, push ghcr.io/localitas/localitas-app-weather:latest and :<VERSION>
make docker-push

# Build dist/weather-server-darwin-arm64 and dist/weather-server-darwin-amd64
# and create GitHub release vX.Y.Z on this repository with them
make release VERSION=vX.Y.Z
```

`VERSION` defaults to `git describe --tags --always --dirty`.

### Download

Pre-built macOS binaries are on the [releases page](https://github.com/localitas/localitas-app-weather/releases):
- `weather-server-darwin-arm64` (Apple Silicon)
- `weather-server-darwin-amd64` (Intel)

Every push to `main` publishes a release: `.github/workflows/release.yml` runs the tests and releases both binaries as `v0.1.<commit count>.<UTC timestamp>`. `make release` publishes one by hand.

Download with the GitHub CLI:

    gh release download --repo localitas/localitas-app-weather --pattern 'weather-server-*'
