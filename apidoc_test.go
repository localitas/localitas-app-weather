package weather

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandleSwagger(t *testing.T) {
	w := httptest.NewRecorder()
	HandleSwagger(w, httptest.NewRequest("GET", "/swagger.json", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var spec APIDoc
	if err := json.Unmarshal(w.Body.Bytes(), &spec); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if spec.AppName != "Weather" {
		t.Errorf("expected app_name Weather, got %q", spec.AppName)
	}
	if len(spec.Endpoints) == 0 {
		t.Error("expected at least one endpoint")
	}
	hasWeather := false
	for _, ep := range spec.Endpoints {
		if strings.Contains(ep.Path, "/api/weather") {
			hasWeather = true
		}
	}
	if !hasWeather {
		t.Error("expected /api/weather endpoint")
	}
}

func TestWeatherDesc(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{0, "Clear sky"},
		{1, "Partly cloudy"},
		{61, "Rain"},
		{95, "Thunderstorm"},
		{75, "Snow"},
	}
	for _, tt := range tests {
		got := weatherDesc(tt.code)
		if got != tt.want {
			t.Errorf("weatherDesc(%d) = %q, want %q", tt.code, got, tt.want)
		}
	}
}

func TestWindDir(t *testing.T) {
	tests := []struct {
		deg  float64
		want string
	}{
		{0, "N"},
		{90, "E"},
		{180, "S"},
		{270, "W"},
		{45, "NE"},
	}
	for _, tt := range tests {
		got := windDir(tt.deg)
		if got != tt.want {
			t.Errorf("windDir(%.0f) = %q, want %q", tt.deg, got, tt.want)
		}
	}
}

func TestUVLevel(t *testing.T) {
	if uvLevel(1) != "Low" {
		t.Error("expected Low")
	}
	if uvLevel(5) != "Moderate" {
		t.Error("expected Moderate")
	}
	if uvLevel(10) != "Very High" {
		t.Error("expected Very High")
	}
}
