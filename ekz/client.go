package ekz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	Backend                 string = "https://be.emob.ekz.ch"
	MaxRefreshAttempts      int    = 3
	RefreshCooldownDuration        = 5 * time.Minute
)

type Client struct {
	httpClient      *http.Client
	config          *Config
	configPath      string
	token           string
	tokenMutex      sync.RWMutex
	refreshAttempts int64
	lastRefreshTime time.Time
}

type loginRequest struct {
	Device        string  `json:"device"`
	Email         string  `json:"email"`
	Password      string  `json:"password"`
	IsSocialLogin bool    `json:"isSocialLogin"`
	Provider      *string `json:"provider"`
	Token         *string `json:"token"`
}

type loginResponse struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
	Token      string `json:"token"`
	IsVerified bool   `json:"is_verified"`
}

var log = logrus.StandardLogger()

func New(config *Config) (*Client, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	log.Debugf("ekz New")
	c := &Client{
		httpClient: http.DefaultClient,
		config:     config,
	}
	c.httpClient.Transport = ekzRoundTripper{
		inner:  http.DefaultTransport,
		client: c,
	}
	return c, nil
}

func (c *Client) SetConfigPath(path string) {
	c.configPath = path
}

func (c *Client) Init() error {
	log.Debugf("initializing client")

	// Reset refresh attempts on initialization
	atomic.StoreInt64(&c.refreshAttempts, 0)

	// Check if token is valid
	log.Debugf("Checking if the token is valid")
	if c.config.Token != "" {
		log.Debugf("token is not empty, trying to use it")
		c.setToken(c.config.Token)
		if err := c.checkToken(); err != nil {
			log.Warnf("token is invalid: %s", err)
			c.setToken("")
			err = c.Login(c.config.Username, c.config.Password)
			if err != nil {
				return err
			}
		}
	} else {
		log.Debugf("token is empty, trying to login")
		err := c.Login(c.config.Username, c.config.Password)
		if err != nil {
			return err
		}
		log.Debugf("login OK")
	}
	return nil
}

func (c *Client) Login(username, password string) error {
	return c.loginWithClient(c.httpClient, username, password)
}

func (c *Client) loginWithClient(client *http.Client, username, password string) error {
	log.Debugf("logging in")
	req := loginRequest{
		Device:        "WEB",
		Email:         username,
		Password:      password,
		IsSocialLogin: false,
		Provider:      nil,
		Token:         nil,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequest(http.MethodPost, Backend+"/users/log-in", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "ekz-go")
	httpReq.Header.Set("Device", "WEB")

	res, err := client.Do(httpReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %s", res.Status)
	}

	var response loginResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return err
	}

	c.setToken(response.Token)
	return c.saveToken()
}

func (c *Client) GetUserChargingStations() ([]ChargingStation, error) {
	req, err := http.NewRequest(http.MethodPost, Backend+"/charging-stations/user-charging-stations", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	var chargingStationResponse Response[ChargingStationResult]
	if err := json.NewDecoder(res.Body).Decode(&chargingStationResponse); err != nil {
		return nil, err
	}

	return chargingStationResponse.Data.ChargingStations, nil
}

type remoteOp struct {
	ChargeBoxID string `json:"charge_box_id"`
	ConnectorID int    `json:"connector_id"`
}

func (c *Client) RemoteStart(boxId string, connectorID int) (*RemoteStartResult, error) {
	return c.remoteOp(boxId, connectorID, "start")
}

func (c *Client) RemoteStop(boxId string, connectorID int) (*RemoteStartResult, error) {
	return c.remoteOp(boxId, connectorID, "stop")
}

func (c *Client) remoteOp(boxId string, connectorID int, op string) (*RemoteStartResult, error) {
	log.Debugf("remoteOp %s on box %s connector %d", op, boxId, connectorID)
	remoteOpRequest := remoteOp{
		ChargeBoxID: boxId,
		ConnectorID: connectorID,
	}
	body, err := json.Marshal(remoteOpRequest)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, Backend+"/saascharge/remote-"+op, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("remote "+op+" failed: %s", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote "+op+" failed: %s", res.Status)
	}

	// Decode response
	var remoteOpResponse Response[RemoteStartResult]
	if err := json.NewDecoder(res.Body).Decode(&remoteOpResponse); err != nil {
		return nil, err
	}
	log.Debugf("remote op response: %+v", remoteOpResponse)
	return &remoteOpResponse.Data, nil
}

func (c *Client) checkToken() error {
	if c.getToken() == "" {
		return fmt.Errorf("token is empty")
	}

	_, err := c.GetProfile()
	if err != nil {
		return err
	}

	return nil
}

// saveToken saves the token to the config file
func (c *Client) saveToken() error {
	c.config.Token = c.getToken()
	if c.configPath == "" {
		return nil
	}
	return SaveConfig(c.config, c.configPath)
}

func (c *Client) GetConfig() Config {
	return *c.config
}

// getToken returns the current token in a thread-safe way
func (c *Client) getToken() string {
	c.tokenMutex.RLock()
	defer c.tokenMutex.RUnlock()
	return c.token
}

// setToken sets the token in a thread-safe way
func (c *Client) setToken(token string) {
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()
	c.token = token
}

// refreshTokenIfNeeded attempts to refresh the token if we get a 401
func (c *Client) refreshTokenIfNeeded() error {
	// Check if we've exceeded max refresh attempts
	attempts := atomic.LoadInt64(&c.refreshAttempts)
	if attempts >= int64(MaxRefreshAttempts) {
		return fmt.Errorf("authentication failed: exceeded maximum refresh attempts (%d). Please check your credentials", MaxRefreshAttempts)
	}

	// Check cooldown period to prevent rapid retries
	if time.Since(c.lastRefreshTime) < RefreshCooldownDuration && attempts > 0 {
		return fmt.Errorf("authentication failed: too many recent attempts. Please wait %v before retrying", RefreshCooldownDuration)
	}

	// Increment attempt counter
	atomic.AddInt64(&c.refreshAttempts, 1)
	c.lastRefreshTime = time.Now()

	log.Debugf("Refreshing token due to 401 response (attempt %d/%d)", atomic.LoadInt64(&c.refreshAttempts), MaxRefreshAttempts)

	// Create a new HTTP client without the roundtripper to avoid infinite recursion
	directClient := &http.Client{
		Transport: http.DefaultTransport,
		Timeout:   30 * time.Second,
	}

	err := c.loginWithClient(directClient, c.config.Username, c.config.Password)
	if err != nil {
		log.Errorf("Token refresh failed (attempt %d/%d): %v", atomic.LoadInt64(&c.refreshAttempts), MaxRefreshAttempts, err)
		return fmt.Errorf("failed to refresh token: %w", err)
	}

	// Reset attempts counter on successful login
	atomic.StoreInt64(&c.refreshAttempts, 0)
	log.Infof("Token refreshed successfully")
	return nil
}
