package weather

import (
	"net/http"

	"github.com/localitas/localitas-go/httputil"
)

type handler struct{}

func (h *handler) handleLookup(w http.ResponseWriter, r *http.Request) {
	location := r.URL.Query().Get("q")
	if location == "" {
		writeErr(w, r, http.StatusBadRequest, "query parameter 'q' is required (zip code or city name)")
		return
	}
	result, err := LookupWeather(r.Context(), location)
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
