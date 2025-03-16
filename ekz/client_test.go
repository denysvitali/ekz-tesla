package ekz

import (
	"os"
	"testing"

	"github.com/h2non/gock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestClient_Login(t *testing.T) {
	gock.New(Backend).Post("/users/log-in").Reply(200).File("../resources/login-successful.json")

	c := New(&Config{})
	err := c.Login("user", "pass")
	assert.Nil(t, err)
}

func TestClient_GetUserChargingStations(t *testing.T) {
	token := "foo"

	gock.New(Backend).
		Post("/charging-stations/user-charging-stations").
		MatchHeader("Authorization", "Token "+token).
		Reply(200).
		File("../resources/user-charging-stations.json")

	c := New(&Config{})
	c.token = token
	chargingStations, err := c.GetUserChargingStations()
	assert.Nil(t, err)
	assert.Equal(t, 1, len(chargingStations))
	assert.Equal(t, "1234", chargingStations[0].ChargeBoxes[0].ChargeBoxID)
}

func TestClient_RemoteStart_Mock(t *testing.T) {
	log.SetLevel(logrus.DebugLevel)
	token := "foo"
	gock.New(Backend).
		Post("/saascharge/remote-start").
		MatchHeader("Authorization", "Token "+token).
		BodyString(`{"charge_box_id":"00000000","connector_id":1}`).
		Reply(200).
		File("../resources/remote-start.json")

	c := New(&Config{})
	c.token = token
	remoteStart, err := c.RemoteStart("00000000", 1)
	log.Debugf("remote start: %+v", remoteStart)
	assert.Nil(t, err)
}

func TestClient_RemoteStop_Mock(t *testing.T) {
	log.SetLevel(logrus.DebugLevel)
	token := "foo"
	gock.New(Backend).
		Post("/saascharge/remote-stop").
		MatchHeader("Authorization", "Token "+token).
		BodyString(`{"charge_box_id":"00000000","connector_id":1}`).
		Reply(200).
		File("../resources/remote-start.json")

	c := New(&Config{})
	c.token = token
	remoteStop, err := c.RemoteStop("00000000", 1)
	assert.Nil(t, err)
	assert.NotNil(t, remoteStop)
	log.Debugf("remote stop: %+v", remoteStop)
}

// TestClient_StartCharge calls the real API and starts charging the vehicle
func TestClient_StartCharge(t *testing.T) {
	chargeBoxId := os.Getenv("CHARGE_BOX_ID")
	if (chargeBoxId == "") {
		t.Skip("CHARGE_BOX_ID is not set")
	}
	log.SetLevel(logrus.DebugLevel)
	cfg, err := GetConfigFromFile("")
	if err != nil {
		t.Fatalf("failed to get config: %v", err)
	}
	c := New(cfg)
	err = c.Login(cfg.Username, cfg.Password)
	if err != nil {
		t.Fatalf("failed to login: %v", err)
	}

	err = c.StartCharge(chargeBoxId, 1)
	if err != nil {
		t.Fatalf("failed to remote start: %v", err)
	}
}
