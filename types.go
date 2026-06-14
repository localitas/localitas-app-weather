package weather

type OpenMeteoResponse struct {
	Current struct {
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

type WeatherResult struct {
	Location string           `json:"location"`
	Lat      float64          `json:"latitude"`
	Lon      float64          `json:"longitude"`
	Current  *CurrentWeather  `json:"current"`
	Hourly   []HourlyForecast `json:"hourly"`
	Daily    []DailyForecast  `json:"daily"`
}

type CurrentWeather struct {
	Temperature   float64 `json:"temperature"`
	FeelsLike     float64 `json:"feels_like"`
	Conditions    string  `json:"conditions"`
	Humidity      int     `json:"humidity"`
	WindSpeed     float64 `json:"wind_speed"`
	WindDirection string  `json:"wind_direction"`
	CloudCover    int     `json:"cloud_cover"`
	Visibility    float64 `json:"visibility_miles"`
	UVIndex       float64 `json:"uv_index"`
	UVLevel       string  `json:"uv_level"`
	Precipitation float64 `json:"precipitation_inches"`
}

type HourlyForecast struct {
	Time       string  `json:"time"`
	Temp       float64 `json:"temperature"`
	FeelsLike  float64 `json:"feels_like"`
	Conditions string  `json:"conditions"`
	PrecipProb int     `json:"precip_probability"`
	WindSpeed  float64 `json:"wind_speed"`
	WindDir    string  `json:"wind_direction"`
}

type DailyForecast struct {
	Day        string  `json:"day"`
	Date       string  `json:"date"`
	High       float64 `json:"high"`
	Low        float64 `json:"low"`
	Conditions string  `json:"conditions"`
	PrecipProb int     `json:"precip_probability"`
	WindMax    float64 `json:"wind_max"`
	UVMax      float64 `json:"uv_max"`
	Sunrise    string  `json:"sunrise"`
	Sunset     string  `json:"sunset"`
}
