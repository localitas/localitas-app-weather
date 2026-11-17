package weather

import (
	"context"
	"math"
	"testing"
)

// Both real providers forecast San Jose, and agree on what both can know:
// the temperature roughly, and tomorrow's sunrise and sunset to the minute
// or two. It needs the internet, like the integration test that asks
// through core.
func TestLive_BothProvidersForecastSanJose(t *testing.T) {
	resetMetCache()
	t.Cleanup(resetMetCache)
	ctx := context.Background()

	om, err := openMeteoForecast(ctx, "San Jose, California", 37.3382, -121.8863)
	if err != nil {
		t.Fatalf("Open-Meteo: %v", err)
	}
	met, err := metNorwayForecast(ctx, "San Jose, California", 37.3382, -121.8863, zone(t, om.TimeZone))
	if err != nil {
		t.Fatalf("MET Norway: %v", err)
	}

	for _, res := range []*WeatherResult{om, met} {
		if res.Current == nil || res.Current.Conditions == "" || res.Current.Conditions == "Unknown" {
			t.Errorf("%s: current conditions %+v", res.Source.Name, res.Current)
		}
		if len(res.Hourly) != 24 || len(res.Daily) != 7 {
			t.Errorf("%s: %d hours and %d days, want 24 and 7", res.Source.Name, len(res.Hourly), len(res.Daily))
		}
	}
	if om.Source.Name != "Open-Meteo" || met.Source.Name != "MET Norway" {
		t.Errorf("sources %q and %q", om.Source.Name, met.Source.Name)
	}
	if d := math.Abs(om.Current.Temperature - met.Current.Temperature); d > 15 {
		t.Errorf("Open-Meteo says %.1f°F and MET %.1f°F", om.Current.Temperature, met.Current.Temperature)
	}
	for _, sun := range []struct{ name, om, met string }{
		{"sunrise", om.Daily[1].Sunrise, met.Daily[1].Sunrise},
		{"sunset", om.Daily[1].Sunset, met.Daily[1].Sunset},
	} {
		if d := clockMinutes(t, sun.om) - clockMinutes(t, sun.met); d < -2 || d > 2 {
			t.Errorf("tomorrow's %s: Open-Meteo %s, MET computed %s", sun.name, sun.om, sun.met)
		}
	}
	t.Logf("Open-Meteo %.1f°F %s; MET Norway %.1f°F %s; tomorrow's sun %s–%s / %s–%s",
		om.Current.Temperature, om.Current.Conditions, met.Current.Temperature, met.Current.Conditions,
		om.Daily[1].Sunrise, om.Daily[1].Sunset, met.Daily[1].Sunrise, met.Daily[1].Sunset)
}
