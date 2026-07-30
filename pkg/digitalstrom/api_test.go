package digitalstrom

import "testing"

func TestApartmentStatusDecodesWeatherMeasurements(t *testing.T) {
	response := map[string]any{
		"id": "apartment-1",
		"attributes": map[string]any{
			"weather": map[string]any{
				"rain": true,
			},
			"measurements": map[string]any{
				"temperature": 21.5,
				"brightness":  12345.0,
				"windSpeed":   2.25,
				"windGust":    4.75,
			},
		},
	}

	status, err := wrapApiResponse[ApartmentStatus](response, nil)
	if err != nil {
		t.Fatalf("expected apartment status to decode: %v", err)
	}
	assertFloatPointer(t, "temperature", status.Attributes.Measurements.Temperature, 21.5)
	assertFloatPointer(t, "brightness", status.Attributes.Measurements.Brightness, 12345.0)
	assertFloatPointer(t, "windSpeed", status.Attributes.Measurements.WindSpeed, 2.25)
	assertFloatPointer(t, "windGust", status.Attributes.Measurements.WindGust, 4.75)
	if status.Attributes.Weather.Rain == nil || !*status.Attributes.Weather.Rain {
		t.Fatalf("expected rain to decode as true, got %#v", status.Attributes.Weather.Rain)
	}
}

func TestApartmentStatusKeepsMissingWeatherMeasurementsUnset(t *testing.T) {
	status, err := wrapApiResponse[ApartmentStatus](map[string]any{"id": "apartment-1"}, nil)
	if err != nil {
		t.Fatalf("expected apartment status to decode: %v", err)
	}
	if status.Attributes.Measurements.Temperature != nil || status.Attributes.Weather.Rain != nil {
		t.Fatalf("expected missing values to remain nil, got %#v", status.Attributes)
	}
}

func assertFloatPointer(t *testing.T, name string, actual *float64, expected float64) {
	t.Helper()
	if actual == nil || *actual != expected {
		t.Fatalf("expected %s %.2f, got %#v", name, expected, actual)
	}
}
