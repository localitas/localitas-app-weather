package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// metCacheEntry is a forecast MET Norway gave, kept until its Expires and
// then asked for again with its Last-Modified, as MET's terms require.
type metCacheEntry struct {
	forecast     *MetForecast
	expires      time.Time
	lastModified string
}

const metCacheMax = 256

var metCache = struct {
	sync.Mutex
	entries map[string]metCacheEntry
}{entries: map[string]metCacheEntry{}}

func resetMetCache() {
	metCache.Lock()
	defer metCache.Unlock()
	metCache.entries = map[string]metCacheEntry{}
}

func rememberMet(key string, e metCacheEntry) {
	metCache.Lock()
	defer metCache.Unlock()
	if len(metCache.entries) >= metCacheMax {
		now := time.Now()
		for k, old := range metCache.entries {
			if now.After(old.expires) {
				delete(metCache.entries, k)
			}
		}
	}
	metCache.entries[key] = e
}

// metCoord is a coordinate cut to the 4 decimals MET Norway accepts.
func metCoord(v float64) string {
	s := strconv.FormatFloat(v, 'f', 6, 64)
	return s[:len(s)-2]
}

// fetchMET gets MET Norway's forecast for lat, lon: the one in hand while
// it has not expired, otherwise MET's answer to If-Modified-Since. A 429
// (MET throttling the app) is not asked again.
func fetchMET(ctx context.Context, lat, lon float64) (*MetForecast, error) {
	la, lo := metCoord(lat), metCoord(lon)
	key := la + "," + lo
	metCache.Lock()
	cached, ok := metCache.entries[key]
	metCache.Unlock()
	if ok && time.Now().Before(cached.expires) {
		return cached.forecast, nil
	}

	up := metUpstream
	if ok && cached.lastModified != "" {
		up.Header = http.Header{"If-Modified-Since": {cached.lastModified}}
	}
	resp, err := up.Get(ctx, "MET Norway", metURL+"?lat="+la+"&lon="+lo)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	expires, _ := http.ParseTime(resp.Header.Get("Expires"))

	switch {
	case resp.StatusCode == http.StatusNotModified && ok:
		cached.expires = expires
		rememberMet(key, cached)
		return cached.forecast, nil
	case resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNonAuthoritativeInfo:
		if resp.StatusCode == http.StatusNonAuthoritativeInfo {
			logger.Warn("MET Norway says Locationforecast 2.0 is deprecated", "status", resp.StatusCode)
		}
		var f MetForecast
		if err := json.NewDecoder(resp.Body).Decode(&f); err != nil {
			return nil, upstreamFailed("MET Norway", fmt.Errorf("parse: %w", err))
		}
		if len(f.Properties.Timeseries) == 0 {
			return nil, upstreamFailed("MET Norway", fmt.Errorf("no forecast in the answer"))
		}
		rememberMet(key, metCacheEntry{forecast: &f, expires: expires, lastModified: resp.Header.Get("Last-Modified")})
		return &f, nil
	default:
		return nil, upstreamFailed("MET Norway", fmt.Errorf("returned %d", resp.StatusCode))
	}
}

func metNorwayForecast(ctx context.Context, name string, lat, lon float64, loc *time.Location) (*WeatherResult, error) {
	data, err := fetchMET(ctx, lat, lon)
	if err != nil {
		return nil, err
	}
	return formatMET(name, lat, lon, data, time.Now(), loc), nil
}

func fahrenheit(c float64) float64 { return round1(c*9/5 + 32) }
func mph(ms float64) float64       { return round1(ms * 2.2369363) }
func round1(v float64) float64     { return math.Round(v*10) / 10 }

func percent(p *float64) *int {
	if p == nil {
		return nil
	}
	return ptr(int(math.Round(*p)))
}

// symbol is what a step says of the hours after it: its next hour's
// symbol, or the next six or twelve hours' where MET forecasts no finer.
func (s MetStep) symbol() string {
	for _, p := range []*MetPeriod{s.Data.Next1Hours, s.Data.Next6Hours, s.Data.Next12Hours} {
		if p != nil && p.Summary.SymbolCode != "" {
			return p.Summary.SymbolCode
		}
	}
	return ""
}

// formatMET is MET's forecast as the app shows one: current conditions at
// the step now is in, the 24 hours after now, and 7 days of loc's clock.
// MET gives no visibility and only a clear-sky UV index, so those are
// null; sunrise and sunset are computed from where and when.
func formatMET(name string, lat, lon float64, data *MetForecast, now time.Time, loc *time.Location) *WeatherResult {
	steps := data.Properties.Timeseries
	cur := steps[0]
	for _, s := range steps {
		if s.Time.After(now) {
			break
		}
		cur = s
	}
	in := cur.Data.Instant.Details
	current := &CurrentWeather{
		Temperature:   fahrenheit(in.AirTemperature),
		Conditions:    metConditions(cur.symbol()),
		Humidity:      int(math.Round(in.RelativeHumidity)),
		WindSpeed:     mph(in.WindSpeed),
		WindDirection: windDir(in.WindFromDirection),
		CloudCover:    int(math.Round(in.CloudAreaFraction)),
	}
	if in.ApparentAirTemperature != nil {
		current.FeelsLike = ptr(fahrenheit(*in.ApparentAirTemperature))
	}
	if p := cur.Data.Next1Hours; p != nil && p.Details.PrecipitationAmount != nil {
		current.Precipitation = *p.Details.PrecipitationAmount / 25.4
	}

	result := &WeatherResult{
		Location: name,
		Lat:      lat,
		Lon:      lon,
		Source:   metSource,
		TimeZone: loc.String(),
		Current:  current,
	}

	for _, s := range steps {
		if len(result.Hourly) == 24 {
			break
		}
		if !s.Time.After(now) || s.Data.Next1Hours == nil {
			continue
		}
		d := s.Data.Instant.Details
		h := HourlyForecast{
			Time:       s.Time.In(loc).Format("3 PM"),
			Temp:       fahrenheit(d.AirTemperature),
			Conditions: metConditions(s.symbol()),
			PrecipProb: percent(s.Data.Next1Hours.Details.ProbabilityOfPrecipitation),
			WindSpeed:  mph(d.WindSpeed),
			WindDir:    windDir(d.WindFromDirection),
		}
		if d.ApparentAirTemperature != nil {
			h.FeelsLike = ptr(fahrenheit(*d.ApparentAirTemperature))
		}
		result.Hourly = append(result.Hourly, h)
	}

	today := now.In(loc)
	for i := 0; i < 7; i++ {
		start := time.Date(today.Year(), today.Month(), today.Day()+i, 0, 0, 0, 0, loc)
		day, ok := metDay(steps, start, start.AddDate(0, 0, 1))
		if !ok {
			break
		}
		switch i {
		case 0:
			day.Day = "Today"
		case 1:
			day.Day = "Tomorrow"
		default:
			day.Day = start.Format("Monday")
		}
		day.Date = start.Format("2006-01-02")
		day.Sunrise, day.Sunset = sunTimes(lat, lon, start.Year(), start.Month(), start.Day(), loc)
		result.Daily = append(result.Daily, day)
	}
	return result
}

// metDay sums up the steps from start until end. The high and low are of
// the temperatures at each step and of the six-hour highs and lows that lie
// inside the day; the chance of precipitation is the highest MET gives for
// a period inside the day; the conditions are of the twelve hours from the
// step nearest 6 AM.
func metDay(steps []MetStep, start, end time.Time) (DailyForecast, bool) {
	var day DailyForecast
	found := false
	high, low := math.Inf(-1), math.Inf(1)
	var chance *float64
	conditions, nearest := "", time.Duration(-1)
	morning := start.Add(6 * time.Hour)

	for _, s := range steps {
		if s.Time.Before(start) || !s.Time.Before(end) {
			continue
		}
		found = true
		d := s.Data.Instant.Details
		high, low = math.Max(high, d.AirTemperature), math.Min(low, d.AirTemperature)
		day.WindMax = math.Max(day.WindMax, mph(d.WindSpeed))

		for _, p := range []struct {
			period *MetPeriod
			length time.Duration
		}{{s.Data.Next1Hours, time.Hour}, {s.Data.Next6Hours, 6 * time.Hour}, {s.Data.Next12Hours, 12 * time.Hour}} {
			if p.period == nil || s.Time.Add(p.length).After(end) {
				continue
			}
			pd := p.period.Details
			if p.length == 6*time.Hour && pd.AirTemperatureMax != nil && pd.AirTemperatureMin != nil {
				high, low = math.Max(high, *pd.AirTemperatureMax), math.Min(low, *pd.AirTemperatureMin)
			}
			if c := pd.ProbabilityOfPrecipitation; c != nil && (chance == nil || *c > *chance) {
				chance = c
			}
		}

		distance := s.Time.Sub(morning)
		if distance < 0 {
			distance = -distance
		}
		if s.Data.Next12Hours != nil && (nearest < 0 || distance < nearest) {
			conditions, nearest = s.Data.Next12Hours.Summary.SymbolCode, distance
		}
	}
	if !found {
		return day, false
	}
	day.High, day.Low = fahrenheit(high), fahrenheit(low)
	day.PrecipProb = percent(chance)
	day.Conditions = metConditions(conditions)
	return day, true
}

// metConditions reads one of MET Norway's weather symbol codes, such as
// "lightrainshowers_day", as the app's words for it.
func metConditions(code string) string {
	base, _, _ := strings.Cut(code, "_")
	switch base {
	case "clearsky":
		return "Clear sky"
	case "fair":
		return "Mainly clear"
	case "partlycloudy":
		return "Partly cloudy"
	case "cloudy":
		return "Cloudy"
	case "fog":
		return "Foggy"
	}
	if strings.Contains(base, "thunder") {
		return "Thunderstorm"
	}
	intensity := ""
	if rest, ok := strings.CutPrefix(base, "light"); ok {
		intensity, base = "Light ", rest
	} else if rest, ok := strings.CutPrefix(base, "heavy"); ok {
		intensity, base = "Heavy ", rest
	}
	kind, showers := strings.CutSuffix(base, "showers")
	switch kind {
	case "rain", "sleet", "snow":
	default:
		return "Unknown"
	}
	if showers {
		kind += " showers"
	}
	if intensity == "" {
		return strings.ToUpper(kind[:1]) + kind[1:]
	}
	return intensity + kind
}

// sunTimes is the sunrise and sunset at lat, lon on a day, in loc's clock,
// by the sunrise equation (within a minute or two of the almanac); both are
// empty when the sun stays up or down all day.
func sunTimes(lat, lon float64, year int, month time.Month, day int, loc *time.Location) (string, string) {
	const j2000 = 2451545.0
	rad := math.Pi / 180
	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
	jStar := float64(midnight.Unix())/86400 + 2440587.5 - j2000 + 0.5 - lon/360

	m := math.Mod(357.5291+0.98560028*jStar, 360)
	c := 1.9148*math.Sin(m*rad) + 0.02*math.Sin(2*m*rad) + 0.0003*math.Sin(3*m*rad)
	lambda := math.Mod(m+c+180+102.9372, 360)
	transit := j2000 + jStar + 0.0053*math.Sin(m*rad) - 0.0069*math.Sin(2*lambda*rad)
	sinDecl := math.Sin(lambda*rad) * math.Sin(23.4397*rad)
	cosDecl := math.Cos(math.Asin(sinDecl))
	cosHour := (math.Sin(-0.833*rad) - math.Sin(lat*rad)*sinDecl) / (math.Cos(lat*rad) * cosDecl)
	if cosHour < -1 || cosHour > 1 {
		return "", ""
	}
	hour := math.Acos(cosHour) / rad / 360
	clock := func(julian float64) string {
		return time.Unix(int64(math.Round((julian-2440587.5)*86400)), 0).In(loc).Format("3:04 PM")
	}
	return clock(transit - hour), clock(transit + hour)
}
