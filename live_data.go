package main

import (
	"time"

	"github.com/denysvitali/ekz-tesla/ekz"

	"github.com/sirupsen/logrus"
)

var liveDataLog = logrus.StandardLogger()

func doLiveData(c *ekz.Client) {
	cfg := c.GetConfig()

	for {
		liveData, err := c.GetLiveData(cfg.ChargingStation.BoxId, cfg.ChargingStation.ConnectorId, "")
		if err != nil {
			liveDataLog.Fatalf("failed to get live data: %v", err)
		}
		printLiveData(liveData)
		time.Sleep(5 * time.Second)
	}
}
