package weather

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const sanJoseFixture = "testdata/met-san-jose-2026-11-17T0041Z.json"

// providers stands in for Nominatim, Open-Meteo and MET Norway, which a test
// cannot make fail or stall on demand. Nominatim finds places; Open-Meteo
// answers its first failures with failStatus before it forecasts; MET
// answers with a saved forecast, or with metStatus when that is set.
type providers struct {
	openMeteoCalls atomic.Int32
	metCalls       atomic.Int32

	failures   int
	failStatus int
	metStatus  int
	metExpires time.Time

	mu          sync.Mutex
	metRequests []*http.Request
}

const metLastModified = "Tue, 17 Nov 2026 00:41:02 GMT"

func serveProviders(t *testing.T, places []nominatimPlace, failures, failStatus int) *providers {
	t.Helper()
	metFixture, err := os.ReadFile(sanJoseFixture)
	if err != nil {
		t.Fatal(err)
	}
	p := &providers{failures: failures, failStatus: failStatus, metExpires: time.Now().Add(time.Hour)}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /search", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(places)
	})
	mux.HandleFunc("GET /v1/forecast", func(w http.ResponseWriter, r *http.Request) {
		if int(p.openMeteoCalls.Add(1)) <= p.failures {
			http.Error(w, "bad gateway", p.failStatus)
			return
		}
		var f OpenMeteoResponse
		f.Timezone = "America/Los_Angeles"
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
	mux.HandleFunc("GET /weatherapi/locationforecast/2.0/complete", func(w http.ResponseWriter, r *http.Request) {
		p.metCalls.Add(1)
		p.mu.Lock()
		p.metRequests = append(p.metRequests, r)
		p.mu.Unlock()
		if p.metStatus != 0 {
			http.Error(w, "no forecast", p.metStatus)
			return
		}
		w.Header().Set("Expires", p.metExpires.UTC().Format(http.TimeFormat))
		w.Header().Set("Last-Modified", metLastModified)
		if r.Header.Get("If-Modified-Since") == metLastModified {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(metFixture)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	oldNominatim, oldOpenMeteo, oldMet := nominatimURL, openMeteoURL, metURL
	oldUpstream, oldOpenMeteoUp, oldMetUp, oldFallback := upstream, openMeteoUpstream, metUpstream, fallbackAfter
	nominatimURL, openMeteoURL, metURL = srv.URL+"/search", srv.URL+"/v1/forecast", srv.URL+"/weatherapi/locationforecast/2.0/complete"
	upstream.Delay, openMeteoUpstream.Delay, metUpstream.Delay = time.Millisecond, time.Millisecond, time.Millisecond
	resetMetCache()
	t.Cleanup(func() {
		nominatimURL, openMeteoURL, metURL = oldNominatim, oldOpenMeteo, oldMet
		upstream, openMeteoUpstream, metUpstream, fallbackAfter = oldUpstream, oldOpenMeteoUp, oldMetUp, oldFallback
		resetMetCache()
	})
	return p
}

func (p *providers) requestsToMet() []*http.Request {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]*http.Request(nil), p.metRequests...)
}

// stallOpenMeteo points Open-Meteo at a listener that takes the connection
// and never answers the TLS handshake, as Open-Meteo's server did for
// minutes at a time.
func stallOpenMeteo(t *testing.T) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var conns []net.Conn
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			conns = append(conns, c)
			mu.Unlock()
		}
	}()
	t.Cleanup(func() {
		ln.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, c := range conns {
			c.Close()
		}
	})
	openMeteoURL = "https://" + ln.Addr().String() + "/v1/forecast"
}

// Nominatim gives more decimals than MET Norway accepts.
var sanJose = []nominatimPlace{{DisplayName: "San Jose, California", Lat: "37.33821012", Lon: "-121.88632861"}}

func lookup(t *testing.T, query string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	(&handler{}).handleLookup(w, httptest.NewRequest("GET", "/api/weather?"+query, nil))
	return w
}

func decodeResult(t *testing.T, w *httptest.ResponseRecorder) WeatherResult {
	t.Helper()
	if w.Code != http.StatusOK {
		t.Fatalf("lookup = %d %s, want 200", w.Code, w.Body)
	}
	var res WeatherResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("decode %s: %v", w.Body, err)
	}
	return res
}

// Open-Meteo answering is Open-Meteo's forecast, and MET Norway is not
// asked.
func TestLookup_OpenMeteoAnsweringIsItsForecast(t *testing.T) {
	p := serveProviders(t, sanJose, 0, 0)

	res := decodeResult(t, lookup(t, "q=San+Jose"))
	if res.Source.Name != "Open-Meteo" || res.Current.Temperature != 61 || res.TimeZone != "America/Los_Angeles" {
		t.Errorf("got source %q, %v°, zone %q; want Open-Meteo's 61° in America/Los_Angeles", res.Source.Name, res.Current.Temperature, res.TimeZone)
	}
	if got := p.metCalls.Load(); got != 0 {
		t.Errorf("MET asked %d times while Open-Meteo answered", got)
	}
}

// A forecast service failing for a moment is tried again, and the person
// gets its forecast.
func TestLookup_AMomentaryUpstreamFailureIsTriedAgain(t *testing.T) {
	p := serveProviders(t, sanJose, 1, http.StatusBadGateway)

	res := decodeResult(t, lookup(t, "q=San+Jose"))
	if got := p.openMeteoCalls.Load(); got != 2 {
		t.Errorf("Open-Meteo asked %d times, want 2", got)
	}
	if res.Source.Name != "Open-Meteo" || p.metCalls.Load() != 0 {
		t.Errorf("source %q after MET was asked %d times, want Open-Meteo without MET", res.Source.Name, p.metCalls.Load())
	}
}

// Open-Meteo failing every try is answered with MET Norway's forecast.
func TestLookup_OpenMeteoFailingIsAnsweredByMET(t *testing.T) {
	p := serveProviders(t, sanJose, 100, http.StatusServiceUnavailable)

	w := lookup(t, "q=San+Jose&tz=America/Los_Angeles")
	res := decodeResult(t, w)
	if got := p.openMeteoCalls.Load(); got != 3 {
		t.Errorf("Open-Meteo asked %d times, want 3", got)
	}
	if res.Source.Name != "MET Norway" || res.TimeZone != "America/Los_Angeles" || res.Current == nil || res.Current.Conditions == "" || len(res.Daily) == 0 {
		t.Errorf("got %+v, want MET Norway's forecast in America/Los_Angeles", res)
	}
	for _, unknown := range []string{`"visibility_miles":null`, `"uv_index":null`, `"uv_max":null`} {
		if !strings.Contains(w.Body.String(), unknown) {
			t.Errorf("answer does not say %s: %s", unknown, w.Body)
		}
	}
}

// Open-Meteo taking the connection and never answering TLS is answered with
// MET Norway's forecast once Open-Meteo's first try stalls, long before
// Open-Meteo's tries run out.
func TestLookup_OpenMeteoStallingAtTLSIsAnsweredByMETInTime(t *testing.T) {
	p := serveProviders(t, sanJose, 0, 0)
	stallOpenMeteo(t)
	openMeteoUpstream.AttemptTimeout, fallbackAfter = 400*time.Millisecond, 400*time.Millisecond

	start := time.Now()
	res := decodeResult(t, lookup(t, "q=San+Jose"))
	took := time.Since(start)

	if res.Source.Name != "MET Norway" {
		t.Errorf("source = %q, want MET Norway", res.Source.Name)
	}
	if took < fallbackAfter || took > 2*fallbackAfter {
		t.Errorf("answered in %v, want MET's answer after Open-Meteo's first try stalled (%v) and before its second did", took, fallbackAfter)
	}
	if got := p.metCalls.Load(); got != 1 {
		t.Errorf("MET asked %d times, want 1", got)
	}
}

// Both providers failing is a bad gateway whose error says how each failed.
func TestLookup_BothProvidersFailingIs502NamingBoth(t *testing.T) {
	p := serveProviders(t, sanJose, 100, http.StatusServiceUnavailable)
	p.metStatus = http.StatusInternalServerError

	w := lookup(t, "q=San+Jose")
	if w.Code != http.StatusBadGateway {
		t.Fatalf("lookup = %d %s, want 502", w.Code, w.Body)
	}
	body := w.Body.String()
	for _, want := range []string{"Open-Meteo: 3 tries failed", "answered 503", "MET Norway: 3 tries failed", "answered 500"} {
		if !strings.Contains(body, want) {
			t.Errorf("502 body %s does not say %q", body, want)
		}
	}
}

// A place no one can find is not found, and no forecast is asked for.
func TestLookup_AnUnknownPlaceIs404(t *testing.T) {
	p := serveProviders(t, nil, 0, 0)

	if w := lookup(t, "q=Nowhere"); w.Code != http.StatusNotFound {
		t.Errorf("lookup = %d %s, want 404", w.Code, w.Body)
	}
	if got := p.openMeteoCalls.Load() + p.metCalls.Load(); got != 0 {
		t.Errorf("forecast asked %d times for no place", got)
	}
}

// A time zone that does not exist is the asker's mistake.
func TestLookup_AnUnknownTimeZoneIs400(t *testing.T) {
	serveProviders(t, sanJose, 0, 0)

	if w := lookup(t, "q=San+Jose&tz=Mars/Olympus_Mons"); w.Code != http.StatusBadRequest {
		t.Errorf("lookup = %d %s, want 400", w.Code, w.Body)
	}
}

// tryBudget is the longest an Upstream asks before giving up: every try
// timing out, with the pauses between them.
func tryBudget(attempts int, attemptTimeout, delay time.Duration) time.Duration {
	total := time.Duration(attempts) * attemptTimeout
	for i := 1; i < attempts; i++ {
		total += time.Duration(i) * delay
	}
	return total
}

// With both providers stalling, the lookup still ends in time for core's
// proxy and the integration test (about 60 s), with room left for
// geocoding.
func TestForecastBudget_BothStallingEndsWellWithinAMinute(t *testing.T) {
	openMeteo := tryBudget(3, openMeteoUpstream.AttemptTimeout, 500*time.Millisecond)
	met := fallbackAfter + tryBudget(3, metUpstream.AttemptTimeout, 500*time.Millisecond)
	worst := max(openMeteo, met)
	if worst > 35*time.Second {
		t.Errorf("both stalling takes %v (Open-Meteo %v, MET from %v on %v), want at most 35s", worst, openMeteo, fallbackAfter, met)
	}
	if fallbackAfter > openMeteoUpstream.AttemptTimeout {
		t.Errorf("MET asked after %v, later than Open-Meteo's first try gives up (%v)", fallbackAfter, openMeteoUpstream.AttemptTimeout)
	}
}
