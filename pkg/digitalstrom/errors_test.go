package digitalstrom

import (
	"github.com/gaetancollaud/digitalstrom-mqtt/pkg/monitor"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
)

func TestRuntimeAuthFailureStopsButInvalidCommandDoesNot(t *testing.T) {
	for _, status := range []int{400, 401, 403, 503} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(status) }))
			defer server.Close()
			u, err := url.Parse(server.URL)
			require.NoError(t, err)
			port, err := strconv.Atoi(u.Port())
			require.NoError(t, err)
			progress := monitor.New()
			c := &client{httpClient: server.Client(), options: ClientOptions{Host: u.Hostname(), Port: port, Monitor: progress}}
			_, err = c.doRequest(http.MethodGet, "api/v1/apartment", nil, nil)
			require.Error(t, err)
			select {
			case <-progress.Failures():
				require.Contains(t, []int{401, 403}, status)
			default:
				require.NotContains(t, []int{401, 403}, status)
			}
			require.NoError(t, progress.Check(), "completed failed requests must not look stuck")
		})
	}
}

func TestResponseErrorClassification(t *testing.T) {
	for _, status := range []int{400, 401, 403, 404} {
		require.True(t, monitor.IsPermanent(responseError(status)))
	}
	for _, status := range []int{408, 429, 500, 503} {
		require.False(t, monitor.IsPermanent(responseError(status)))
	}
}
