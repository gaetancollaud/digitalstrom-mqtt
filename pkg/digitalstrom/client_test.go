package digitalstrom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestGetApartmentDecodesDeviceScenariosWithoutAdditionalInclude(t *testing.T) {
	var method string
	var requestPath string
	var include string
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method = request.Method
		requestPath = request.URL.Path
		include = request.URL.Query().Get("include")
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{
			"data": {
				"id": "apartment-1",
				"included": {
					"dsDevices": [{
						"id": "blind-1",
						"attributes": {
							"name": "Office Blind",
							"scenarios": ["device-blind-1-std.stop"]
						}
					}]
				}
			}
		}`))
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(serverURL.Port())
	if err != nil {
		t.Fatalf("parse test server port: %v", err)
	}
	client := &client{
		httpClient: server.Client(),
		options: ClientOptions{
			Host:   serverURL.Hostname(),
			Port:   port,
			ApiKey: "test-api-key",
		},
	}

	apartment, err := client.GetApartment()

	if err != nil {
		t.Fatalf("get apartment: %v", err)
	}
	if method != http.MethodGet {
		t.Fatalf("expected GET, got %s", method)
	}
	if requestPath != "/api/v1/apartment" {
		t.Fatalf("unexpected request path: %s", requestPath)
	}
	expectedInclude := "installation,dsDevices,submodules,functionBlocks,zones,controllers,meterings"
	if include != expectedInclude {
		t.Fatalf("expected include query %q, got %q", expectedInclude, include)
	}
	if len(apartment.Included.Devices) != 1 {
		t.Fatalf("expected one decoded device, got %d", len(apartment.Included.Devices))
	}
	scenarios := apartment.Included.Devices[0].Attributes.Scenarios
	if len(scenarios) != 1 || scenarios[0] != "device-blind-1-std.stop" {
		t.Fatalf("unexpected decoded scenarios: %#v", scenarios)
	}
}

func TestInvokeScenarioByIDPostsToScenarioResource(t *testing.T) {
	var method string
	var requestPath string
	var authorization string
	var requestBody map[string]interface{}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method = request.Method
		requestPath = request.URL.Path
		authorization = request.Header.Get("Authorization")
		if err := json.NewDecoder(request.Body).Decode(&requestBody); err != nil {
			t.Errorf("decode request body: %v", err)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	serverURL, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server URL: %v", err)
	}
	port, err := strconv.Atoi(serverURL.Port())
	if err != nil {
		t.Fatalf("parse test server port: %v", err)
	}
	client := &client{
		httpClient: server.Client(),
		options: ClientOptions{
			Host:   serverURL.Hostname(),
			Port:   port,
			ApiKey: "test-api-key",
		},
	}

	err = client.InvokeScenarioByID("device-blind-1-std.stop")

	if err != nil {
		t.Fatalf("invoke scenario by ID: %v", err)
	}
	if method != http.MethodPost {
		t.Fatalf("expected POST, got %s", method)
	}
	if requestPath != "/api/v1/apartment/scenarios/device-blind-1-std.stop/invoke" {
		t.Fatalf("unexpected request path: %s", requestPath)
	}
	if authorization != "Bearer test-api-key" {
		t.Fatalf("unexpected authorization header: %s", authorization)
	}
	if len(requestBody) != 0 {
		t.Fatalf("expected empty request body, got %#v", requestBody)
	}
}
