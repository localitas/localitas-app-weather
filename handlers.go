package weather

import (
	"errors"
	"net/http"
	"time"
	_ "time/tzdata"

	"github.com/localitas/localitas-go/httputil"
)

type handler struct{}

func (h *handler) handleLookup(w http.ResponseWriter, r *http.Request) {
	location := r.URL.Query().Get("q")
	if location == "" {
		writeErr(w, r, http.StatusBadRequest, "query parameter 'q' is required (zip code or city name)")
		return
	}
	loc := time.UTC
	if tz := r.URL.Query().Get("tz"); tz != "" {
		var err error
		if loc, err = time.LoadLocation(tz); err != nil {
			writeErr(w, r, http.StatusBadRequest, "unknown time zone %q", tz)
			return
		}
	}
	result, err := LookupWeather(r.Context(), location, loc)
	var failed *httputil.UpstreamError
	if errors.As(err, &failed) {
		writeErr(w, r, http.StatusBadGateway, "%v", err)
		return
	}
	if err != nil {
		writeErr(w, r, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, r, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, r *http.Request, status int, v interface{}) {
	httputil.WriteResponse(w, r, status, v)
}

func writeErr(w http.ResponseWriter, r *http.Request, status int, format string, args ...interface{}) {
	httputil.WriteError(w, r, status, format, args...)
}
