package ekz_test

import (
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/denysvitali/ekz-tesla/ekz"
)

func TestClient_GetProfile(t *testing.T) {
	log := logrus.StandardLogger()
	log.SetLevel(logrus.DebugLevel)
	c := ekz.New()
	cfg, err := ekz.GetConfigFromFile()
	if err != nil {
		t.Fatal(err)
	}
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
