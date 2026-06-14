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

### Version

```bash
./weather-server --version
```

### Build from source

```bash
# Development (native)
cd apps/weather && go build -o bin/weather-server ./cmd/weather-server

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -trimpath -o bin/weather-server-linux-amd64 ./cmd/weather-server
```

### Docker

Build a Docker image directly from the binary:

```bash
# Default base image (debian:12-slim)
./weather-server docker-build

# Custom base image
./weather-server docker-build --base ubuntu:24.04

# Custom Dockerfile
./weather-server docker-build --dockerfile ./my.Dockerfile

# Tag and push to registry
./weather-server docker-build --tag ghcr.io/localitas/weather:latest --push
```

The `docker-build` command requires a Linux amd64 binary in the same directory. Run `make deploy-build` from the project root first.

### Download

Pre-built binaries are available on the [GitHub releases page](https://github.com/localitas/localitas/releases).

Each release includes three builds per app:
- `weather-server-darwin-arm64` (macOS Apple Silicon)
- `weather-server-linux-amd64` (Linux x86_64)
- `weather-server-linux-arm64` (Linux ARM64)

Download with the GitHub CLI:

    gh release download --repo localitas/localitas --pattern 'weather-server-*'

### Release

All app binaries are published to GitHub releases as part of `make deploy-upload-image`.
