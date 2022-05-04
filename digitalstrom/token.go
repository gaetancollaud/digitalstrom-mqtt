package digitalstrom

import (
	"time"

	"github.com/gaetancollaud/digitalstrom-mqtt/config"
	"github.com/rs/zerolog/log"
)

type TokenManager struct {
	config        *config.ConfigDigitalstrom
	httpClient    *HttpClient
	token         string
	lastTokenTime time.Time
	tokenCounter  int
}

func NewTokenManager(config *config.ConfigDigitalstrom, httpClient *HttpClient) *TokenManager {
	tm := new(TokenManager)
	tm.config = config
	tm.httpClient = httpClient
	tm.tokenCounter = 0
	return tm
}

func (tm *TokenManager) refreshToken() string {
	response, err := tm.httpClient.SystemLogin(tm.config.Username, tm.config.Password)

	if err != nil {
		log.Error().Err(err).Msg("Unable to refresh token, will wait a bit for next retry")
		time.Sleep(2 * time.Second)
		return ""
	}
	return response.Token
}

func (tm *TokenManager) GetToken() string {
	// no token, or more than 50sec
	if tm.token == "" {
		log.Debug().Dur("last token", time.Since(tm.lastTokenTime)).Msg("Refreshing token")
		tm.token = tm.refreshToken()
		tm.lastTokenTime = time.Now()
		tm.tokenCounter++
	}
	return tm.token
}

func (tm *TokenManager) InvalidateToken() {
	tm.token = ""
	log.Info().Msg("Invalidating token")
}
