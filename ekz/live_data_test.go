package ekz

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestClient_GetLiveData(t *testing.T) {
	log.SetLevel(logrus.DebugLevel)
	token := "foo"
	gock.New(Backend).
		Post("/charging-stations/charging-live-data").
		MatchHeader("Authorization", "Token "+token).
		Reply(http.StatusOK).
		File("../resources/live-data.json")

	c := New("")
	c.token = token
	liveData, err := c.GetLiveData("1234", 1, ConnectorStatusCharging)
	assert.Nil(t, err)
	assert.NotNil(t, liveData)
	log.Debugf("live data: %+v", liveData)
}

func TestClient_GetLiveData_Failure(t *testing.T) {
	log.SetLevel(logrus.DebugLevel)
	token := "foo"
	gock.New(Backend).
		Post("/charging-stations/charging-live-data").
		MatchHeader("Authorization", "Token "+token).
		Reply(http.StatusNotFound).
		File("../resources/live-data-fail.json")

	c := New("")
	c.token = token
	liveData, err := c.GetLiveData("1234", 1, ConnectorStatusCharging)
	assert.NotNil(t, err)
	assert.Nil(t, liveData)
	log.Debugf("error=%v", err)
}
