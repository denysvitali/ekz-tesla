package ekz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"
)

const (
	Backend string = "https://be.emob.ekz.ch"
)

type Client struct {
	httpClient *http.Client
	config     *Config
	configPath string
	token      string
	tokenMutex sync.RWMutex
}

type loginRequest struct {
	Device        string `json:"device"`
	Email         string `json:"email"`
	Password      string `json:"password"`
	IsSocialLogin bool   `json:"isSocialLogin"`
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
	log.Debugf("logging in")
	req := loginRequest{
		Device:        "WEB",
		Email:         username,
		Password:      password,
		IsSocialLogin: false,
	}
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}
	res, err := c.httpClient.Post(Backend+"/users/log-in", "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}

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
	log.Debugf("Refreshing token due to 401 response")
	err := c.Login(c.config.Username, c.config.Password)
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	return nil
}
