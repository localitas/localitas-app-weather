package weather

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type handler struct{}

func (h *handler) handleLookup(w http.ResponseWriter, r *http.Request) {
	location := r.URL.Query().Get("q")
	if location == "" {
		writeErr(w, http.StatusBadRequest, "query parameter 'q' is required (zip code or city name)")
		return
	}
	result, err := LookupWeather(r.Context(), location)
	if err != nil {
		writeErr(w, http.StatusNotFound, "%v", err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, format string, args ...interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf(format, args...)})
}
