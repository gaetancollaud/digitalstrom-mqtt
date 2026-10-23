package digitalstrom

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// The Smarthome API notifications don't carry any information about scene
// calls. Scene calls are only available through the legacy JSON event API
// (json/event/subscribe and json/event/get), which is still served by the
// dSS. The API key is an application token, which allows to open a session on
// the legacy API using json/system/loginApplication.
//
// The legacy client is only created by the scenes module when it is enabled,
// so nothing calls the legacy API when scenes are disabled.

const (
	legacyPollTimeout     = 25 * time.Second
	legacyRequestTimeout  = 10 * time.Second
	legacyHttpTimeout     = legacyPollTimeout + 15*time.Second
	legacyRetryMinDelay   = 5 * time.Second
	legacyRetryMaxDelay   = 5 * time.Minute
	legacySubscriptionMin = 1000
	legacyNoGroup         = -1
)

// SceneCall is a scene call reported by the digitalSTROM server, with the
// zone and scene names resolved like version 1.x did.
type SceneCall struct {
	ZoneId    int
	ZoneName  string
	GroupId   int
	SceneId   int
	SceneName string
}

type SceneCallCallback func(call SceneCall)

// LegacyClient reads the scene calls from the legacy JSON API of the dSS.
type LegacyClient interface {
	SceneCallsStart(callback SceneCallCallback) error
	SceneCallsStop()
}

type legacyResponse struct {
	Ok      bool            `json:"ok"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type legacyEvent struct {
	Name       string                 `json:"name"`
	Properties map[string]interface{} `json:"properties"`
	Source     map[string]interface{} `json:"source"`
}

type sceneIdentifier struct {
	ZoneId  int
	GroupId int
	SceneId int
}

type legacyClient struct {
	options        ClientOptions
	httpClient     *http.Client
	subscriptionId int

	mu     sync.Mutex
	cancel context.CancelFunc
	done   chan struct{}

	// Only used by the event loop goroutine: the session token is the one
	// the subscription was created in, so it's never replaced by another
	// caller.
	token      string
	zoneNames  map[int]string
	sceneNames map[sceneIdentifier]string
}

func NewLegacyClient(options *ClientOptions) LegacyClient {
	return &legacyClient{
		options: *options,
		httpClient: &http.Client{
			Timeout: legacyHttpTimeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
		subscriptionId: legacySubscriptionMin + rand.Intn(1000000),
		zoneNames:      map[int]string{},
		sceneNames:     map[sceneIdentifier]string{},
	}
}

func (c *legacyClient) SceneCallsStart(callback SceneCallCallback) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cancel != nil {
		return errors.New("scene calls listener already started")
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.done = make(chan struct{})
	go c.run(ctx, callback)
	return nil
}

func (c *legacyClient) SceneCallsStop() {
	c.mu.Lock()
	cancel, done := c.cancel, c.done
	c.cancel = nil
	c.mu.Unlock()
	if cancel == nil {
		return
	}
	cancel()
	<-done
}

func (c *legacyClient) run(ctx context.Context, callback SceneCallCallback) {
	defer close(c.done)
	defer c.closeSession()

	delay := legacyRetryMinDelay
	retry := func(err error, msg string) bool {
		log.Warn().Err(err).Dur("retryIn", delay).Msg(msg)
		c.closeSession()
		if !sleepContext(ctx, delay) {
			return false
		}
		delay = min(delay*2, legacyRetryMaxDelay)
		return true
	}

	for ctx.Err() == nil {
		if c.token == "" {
			if err := c.subscribe(ctx); err != nil {
				if ctx.Err() != nil || !retry(err, "Unable to subscribe to digitalSTROM scene events") {
					return
				}
				continue
			}
			log.Info().Int("subscriptionId", c.subscriptionId).Msg("Subscribed to digitalSTROM scene events")
		}

		events, err := c.poll(ctx)
		if err != nil {
			if ctx.Err() != nil || !retry(err, "Error polling digitalSTROM scene events, will resubscribe") {
				return
			}
			continue
		}
		delay = legacyRetryMinDelay
		for _, event := range events {
			log.Debug().
				Str("name", event.Name).
				Interface("properties", event.Properties).
				Interface("source", event.Source).
				Msg("digitalSTROM event received")
			call, ok := parseSceneCall(event)
			if !ok {
				continue
			}
			// Like version 1.x, the event is dropped when the names can't be
			// retrieved.
			if call.ZoneName, err = c.zoneName(ctx, call.ZoneId); err != nil {
				log.Warn().Err(err).Int("zoneId", call.ZoneId).Msg("Unable to get the zone name, scene event dropped")
				continue
			}
			if call.SceneName, err = c.sceneName(ctx, call.ZoneId, call.GroupId, call.SceneId); err != nil {
				log.Warn().Err(err).Int("zoneId", call.ZoneId).Int("sceneId", call.SceneId).Msg("Unable to get the scene name, scene event dropped")
				continue
			}
			callback(call)
		}
	}
}

func (c *legacyClient) subscribe(ctx context.Context) error {
	params := url.Values{}
	params.Set("loginToken", c.options.ApiKey)
	result, err := c.request(ctx, "json/system/loginApplication", params)
	if err != nil {
		return fmt.Errorf("error logging in on the digitalSTROM JSON API: %w", err)
	}
	var login struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(result, &login); err != nil || login.Token == "" {
		return errors.New("no session token returned by json/system/loginApplication")
	}
	c.token = login.Token

	params = url.Values{}
	params.Set("name", string(EventTypeCallScene))
	params.Set("subscriptionID", strconv.Itoa(c.subscriptionId))
	params.Set("token", c.token)
	if _, err := c.request(ctx, "json/event/subscribe", params); err != nil {
		return fmt.Errorf("error subscribing to %s: %w", EventTypeCallScene, err)
	}
	return nil
}

// closeSession unsubscribes and logs out, so the dSS doesn't keep abandoned
// sessions (their number is limited).
func (c *legacyClient) closeSession() {
	if c.token == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	params := url.Values{}
	params.Set("name", string(EventTypeCallScene))
	params.Set("subscriptionID", strconv.Itoa(c.subscriptionId))
	params.Set("token", c.token)
	_, _ = c.request(ctx, "json/event/unsubscribe", params)
	params = url.Values{}
	params.Set("token", c.token)
	_, _ = c.request(ctx, "json/system/logout", params)
	c.token = ""
}

func (c *legacyClient) poll(ctx context.Context) ([]legacyEvent, error) {
	params := url.Values{}
	params.Set("subscriptionID", strconv.Itoa(c.subscriptionId))
	params.Set("timeout", strconv.Itoa(int(legacyPollTimeout.Milliseconds())))
	params.Set("token", c.token)
	result, err := c.request(ctx, "json/event/get", params)
	if err != nil {
		return nil, err
	}
	var response struct {
		Events []legacyEvent `json:"events"`
	}
	if len(result) > 0 {
		if err := json.Unmarshal(result, &response); err != nil {
			return nil, fmt.Errorf("error decoding events: %w", err)
		}
	}
	log.Trace().Int("count", len(response.Events)).Msg("Scene events received")
	return response.Events, nil
}

func (c *legacyClient) zoneName(ctx context.Context, zoneId int) (string, error) {
	if name, ok := c.zoneNames[zoneId]; ok {
		return name, nil
	}
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	name, err := c.getName(ctx, "json/zone/getName", params)
	if err != nil {
		return "", err
	}
	if name == "" {
		name = "unnamed-zone-" + strconv.Itoa(zoneId)
	}
	c.zoneNames[zoneId] = name
	return name, nil
}

func (c *legacyClient) sceneName(ctx context.Context, zoneId int, groupId int, sceneId int) (string, error) {
	// Version 1.x didn't look up the name of calls without a group.
	if groupId == legacyNoGroup {
		return "", nil
	}
	id := sceneIdentifier{ZoneId: zoneId, GroupId: groupId, SceneId: sceneId}
	if name, ok := c.sceneNames[id]; ok {
		return name, nil
	}
	params := url.Values{}
	params.Set("id", strconv.Itoa(zoneId))
	params.Set("groupID", strconv.Itoa(groupId))
	params.Set("sceneNumber", strconv.Itoa(sceneId))
	name, err := c.getName(ctx, "json/zone/sceneGetName", params)
	if err != nil {
		return "", err
	}
	if name == "" {
		name = "unnamed-scene-" + strconv.Itoa(sceneId)
	}
	c.sceneNames[id] = name
	return name, nil
}

func (c *legacyClient) getName(ctx context.Context, path string, params url.Values) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, legacyRequestTimeout)
	defer cancel()
	params.Set("token", c.token)
	result, err := c.request(ctx, path, params)
	if err != nil {
		return "", err
	}
	var response struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(result, &response); err != nil {
		return "", fmt.Errorf("error decoding the response of %s: %w", path, err)
	}
	return response.Name, nil
}

// request calls the legacy JSON API and returns the content of the "result"
// field.
func (c *legacyClient) request(ctx context.Context, path string, params url.Values) (json.RawMessage, error) {
	callUrl := "https://" + c.options.Host + ":" + strconv.Itoa(c.options.Port) + "/" + path + "?" + params.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, callUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("error building the request %s: %w", path, err)
	}
	resp, err := c.httpClient.Do(request)
	if err != nil {
		// Drop the URL from the error: its query string carries the API key
		// and the session token.
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}
		return nil, fmt.Errorf("error doing the request %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading the response of %s: %w", path, err)
	}
	if resp.StatusCode >= 300 {
		return nil, responseError(resp.StatusCode)
	}
	var response legacyResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("error parsing response for path %s: %w", path, err)
	}
	if !response.Ok {
		return nil, fmt.Errorf("request %s failed: %s", path, response.Message)
	}
	return response.Result, nil
}

// parseSceneCall reads a callScene event like version 1.x: the zone comes
// from the source, and calls without a group (e.g. device scene calls) have
// the group -1. Unlike version 1.x, the group is also read from the source:
// some group calls (e.g. from sensors) only carry it there.
func parseSceneCall(event legacyEvent) (SceneCall, bool) {
	if EventType(event.Name) != EventTypeCallScene {
		return SceneCall{}, false
	}
	sceneId, ok := intField(event.Properties, "sceneID")
	if !ok {
		return SceneCall{}, false
	}
	zoneId, ok := intField(event.Source, "zoneID")
	if !ok {
		return SceneCall{}, false
	}
	groupId, ok := intField(event.Properties, "groupID")
	if !ok {
		groupId, ok = intField(event.Source, "groupID")
	}
	if !ok {
		groupId = legacyNoGroup
	}
	return SceneCall{ZoneId: zoneId, GroupId: groupId, SceneId: sceneId}, true
}

// intField reads a number that the dSS sends either as string or as number.
func intField(values map[string]interface{}, key string) (int, bool) {
	switch v := values[key].(type) {
	case float64:
		return int(v), true
	case string:
		i, err := strconv.Atoi(v)
		return i, err == nil
	}
	return 0, false
}

func sleepContext(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
