package weather

import "time"

type OpenMeteoResponse struct {
	Timezone string `json:"timezone"`
	Current  struct {
		Time          string  `json:"time"`
		Temperature   float64 `json:"temperature_2m"`
		ApparentTemp  float64 `json:"apparent_temperature"`
		Humidity      int     `json:"relative_humidity_2m"`
		WeatherCode   int     `json:"weather_code"`
		WindSpeed     float64 `json:"wind_speed_10m"`
		WindDirection float64 `json:"wind_direction_10m"`
		Precipitation float64 `json:"precipitation"`
		CloudCover    int     `json:"cloud_cover"`
		Visibility    float64 `json:"visibility"`
		UVIndex       float64 `json:"uv_index"`
	} `json:"current"`
	Hourly struct {
		Time          []string  `json:"time"`
		Temperature   []float64 `json:"temperature_2m"`
		ApparentTemp  []float64 `json:"apparent_temperature"`
		PrecipProb    []int     `json:"precipitation_probability"`
		Precipitation []float64 `json:"precipitation"`
		WeatherCode   []int     `json:"weather_code"`
		WindSpeed     []float64 `json:"wind_speed_10m"`
		WindDirection []float64 `json:"wind_direction_10m"`
		Humidity      []int     `json:"relative_humidity_2m"`
		CloudCover    []int     `json:"cloud_cover"`
		Visibility    []float64 `json:"visibility"`
		UVIndex       []float64 `json:"uv_index"`
	} `json:"hourly"`
	Daily struct {
		Time          []string  `json:"time"`
		TempMax       []float64 `json:"temperature_2m_max"`
		TempMin       []float64 `json:"temperature_2m_min"`
		WeatherCode   []int     `json:"weather_code"`
		PrecipSum     []float64 `json:"precipitation_sum"`
		PrecipProbMax []int     `json:"precipitation_probability_max"`
		Sunrise       []string  `json:"sunrise"`
		Sunset        []string  `json:"sunset"`
		UVIndexMax    []float64 `json:"uv_index_max"`
		WindSpeedMax  []float64 `json:"wind_speed_10m_max"`
	} `json:"daily"`
}

// MetForecast is MET Norway's Locationforecast 2.0 "complete" answer: a
// series of times, hourly for about two and a half days and six-hourly
// after, each with the values at that instant and summaries of the hour,
// six hours and twelve hours that start there.
type MetForecast struct {
	Properties struct {
		Timeseries []MetStep `json:"timeseries"`
	} `json:"properties"`
}

type MetStep struct {
	Time time.Time `json:"time"`
	Data struct {
		Instant struct {
			Details MetInstant `json:"details"`
		} `json:"instant"`
		Next1Hours  *MetPeriod `json:"next_1_hours"`
		Next6Hours  *MetPeriod `json:"next_6_hours"`
		Next12Hours *MetPeriod `json:"next_12_hours"`
	} `json:"data"`
}

// MetInstant is in MET's units: °C, %, degrees, m/s.
type MetInstant struct {
	AirTemperature         float64  `json:"air_temperature"`
	ApparentAirTemperature *float64 `json:"apparent_air_temperature"`
	CloudAreaFraction      float64  `json:"cloud_area_fraction"`
	RelativeHumidity       float64  `json:"relative_humidity"`
	WindFromDirection      float64  `json:"wind_from_direction"`
	WindSpeed              float64  `json:"wind_speed"`
}

// MetPeriod summarizes the hours after a step; MET gives probabilities only
// for some regions, so each detail may be missing.
type MetPeriod struct {
	Summary struct {
		SymbolCode string `json:"symbol_code"`
	} `json:"summary"`
	Details struct {
		AirTemperatureMax          *float64 `json:"air_temperature_max"`
		AirTemperatureMin          *float64 `json:"air_temperature_min"`
		PrecipitationAmount        *float64 `json:"precipitation_amount"`
		ProbabilityOfPrecipitation *float64 `json:"probability_of_precipitation"`
	} `json:"details"`
}

// WeatherResult is one forecast in US units (°F, mph, inches, miles). A
// value the provider that answered does not give is null, never a guess.
type WeatherResult struct {
	Location string           `json:"location"`
	Lat      float64          `json:"latitude"`
	Lon      float64          `json:"longitude"`
	Source   Source           `json:"source"`
	TimeZone string           `json:"time_zone"`
	Current  *CurrentWeather  `json:"current"`
	Hourly   []HourlyForecast `json:"hourly"`
	Daily    []DailyForecast  `json:"daily"`
}

// Source is the provider whose forecast this is, credited as its licence
// asks.
type Source struct {
	Name       string `json:"name"`
	URL        string `json:"url"`
	License    string `json:"license"`
	LicenseURL string `json:"license_url"`
	// Changes says how the app changed the provider's data, as CC BY asks.
	Changes string `json:"changes,omitempty"`
}

type CurrentWeather struct {
	Temperature   float64  `json:"temperature"`
	FeelsLike     *float64 `json:"feels_like"`
	Conditions    string   `json:"conditions"`
	Humidity      int      `json:"humidity"`
	WindSpeed     float64  `json:"wind_speed"`
	WindDirection string   `json:"wind_direction"`
	CloudCover    int      `json:"cloud_cover"`
	Visibility    *float64 `json:"visibility_miles"`
	UVIndex       *float64 `json:"uv_index"`
	UVLevel       string   `json:"uv_level"`
	Precipitation float64  `json:"precipitation_inches"`
}

type HourlyForecast struct {
	Time       string   `json:"time"`
	Temp       float64  `json:"temperature"`
	FeelsLike  *float64 `json:"feels_like"`
	Conditions string   `json:"conditions"`
	PrecipProb *int     `json:"precip_probability"`
	WindSpeed  float64  `json:"wind_speed"`
	WindDir    string   `json:"wind_direction"`
}

type DailyForecast struct {
	Day        string   `json:"day"`
	Date       string   `json:"date"`
	High       float64  `json:"high"`
	Low        float64  `json:"low"`
	Conditions string   `json:"conditions"`
	PrecipProb *int     `json:"precip_probability"`
	WindMax    float64  `json:"wind_max"`
	UVMax      *float64 `json:"uv_max"`
	Sunrise    string   `json:"sunrise"`
	Sunset     string   `json:"sunset"`
}
