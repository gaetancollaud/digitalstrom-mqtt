package digitalstrom

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestGetAPIKeyRejectsMissingLocationHeader(t *testing.T) {
	client := http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusCreated,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("")),
		}, nil
	})}

	_, err := getApiKey(client, "dss.local", 8080, "token", "integration")
	if err == nil || !strings.Contains(err.Error(), "empty API key") {
		t.Fatalf("expected empty API key error, got %v", err)
	}
}

func TestDoRequestDoesNotExposeQueryValuesInTransportError(t *testing.T) {
	client := http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return nil, &url.Error{
			Op:  request.Method,
			URL: request.URL.String(),
			Err: errors.New("dial failed"),
		}
	})}
	params := url.Values{"password": {"SuperSecret123"}, "user": {"dssadmin"}}

	_, _, err := doRequest(client, http.MethodGet, "dss.local", 8080, "json/system/login", params, nil)
	if err == nil {
		t.Fatal("expected transport error")
	}
	if strings.Contains(err.Error(), "SuperSecret123") || strings.Contains(err.Error(), "password=") {
		t.Fatalf("transport error exposed query values: %v", err)
	}
	if !strings.Contains(err.Error(), "dial failed") {
		t.Fatalf("transport error lost useful cause: %v", err)
	}
}

func TestDoRequestDoesNotExposeErrorResponseBody(t *testing.T) {
	client := http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusUnauthorized,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("login rejected for SuperSecret123")),
		}, nil
	})}

	_, _, err := doRequest(client, http.MethodGet, "dss.local", 8080, "json/system/login", nil, nil)
	if err == nil {
		t.Fatal("expected HTTP error")
	}
	if strings.Contains(err.Error(), "SuperSecret123") {
		t.Fatalf("HTTP error exposed response body: %v", err)
	}
}
