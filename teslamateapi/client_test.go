package teslamateapi_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/denysvitali/ekz-tesla/teslamateapi"
)

func TestClient_GetCarStatus(t *testing.T) {
	teslamateApiUrl := os.Getenv("TESLAMATE_API_URL")
	if teslamateApiUrl == "" {
		t.Skip("TESLAMATE_API_URL not set")
	}

	expectedCarName := os.Getenv("TESLAMATE_API_CAR_NAME")
	if expectedCarName == "" {
		t.Skip("TESLAMATE_API_CAR_NAME not set")
	}

	c, err := teslamateapi.New(teslamateApiUrl)
	assert.Nil(t, err)
	status, err := c.GetCarStatus(1)
	assert.Nil(t, err)
	assert.Equal(t, expectedCarName, status.Car.CarName)
}
