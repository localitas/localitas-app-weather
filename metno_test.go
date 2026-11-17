package weather

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/localitas/localitas-go/httputil"
)

func loadMetFixture(t *testing.T, path string) *MetForecast {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var f MetForecast
	if err := json.Unmarshal(b, &f); err != nil {
		t.Fatal(err)
	}
	return &f
}

func zone(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func clockMinutes(t *testing.T, s string) int {
	t.Helper()
	c, err := time.Parse("3:04 PM", s)
	if err != nil {
		t.Fatalf("clock %q: %v", s, err)
	}
	return c.Hour()*60 + c.Minute()
}

func within(t *testing.T, name string, got, want, tolerance float64) {
	t.Helper()
	if got < want-tolerance || got > want+tolerance {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// MET Norway's forecast for San Jose, saved from api.met.no at 00:41 UTC on
// 17 November 2026, is the same forecast in the app's units and in San
// Jose's clock; what MET does not give there is null.
func TestFormatMET_SanJose(t *testing.T) {
	data := loadMetFixture(t, sanJoseFixture)
	now := time.Date(2026, 11, 17, 0, 41, 2, 0, time.UTC)

	res := formatMET("San Jose, California", 37.3382, -121.8863, data, now, zone(t, "America/Los_Angeles"))

	if res.Source.Name != "MET Norway" || res.Source.License != "CC BY 4.0" || res.TimeZone != "America/Los_Angeles" {
		t.Errorf("source %+v in %q, want MET Norway (CC BY 4.0) in America/Los_Angeles", res.Source, res.TimeZone)
	}

	c := res.Current
	within(t, "temperature", c.Temperature, 56.5, 0.05)
	if c.FeelsLike == nil {
		t.Fatal("feels like is null, MET gives it")
	}
	within(t, "feels like", *c.FeelsLike, 55.9, 0.05)
	within(t, "wind", c.WindSpeed, 4.7, 0.05)
	if c.Humidity != 68 || c.CloudCover != 27 || c.WindDirection != "W" || c.Conditions != "Mainly clear" {
		t.Errorf("humidity %d, clouds %d, wind from %s, %q; want 68, 27, W, Mainly clear", c.Humidity, c.CloudCover, c.WindDirection, c.Conditions)
	}
	if c.Visibility != nil || c.UVIndex != nil || c.UVLevel != "" {
		t.Errorf("visibility %v, UV %v %q; MET gives neither, want null", c.Visibility, c.UVIndex, c.UVLevel)
	}

	if len(res.Hourly) != 24 {
		t.Fatalf("%d hours, want 24", len(res.Hourly))
	}
	h := res.Hourly[0]
	if h.Time != "5 PM" || h.Conditions != "Mainly clear" || h.WindDir != "NW" || h.PrecipProb != nil {
		t.Errorf("first hour %+v, want 5 PM, Mainly clear, NW, no precipitation probability", h)
	}
	within(t, "first hour temperature", h.Temp, 54.1, 0.05)
	within(t, "first hour wind", h.WindSpeed, 2.0, 0.05)
	if res.Hourly[23].Time != "4 PM" {
		t.Errorf("last hour %q, want 4 PM", res.Hourly[23].Time)
	}

	if len(res.Daily) != 7 {
		t.Fatalf("%d days, want 7", len(res.Daily))
	}
	days := []struct {
		i              int
		day, date      string
		high, low      float64
		conditionsWant string
	}{
		{0, "Today", "2026-11-16", 56.5, 48.6, "Mainly clear"},
		{1, "Tomorrow", "2026-11-17", 57.7, 43.0, "Mainly clear"},
		{6, "Sunday", "2026-11-22", 57.9, 44.1, "Partly cloudy"},
	}
	for _, want := range days {
		d := res.Daily[want.i]
		if d.Day != want.day || d.Date != want.date || d.Conditions != want.conditionsWant {
			t.Errorf("day %d = %s %s %q, want %s %s %q", want.i, d.Day, d.Date, d.Conditions, want.day, want.date, want.conditionsWant)
		}
		within(t, want.date+" high", d.High, want.high, 0.05)
		within(t, want.date+" low", d.Low, want.low, 0.05)
		if d.UVMax != nil || d.PrecipProb != nil {
			t.Errorf("%s UV %v, precipitation %v; MET gives neither for San Jose, want null", want.date, d.UVMax, d.PrecipProb)
		}
	}
	if res.Daily[2].Day != "Wednesday" {
		t.Errorf("day 2 = %s, want Wednesday", res.Daily[2].Day)
	}

	// Open-Meteo gave San Jose's sunrise on 17 November 2026 as 06:49 and
	// its sunset as 16:55.
	tomorrow := res.Daily[1]
	if got := clockMinutes(t, tomorrow.Sunrise); got < 6*60+47 || got > 6*60+51 {
		t.Errorf("sunrise %s, want 6:49 AM within 2 minutes", tomorrow.Sunrise)
	}
	if got := clockMinutes(t, tomorrow.Sunset); got < 16*60+53 || got > 16*60+57 {
		t.Errorf("sunset %s, want 4:55 PM within 2 minutes", tomorrow.Sunset)
	}
}

// Where MET gives the chance of precipitation, as it does around the
// Nordics, the hour carries it and the day carries the highest chance of
// the periods inside it.
func TestFormatMET_OsloHasPrecipitationChances(t *testing.T) {
	data := loadMetFixture(t, "testdata/met-oslo-2026-11-17T0047Z.json")
	now := time.Date(2026, 11, 17, 0, 47, 19, 0, time.UTC)

	res := formatMET("Oslo, Norway", 59.9133, 10.7389, data, now, zone(t, "Europe/Oslo"))
	if len(res.Hourly) == 0 || len(res.Daily) < 2 {
		t.Fatalf("%d hours and %d days, want 24 and 7", len(res.Hourly), len(res.Daily))
	}

	if h := res.Hourly[0]; h.Time != "2 AM" || h.PrecipProb == nil || *h.PrecipProb != 0 {
		t.Errorf("first hour %s with chance %v, want 2 AM with 0%%", h.Time, h.PrecipProb)
	}
	if d := res.Daily[1]; d.Date != "2026-11-18" || d.PrecipProb == nil || *d.PrecipProb != 2 {
		t.Errorf("day 1 %s with chance %v, want 2026-11-18 with 2%%", d.Date, d.PrecipProb)
	}
	if res.Current.FeelsLike == nil {
		t.Fatal("feels like is null, MET gives it")
	}
	within(t, "feels like", *res.Current.FeelsLike, 29.1, 0.05)
}

// MET's symbol codes read as the app's conditions; the weather icon is
// chosen from the words.
func TestMetConditions(t *testing.T) {
	for code, want := range map[string]string{
		"clearsky_day":                     "Clear sky",
		"fair_night":                       "Mainly clear",
		"partlycloudy_polartwilight":       "Partly cloudy",
		"cloudy":                           "Cloudy",
		"fog":                              "Foggy",
		"lightrain":                        "Light rain",
		"rain":                             "Rain",
		"heavyrainshowers_day":             "Heavy rain showers",
		"sleet":                            "Sleet",
		"lightsleetshowers_night":          "Light sleet showers",
		"snowshowers_night":                "Snow showers",
		"heavysnow":                        "Heavy snow",
		"rainandthunder":                   "Thunderstorm",
		"lightssleetshowersandthunder_day": "Thunderstorm",
		"":                                 "Unknown",
		"volcano":                          "Unknown",
	} {
		if got := metConditions(code); got != want {
			t.Errorf("metConditions(%q) = %q, want %q", code, got, want)
		}
	}
}

// Where the sun neither rises nor sets that day, there is no sunrise or
// sunset to show.
func TestSunTimes_PolarNightAndMidnightSun(t *testing.T) {
	oslo := zone(t, "Europe/Oslo")
	for _, date := range []time.Time{
		time.Date(2026, 12, 20, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 6, 21, 0, 0, 0, 0, time.UTC),
	} {
		rise, set := sunTimes(69.6492, 18.9553, date.Year(), date.Month(), date.Day(), oslo)
		if rise != "" || set != "" {
			t.Errorf("Tromsø on %s: sunrise %q, sunset %q; want neither", date.Format("2 Jan"), rise, set)
		}
	}
}

// MET's terms: an app names itself and how to reach its makers, and asks
// for no more than 4 decimals of latitude and longitude.
func TestMET_RequestFollowsTheTermsOfService(t *testing.T) {
	p := serveProviders(t, sanJose, 0, 0)

	if _, err := fetchMET(context.Background(), 37.33821012, -121.88632861); err != nil {
		t.Fatal(err)
	}
	reqs := p.requestsToMet()
	if len(reqs) != 1 {
		t.Fatalf("%d requests to MET, want 1", len(reqs))
	}
	r := reqs[0]
	if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "Localitas Weather/") || !strings.Contains(ua, "https://github.com/localitas/localitas-app-weather") {
		t.Errorf("User-Agent %q, want the app's name and its repository", ua)
	}
	if lat, lon := r.URL.Query().Get("lat"), r.URL.Query().Get("lon"); lat != "37.3382" || lon != "-121.8863" {
		t.Errorf("asked for %s,%s, want 37.3382,-121.8863", lat, lon)
	}
}

// A forecast MET says has not expired is not asked for again.
func TestMET_AForecastIsReusedUntilItExpires(t *testing.T) {
	p := serveProviders(t, sanJose, 0, 0)

	for range 2 {
		if _, err := fetchMET(context.Background(), 37.3382, -121.8863); err != nil {
			t.Fatal(err)
		}
	}
	if got := p.metCalls.Load(); got != 1 {
		t.Errorf("MET asked %d times, want 1", got)
	}
}

// An expired forecast is asked for again with If-Modified-Since, and MET's
// 304 keeps the forecast in hand.
func TestMET_AnExpiredForecastIsRevalidated(t *testing.T) {
	p := serveProviders(t, sanJose, 0, 0)
	p.metExpires = time.Now().Add(-time.Minute)

	first, err := fetchMET(context.Background(), 37.3382, -121.8863)
	if err != nil {
		t.Fatal(err)
	}
	second, err := fetchMET(context.Background(), 37.3382, -121.8863)
	if err != nil {
		t.Fatal(err)
	}
	reqs := p.requestsToMet()
	if len(reqs) != 2 {
		t.Fatalf("%d requests to MET, want 2", len(reqs))
	}
	if got := reqs[1].Header.Get("If-Modified-Since"); got != metLastModified {
		t.Errorf("If-Modified-Since %q, want MET's Last-Modified %q", got, metLastModified)
	}
	if second != first {
		t.Error("304 did not keep the forecast in hand")
	}
}

// MET throttling the app is not asked again at once, and is a failure of
// the provider.
func TestMET_ThrottlingIsNotAskedAgain(t *testing.T) {
	p := serveProviders(t, sanJose, 0, 0)
	p.metStatus = http.StatusTooManyRequests

	_, err := fetchMET(context.Background(), 37.3382, -121.8863)
	var failed *httputil.UpstreamError
	if !errors.As(err, &failed) || !strings.Contains(err.Error(), "429") {
		t.Errorf("err = %v, want MET Norway's 429 as an upstream failure", err)
	}
	if got := p.metCalls.Load(); got != 1 {
		t.Errorf("MET asked %d times, want 1", got)
	}
}
