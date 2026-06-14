package weather

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

type APIEndpoint struct {
	Method      string     `json:"method"`
	Path        string     `json:"path"`
	Summary     string     `json:"summary"`
	QueryParams []APIParam `json:"query_params,omitempty"`
	Response    *APIBody   `json:"response,omitempty"`
}

type APIParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
}

type APIBody struct {
	ContentType string `json:"content_type"`
	Example     string `json:"example"`
}

type APIDoc struct {
	AppName     string        `json:"app_name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Keywords    []string      `json:"keywords,omitempty"`
	Endpoints   []APIEndpoint `json:"endpoints"`
}

var WeatherAPIDoc = APIDoc{
	AppName:     "Weather",
	Version:     "0.1.0",
	Description: "Weather forecast using Open-Meteo API. Supports US zip codes and worldwide city names. Returns current conditions, 24-hour hourly forecast, and 7-day daily forecast.",
	Keywords:    []string{"weather", "forecast", "temperature", "climate", "conditions", "rain", "snow", "wind", "humidity", "UV", "sunrise", "sunset", "hot", "cold", "warm"},
	Endpoints: []APIEndpoint{
		{
			Method:  "GET",
			Path:    "/api/weather",
			Summary: "Get weather forecast for a location",
			QueryParams: []APIParam{
				{Name: "q", Type: "string", Required: true, Description: "Location: US zip code (94102) or city name (Tokyo, London, Seoul)"},
			},
			Response: &APIBody{
				ContentType: "application/json",
				Example:     `{"location":"San Francisco, CA","current":{"temperature":62,"feels_like":58,"conditions":"Partly cloudy","humidity":72,"wind_speed":12,"wind_direction":"W","uv_index":5,"uv_level":"Moderate"},"hourly":[{"time":"2 PM","temperature":63,"conditions":"Clear sky","precip_probability":0}],"daily":[{"day":"Today","high":65,"low":52,"conditions":"Clear sky","sunrise":"6:30 AM","sunset":"7:45 PM"}]}`,
			},
		},
	},
}

func HandleSwagger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(WeatherAPIDoc)
}

func RenderDocsHTML(doc APIDoc) template.HTML {
	var sb strings.Builder
	sb.WriteString(`<h3 style="font-size: 0.875rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em; color: var(--color-text-secondary); margin-bottom: 1rem;">API</h3><div class="accordion-list">`)
	for _, ep := range doc.Endpoints {
		title := fmt.Sprintf("%s %s — %s", ep.Method, ep.Path, ep.Summary)
		sb.WriteString(fmt.Sprintf(`<details class="glass-panel" style="border-radius: 0.5rem; margin-bottom: 0.5rem;"><summary style="padding: 0.75rem 1rem; cursor: pointer; font-weight: 500; color: var(--color-text-primary);">%s</summary><div style="padding: 0 1rem 0.75rem; font-size: 0.875rem; color: var(--color-text-secondary);">`, template.HTMLEscapeString(title)))
		if ep.Response != nil {
			sb.WriteString(fmt.Sprintf(`<pre style="background: var(--color-bg-base); padding: 0.75rem; border-radius: 0.375rem; overflow-x: auto; font-size: 0.8125rem;">%s</pre>`, template.HTMLEscapeString(prettyJSON(ep.Response.Example))))
		}
		sb.WriteString(`</div></details>`)
	}
	sb.WriteString(`</div>`)
	return template.HTML(sb.String())
}

func prettyJSON(s string) string {
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return s
	}
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
