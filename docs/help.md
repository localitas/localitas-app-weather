---
title: Weather
description: Weather forecasts and current conditions
---

# Weather

Get current weather conditions and forecasts for any location worldwide.

Full API spec: swagger.json

## API Endpoints

Method | Path | Description
GET | /api/weather?q={location}&tz={time zone} | Current conditions + forecast

## Weather Lookup

Provide a zip code, city name, or address to get current conditions, 24 hours and 7 days of forecast. The app finds the place with zippopotam.us (US zip codes) or Nominatim, then asks Open-Meteo for the forecast, and MET Norway when Open-Meteo does not answer.

    GET /api/weather?q=San+Francisco
    GET /api/weather?q=94102&tz=America/Los_Angeles

`tz` is optional: an IANA time zone in which to show MET Norway's times (the web page sends the browser's). Without it they are in UTC. Open-Meteo always gives times in the place's own clock. An unknown `tz` is a 400.

## Response Data

Each weather response includes:
- Location name and coordinates
- `source`: the provider that forecast it, its licence, and how the app changed its data
- `time_zone`: the time zone the hours, days, sunrise and sunset are in
- Current temperature, feels-like, humidity, wind, cloud cover, visibility, UV index and precipitation
- 24 hourly forecasts and 7 daily forecasts with highs and lows, chance of precipitation, sunrise and sunset

A value the answering provider does not give is `null`, never a guess, and the web page shows it as "—".

## Data Sources

- **zippopotam.us** and **Nominatim (OpenStreetMap)** - Turn zip codes and place names into coordinates
- **Open-Meteo** - The forecast, asked first
- **MET Norway** (api.met.no, Locationforecast 2.0) - The same forecast when Open-Meteo fails or stalls

Both forecast providers are free, need no API key, and license their data CC BY 4.0; the page credits whichever answered under the forecast.

### When Open-Meteo does not answer

Open-Meteo is tried up to 3 times, each try given 5 s. When it has failed, or its first try has not answered within 5 s, MET Norway is asked too (3 tries of 8 s), and the first forecast either gives is the answer. With both stalling the lookup ends after about 30 s; with both failing it is a 502 whose error says how each try of each provider failed.

The app follows MET Norway's terms: its User-Agent names the app and this repository, coordinates are cut to 4 decimals, a forecast is reused until its `Expires` and then revalidated with `If-Modified-Since`, and a 429 (throttled) is not retried.

### What MET Norway's forecast gives

Field | Open-Meteo | MET Norway
Temperature, humidity, wind, cloud cover, conditions | yes | yes
Feels like | yes | yes (MET's apparent temperature)
Precipitation now | yes | MET's forecast for this hour
Visibility, UV index, daily UV | yes | null
Chance of precipitation | yes | only where MET forecasts it (around the Nordics), else null
Daily high and low | yes | from MET's hourly values and 6-hour highs and lows inside the day
Daily conditions | yes | MET's 12 hours from the step nearest 6 AM
Sunrise and sunset | yes | computed from the place and date (within a minute or two)

## Web Interface

The browser UI provides a search box for location lookup. Enter a zip code or city to see current conditions, the next 24 hours and the next 7 days. The provider that answered and the time zone of the times are shown under the forecast.

## Units

Temperatures are in °F, wind in mph, precipitation in inches and visibility in miles, whichever provider answered.

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
