package digitalstrom

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func newTestLegacyClient(t *testing.T, server *httptest.Server, apiKey string) *legacyClient {
	t.Helper()
	serverURL, _ := url.Parse(server.URL)
	port, _ := strconv.Atoi(serverURL.Port())
	client := NewLegacyClient(&ClientOptions{Host: serverURL.Hostname(), Port: port, ApiKey: apiKey}).(*legacyClient)
	client.httpClient = server.Client()
	return client
}

func TestLegacyClientSceneCalls(t *testing.T) {
	var mu sync.Mutex
	var calls []string
	polls := 0
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		query := request.URL.Query()
		calls = append(calls, strings.TrimPrefix(request.URL.Path, "/json/"))
		if request.URL.Path != "/json/system/loginApplication" && query.Get("token") != "session-token" {
			t.Errorf("unexpected token %q for %s", query.Get("token"), request.URL.Path)
		}
		switch request.URL.Path {
		case "/json/system/loginApplication":
			if query.Get("loginToken") != "test-api-key" {
				_, _ = writer.Write([]byte(`{"ok":false,"message":"Application-Authentication failed"}`))
				return
			}
			_, _ = writer.Write([]byte(`{"ok":true,"result":{"token":"session-token"}}`))
		case "/json/event/subscribe", "/json/event/unsubscribe", "/json/system/logout":
			_, _ = writer.Write([]byte(`{"ok":true}`))
		case "/json/event/get":
			polls++
			if polls == 1 {
				_, _ = writer.Write([]byte(`{"ok":true,"result":{"events":[
					{"name":"callScene","properties":{"sceneID":"17","groupID":"1","zoneID":"9999"},"source":{"zoneID":1234,"groupID":1}},
					{"name":"callScene","properties":{"sceneID":"72","groupID":"0"},"source":{"zoneID":0,"isApartment":true}},
					{"name":"callScene","properties":{"sceneID":"5"},"source":{"zoneID":1234,"isDevice":true}},
					{"name":"callScene","properties":{"sceneID":"0"},"source":{"zoneID":1234,"groupID":1,"isGroup":true}},
					{"name":"callScene","properties":{"sceneID":"5","groupID":"1"},"source":{"zoneID":666}},
					{"name":"undoScene","properties":{"sceneID":"5","groupID":"1"},"source":{"zoneID":1234}}
				]}}`))
				return
			}
			time.Sleep(20 * time.Millisecond)
			_, _ = writer.Write([]byte(`{"ok":true,"result":{}}`))
		case "/json/zone/getName":
			switch query.Get("id") {
			case "1234":
				_, _ = writer.Write([]byte(`{"ok":true,"result":{"name":" Living Room"}}`))
			case "0":
				_, _ = writer.Write([]byte(`{"ok":true,"result":{"name":""}}`))
			default:
				_, _ = writer.Write([]byte(`{"ok":false,"message":"Could not find zone"}`))
			}
		case "/json/zone/sceneGetName":
			if query.Get("sceneNumber") == "17" {
				_, _ = writer.Write([]byte(`{"ok":true,"result":{"name":"Movie"}}`))
				return
			}
			_, _ = writer.Write([]byte(`{"ok":true,"result":{"name":""}}`))
		default:
			t.Errorf("unexpected path %s", request.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestLegacyClient(t, server, "test-api-key")
	received := make(chan SceneCall, 10)
	if err := client.SceneCallsStart(func(call SceneCall) { received <- call }); err != nil {
		t.Fatalf("start: %v", err)
	}
	var got []SceneCall
	for len(got) < 4 {
		select {
		case call := <-received:
			got = append(got, call)
		case <-time.After(5 * time.Second):
			t.Fatalf("timeout waiting for scene calls, got %+v", got)
		}
	}
	client.SceneCallsStop()

	expected := []SceneCall{
		// The zone comes from the source, the name is not trimmed.
		{ZoneId: 1234, ZoneName: " Living Room", GroupId: 1, SceneId: 17, SceneName: "Movie"},
		{ZoneId: 0, ZoneName: "unnamed-zone-0", GroupId: 0, SceneId: 72, SceneName: "unnamed-scene-72"},
		// Calls without a group have the group -1 and no scene name.
		{ZoneId: 1234, ZoneName: " Living Room", GroupId: -1, SceneId: 5, SceneName: ""},
		// The group is read from the source when the properties don't have it.
		{ZoneId: 1234, ZoneName: " Living Room", GroupId: 1, SceneId: 0, SceneName: "unnamed-scene-0"},
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("call %d: got %+v, expected %+v", i, got[i], expected[i])
		}
	}
	select {
	case call := <-received:
		t.Errorf("unexpected call %+v: unknown zones and undo must be dropped", call)
	default:
	}

	mu.Lock()
	defer mu.Unlock()
	if calls[0] != "system/loginApplication" || calls[1] != "event/subscribe" {
		t.Errorf("unexpected first calls %v", calls[:2])
	}
	if last := calls[len(calls)-2:]; last[0] != "event/unsubscribe" || last[1] != "system/logout" {
		t.Errorf("session not closed on stop, last calls %v", last)
	}
}

func TestLegacyClientErrorsDoNotExposeCredentials(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	client := newTestLegacyClient(t, server, "secret-api-key")
	server.Close()

	err := client.subscribe(context.Background())
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "secret-api-key") || strings.Contains(err.Error(), "loginToken") {
		t.Fatalf("error exposes the API key: %v", err)
	}
}

func TestLegacyClientNameDecodeErrorIsNotCached(t *testing.T) {
	responses := []string{`{"ok":true,"result":"invalid"}`, `{"ok":true,"result":{"name":"Movie"}}`}
	server := httptest.NewTLSServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(responses[0]))
		responses = responses[1:]
	}))
	defer server.Close()
	client := newTestLegacyClient(t, server, "test-api-key")

	if _, err := client.sceneName(context.Background(), 1, 1, 17); err == nil {
		t.Fatal("expected a decode error")
	}
	if name, err := client.sceneName(context.Background(), 1, 1, 17); err != nil || name != "Movie" {
		t.Fatalf("got %q, %v", name, err)
	}
}
