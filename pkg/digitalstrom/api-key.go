package digitalstrom

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

/**
 * This file is here only to generate a new API key for the integration. It should never be used for another purpose.
 */

type NewApiKeyRequest struct {
	Data NewApiKeyRequestData `json:"data"`
}

type NewApiKeyRequestData struct {
	Type       string                     `json:"type"`
	Attributes NewApiKeyRequestAttributes `json:"attributes"`
}
type NewApiKeyRequestAttributes struct {
	Name string `json:"name"`
}

func GetApiKey(host string, port int, user string, password string, integrationName string) (string, error) {
	httpClient := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	token, err := getToken(httpClient, host, port, user, password)
	if err != nil {
		return "", err
	}

	apiKey, err := getApiKey(httpClient, host, port, token, integrationName)
	if err != nil {
		return "", err
	}

	return apiKey, nil
}

func getToken(httpClient http.Client, host string, port int, user string, password string) (string, error) {
	params := url.Values{}
	params.Set("user", user)
	params.Set("password", password)

	tokenResponse, _, err := doRequest(httpClient, http.MethodGet, host, port, "json/system/login", params, nil)
	if err != nil {
		return "", fmt.Errorf("digitalSTROM login failed: %w", err)
	}

	loginAccepted, ok := tokenResponse["ok"].(bool)
	if !ok {
		return "", errors.New("no 'ok' field present, cannot check request")
	}
	if !loginAccepted {
		return "", errors.New("digitalSTROM login was rejected")
	}

	result, ok := tokenResponse["result"].(map[string]interface{})
	if !ok {
		return "", errors.New("no 'token' field present, cannot get token from request")
	}
	token, ok := result["token"].(string)
	if !ok || strings.TrimSpace(token) == "" {
		return "", errors.New("no 'token' field present, cannot get token from request")
	}
	return token, nil
}

func getApiKey(httpClient http.Client, host string, port int, token string, integrationName string) (string, error) {
	params := url.Values{}
	params.Set("token", token)

	requestBody := &NewApiKeyRequest{
		Data: NewApiKeyRequestData{
			Type: "applicationToken",
			Attributes: NewApiKeyRequestAttributes{
				Name: integrationName,
			},
		},
	}

	_, apiKeyResponse, err := doRequest(httpClient, http.MethodPost, host, port, "/api/v1/apartment/applicationTokens", params, requestBody)
	if err != nil {
		return "", fmt.Errorf("error when getting api key: %w", err)
	}

	var apiKey string
	if apiKeyResponse.StatusCode == 201 {
		apiKey = apiKeyResponse.Header.Get("Location")
	} else {
		return "", fmt.Errorf("error creating api key, status was: %d", apiKeyResponse.StatusCode)
	}
	if strings.TrimSpace(apiKey) == "" {
		return "", errors.New("digitalSTROM returned an empty API key")
	}

	return apiKey, nil
}

func doRequest(httpClient http.Client, method string, host string, port int, path string, params url.Values, body interface{}) (map[string]interface{}, *http.Response, error) {
	var bodyReader io.Reader = nil
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		bodyReader = strings.NewReader(string(jsonBody))
	}
	callUrl := "https://" + host + ":" + strconv.Itoa(port) + "/" + path
	if params != nil && len(params) > 0 {
		callUrl = callUrl + "?" + params.Encode()
	}

	request, err := http.NewRequest(method, callUrl, bodyReader)
	if err != nil {
		return nil, nil, errors.New("error building the request")
	}
	resp, err := httpClient.Do(request)
	if err != nil {
		requestError := err
		for {
			urlError, isURLError := requestError.(*url.Error)
			if !isURLError {
				break
			}
			requestError = urlError.Err
		}
		return nil, nil, fmt.Errorf("error doing the request: %w", requestError)
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	responseBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, nil, fmt.Errorf("error reading the response: %w", readErr)
	}

	if resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("error response from server, httpStatus=%d", resp.StatusCode)
	}

	log.Debug().
		Str("url", request.URL.Scheme+"://"+request.URL.Host+request.URL.Path).
		Str("status", resp.Status).
		Msg("Response received")
	log.Trace().
		Int("body_bytes", len(responseBody)).
		Msg("Response body received")

	if len(responseBody) > 0 {
		var jsonResponse map[string]interface{}
		err = json.Unmarshal(responseBody, &jsonResponse)
		if err != nil {
			return nil, nil, fmt.Errorf("error parsing response for token: %w", err)
		}
		return jsonResponse, resp, nil
	}

	return nil, resp, nil
}
