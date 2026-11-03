package weather

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// upstreams stands in for Nominatim and Open-Meteo, which a test cannot make
// fail on demand: Nominatim finds places, and Open-Meteo answers its first
// failures with the given status before it forecasts.
type upstreams struct {
	forecastCalls atomic.Int32
}

func serveUpstreams(t *testing.T, places []nominatimPlace, failures int, failStatus int) *upstreams {
	t.Helper()
	u := &upstreams{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(places)
	})
	mux.HandleFunc("GET /v1/forecast", func(w http.ResponseWriter, r *http.Request) {
		if int(u.forecastCalls.Add(1)) <= failures {
			http.Error(w, "bad gateway", failStatus)
			return
		}
		var f OpenMeteoResponse
		f.Current.Temperature = 61
		f.Daily.Time = []string{"2026-11-03"}
		f.Daily.TempMax = []float64{70}
		f.Daily.TempMin = []float64{50}
		f.Daily.WeatherCode = []int{0}
		f.Daily.PrecipProbMax = []int{0}
		f.Daily.WindSpeedMax = []float64{5}
		f.Daily.UVIndexMax = []float64{3}
		json.NewEncoder(w).Encode(f)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	oldNominatim, oldOpenMeteo, oldDelay := nominatimURL, openMeteoURL, retryDelay
	nominatimURL, openMeteoURL, retryDelay = srv.URL+"/search", srv.URL+"/v1/forecast", time.Millisecond
	t.Cleanup(func() { nominatimURL, openMeteoURL, retryDelay = oldNominatim, oldOpenMeteo, oldDelay })
	return u
}

var sanJose = []nominatimPlace{{DisplayName: "San Jose, California", Lat: "37.3382", Lon: "-121.8863"}}

func lookup(t *testing.T) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	(&handler{}).handleLookup(w, httptest.NewRequest("GET", "/api/weather?q=San+Jose", nil))
	return w
}

// A forecast service failing for a moment is tried again, and the person
// gets the forecast.
func TestLookup_AMomentaryUpstreamFailureIsTriedAgain(t *testing.T) {
	u := serveUpstreams(t, sanJose, 1, http.StatusBadGateway)

	w := lookup(t)
	if w.Code != http.StatusOK {
		t.Fatalf("lookup = %d %s, want 200", w.Code, w.Body)
	}
	if got := u.forecastCalls.Load(); got != 2 {
		t.Errorf("forecast asked %d times, want 2", got)
	}
}

// A forecast service that keeps failing is a bad gateway, not a place that
// was not found.
func TestLookup_AnUpstreamThatKeepsFailingIs502(t *testing.T) {
	u := serveUpstreams(t, sanJose, 100, http.StatusServiceUnavailable)

	if w := lookup(t); w.Code != http.StatusBadGateway {
		t.Errorf("lookup = %d %s, want 502", w.Code, w.Body)
	}
	if got := u.forecastCalls.Load(); got != upstreamAttempts {
		t.Errorf("forecast asked %d times, want %d", got, upstreamAttempts)
	}
}

// A place no one can find is not found, and is not asked about again.
func TestLookup_AnUnknownPlaceIs404(t *testing.T) {
	u := serveUpstreams(t, nil, 0, 0)

	if w := lookup(t); w.Code != http.StatusNotFound {
		t.Errorf("lookup = %d %s, want 404", w.Code, w.Body)
	}
	if got := u.forecastCalls.Load(); got != 0 {
		t.Errorf("forecast asked %d times for no place", got)
	}
}
