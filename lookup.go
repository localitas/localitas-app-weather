package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

var (
	zippopotamURL = "https://api.zippopotam.us/us/"
	nominatimURL  = "https://nominatim.openstreetmap.org/search"
	openMeteoURL  = "https://api.open-meteo.com/v1/forecast"
	retryDelay    = 500 * time.Millisecond
)

// upstreamAttempts is how many times a public service is asked before its
// failure is the lookup's.
const upstreamAttempts = 3

// upstreamError is a failure of a service the lookup depends on, not of the
// place asked about.
type upstreamError struct{ err error }

func (e *upstreamError) Error() string { return e.err.Error() }
func (e *upstreamError) Unwrap() error { return e.err }

// getUpstream GETs u from service, asking again after a pause when it fails
// (a network error or a 5xx), as public APIs do for a moment now and then.
// A response it returns has a status below 500.
func getUpstream(ctx context.Context, service, u string) (*http.Response, error) {
	var last error
	for attempt := 0; attempt < upstreamAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, &upstreamError{fmt.Errorf("%s: %w", service, ctx.Err())}
			case <-time.After(retryDelay * time.Duration(attempt)):
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Localitas Weather/1.0")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			last = fmt.Errorf("%s failed: %w", service, err)
			continue
		}
		if resp.StatusCode >= http.StatusInternalServerError {
			resp.Body.Close()
			last = fmt.Errorf("%s returned %d", service, resp.StatusCode)
			continue
		}
		return resp, nil
	}
	return nil, &upstreamError{last}
}

type nominatimPlace struct {
	DisplayName string `json:"display_name"`
	Lat         string `json:"lat"`
	Lon         string `json:"lon"`
}

func LookupWeather(ctx context.Context, location string) (*WeatherResult, error) {
	if location == "" {
		return nil, fmt.Errorf("location is required")
	}

	lat, lon, name, err := geocode(ctx, location)
	if err != nil {
		return nil, err
	}

	raw, err := fetchOpenMeteo(ctx, lat, lon)
	if err != nil {
		return nil, err
	}

	return formatResult(name, lat, lon, raw), nil
}

func geocode(ctx context.Context, location string) (float64, float64, string, error) {
	if len(location) == 5 {
		if _, err := strconv.Atoi(location); err == nil {
			lat, lon, name, err := geocodeZip(ctx, location)
			if err == nil {
				return lat, lon, name, nil
			}
		}
	}
	return geocodeNominatim(ctx, location)
}

func geocodeZip(ctx context.Context, zip string) (float64, float64, string, error) {
	resp, err := getUpstream(ctx, "zip lookup", zippopotamURL+zip)
	if err != nil {
		return 0, 0, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, "", fmt.Errorf("no results for zip %s", zip)
	}
	var result struct {
		Places []struct {
			PlaceName string `json:"place name"`
			State     string `json:"state abbreviation"`
			Latitude  string `json:"latitude"`
			Longitude string `json:"longitude"`
		} `json:"places"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil || len(result.Places) == 0 {
		return 0, 0, "", fmt.Errorf("no results for zip %s", zip)
	}
	lat, _ := strconv.ParseFloat(result.Places[0].Latitude, 64)
	lon, _ := strconv.ParseFloat(result.Places[0].Longitude, 64)
	return lat, lon, fmt.Sprintf("%s, %s", result.Places[0].PlaceName, result.Places[0].State), nil
}

func geocodeNominatim(ctx context.Context, location string) (float64, float64, string, error) {
	u := fmt.Sprintf("%s?q=%s&format=json&limit=1&accept-language=en", nominatimURL, url.QueryEscape(location))
	resp, err := getUpstream(ctx, "geocoding", u)
	if err != nil {
		return 0, 0, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, 0, "", &upstreamError{fmt.Errorf("geocoding returned %d", resp.StatusCode)}
	}
	var results []nominatimPlace
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil || len(results) == 0 {
		return 0, 0, "", fmt.Errorf("no location found for: %s", location)
	}
	lat, _ := strconv.ParseFloat(results[0].Lat, 64)
	lon, _ := strconv.ParseFloat(results[0].Lon, 64)
	return lat, lon, results[0].DisplayName, nil
}

func fetchOpenMeteo(ctx context.Context, lat, lon float64) (*OpenMeteoResponse, error) {
	u := fmt.Sprintf(
		"%s?latitude=%.4f&longitude=%.4f"+
			"&current=temperature_2m,relative_humidity_2m,apparent_temperature,weather_code,wind_speed_10m,wind_direction_10m,precipitation,cloud_cover,visibility,uv_index"+
			"&hourly=temperature_2m,apparent_temperature,precipitation_probability,precipitation,weather_code,wind_speed_10m,wind_direction_10m,relative_humidity_2m,cloud_cover,visibility,uv_index"+
			"&daily=temperature_2m_max,temperature_2m_min,weather_code,precipitation_sum,precipitation_probability_max,sunrise,sunset,uv_index_max,wind_speed_10m_max"+
			"&timezone=auto&temperature_unit=fahrenheit&wind_speed_unit=mph&forecast_days=7",
		openMeteoURL, lat, lon)
	resp, err := getUpstream(ctx, "Open-Meteo", u)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, &upstreamError{fmt.Errorf("Open-Meteo returned %d", resp.StatusCode)}
	}
	var data OpenMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, &upstreamError{fmt.Errorf("parse Open-Meteo: %w", err)}
	}
	return &data, nil
}

func formatResult(location string, lat, lon float64, data *OpenMeteoResponse) *WeatherResult {
	result := &WeatherResult{
		Location: location,
		Lat:      lat,
		Lon:      lon,
		Current: &CurrentWeather{
			Temperature:   data.Current.Temperature,
			FeelsLike:     data.Current.ApparentTemp,
			Conditions:    weatherDesc(data.Current.WeatherCode),
			Humidity:      data.Current.Humidity,
			WindSpeed:     data.Current.WindSpeed,
			WindDirection: windDir(data.Current.WindDirection),
			CloudCover:    data.Current.CloudCover,
			Visibility:    data.Current.Visibility / 1609.34,
			UVIndex:       data.Current.UVIndex,
			UVLevel:       uvLevel(data.Current.UVIndex),
			Precipitation: data.Current.Precipitation / 25.4,
		},
	}

	now := time.Now()
	startIdx := 0
	for i, t := range data.Hourly.Time {
		ht, err := time.Parse("2006-01-02T15:04", t)
		if err == nil && ht.After(now) {
			startIdx = i
			break
		}
	}
	for i := startIdx; i < startIdx+24 && i < len(data.Hourly.Time); i++ {
		ht, _ := time.Parse("2006-01-02T15:04", data.Hourly.Time[i])
		result.Hourly = append(result.Hourly, HourlyForecast{
			Time:       ht.Format("3 PM"),
			Temp:       data.Hourly.Temperature[i],
			FeelsLike:  data.Hourly.ApparentTemp[i],
			Conditions: weatherDesc(data.Hourly.WeatherCode[i]),
			PrecipProb: data.Hourly.PrecipProb[i],
			WindSpeed:  data.Hourly.WindSpeed[i],
			WindDir:    windDir(data.Hourly.WindDirection[i]),
		})
	}

	for i := 0; i < len(data.Daily.Time) && i < 7; i++ {
		dt, _ := time.Parse("2006-01-02", data.Daily.Time[i])
		day := dt.Format("Monday")
		if i == 0 {
			day = "Today"
		} else if i == 1 {
			day = "Tomorrow"
		}
		sunrise, sunset := "", ""
		if i < len(data.Daily.Sunrise) {
			st, _ := time.Parse("2006-01-02T15:04", data.Daily.Sunrise[i])
			sunrise = st.Format("3:04 AM")
		}
		if i < len(data.Daily.Sunset) {
			st, _ := time.Parse("2006-01-02T15:04", data.Daily.Sunset[i])
			sunset = st.Format("3:04 PM")
		}
		result.Daily = append(result.Daily, DailyForecast{
			Day: day, Date: data.Daily.Time[i],
			High: data.Daily.TempMax[i], Low: data.Daily.TempMin[i],
			Conditions: weatherDesc(data.Daily.WeatherCode[i]),
			PrecipProb: data.Daily.PrecipProbMax[i],
			WindMax:    data.Daily.WindSpeedMax[i],
			UVMax:      data.Daily.UVIndexMax[i],
			Sunrise:    sunrise, Sunset: sunset,
		})
	}

	return result
}

func weatherDesc(code int) string {
	switch code {
	case 0:
		return "Clear sky"
	case 1, 2, 3:
		return "Partly cloudy"
	case 45, 48:
		return "Foggy"
	case 51, 53, 55:
		return "Drizzle"
	case 56, 57:
		return "Freezing drizzle"
	case 61, 63, 65:
		return "Rain"
	case 66, 67:
		return "Freezing rain"
	case 71, 73, 75:
		return "Snow"
	case 77:
		return "Snow grains"
	case 80, 81, 82:
		return "Rain showers"
	case 85, 86:
		return "Snow showers"
	case 95:
		return "Thunderstorm"
	case 96, 99:
		return "Thunderstorm with hail"
	default:
		return "Unknown"
	}
}

func windDir(degrees float64) string {
	dirs := []string{"N", "NE", "E", "SE", "S", "SW", "W", "NW"}
	return dirs[int((degrees+22.5)/45.0)%8]
}

func uvLevel(uv float64) string {
	switch {
	case uv < 3:
		return "Low"
	case uv < 6:
		return "Moderate"
	case uv < 8:
		return "High"
	case uv < 11:
		return "Very High"
	default:
		return "Extreme"
	}
}
