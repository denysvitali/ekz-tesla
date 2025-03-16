package ekz_test

import (
	"net/http"
	"testing"

	"github.com/h2non/gock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"github.com/denysvitali/ekz-tesla/ekz"
)

func TestClient_GetProfile(t *testing.T) {
	gock.New(ekz.Backend).
		Post("/users/log-in").
		JSON(
			map[string]any{
				"device":        "WEB",
				"email":         "user@example.com",
				"password":      "password",
				"isSocialLogin": false,
			},
		).
		Reply(http.StatusOK).
		File("../resources/login-successful.json")

	gock.New(ekz.Backend).
		Get("/profile").Reply(200).
		File("../resources/profile.json")

	log := logrus.StandardLogger()
	log.SetLevel(logrus.DebugLevel)
	cfg := ekz.Config{
		Username: "user@example.com",
		Password: "password",
	}
	c, err := ekz.New(&cfg)
	require.NoError(t, err)
	err = c.Login(cfg.Username, cfg.Password)
	if err != nil {
		t.Fatal(err)
	}

	profile, err := c.GetProfile()
	if err != nil {
		t.Fatal(err)
	}
	log.Debugf("profile: %+v", profile)
}
