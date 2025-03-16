package teslamateapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type Client struct {
	endpoint string
}

func New(addr string) (*Client, error) {
	_, err := url.Parse(addr)
	if err != nil {
		return nil, err
	}

	c := Client{
		endpoint: addr,
	}

	return &c, nil
}

func (c *Client) GetCarStatus(carId int) (*CarStatusResponse, error) {
	res, err := http.Get(c.endpoint + "/api/v1/cars/" + fmt.Sprintf("%d", carId) + "/status")
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", res.Status)
	}

	// Decode JSON
	var response genericResponse[CarStatusResponse]
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	return &response.Data, nil
}
