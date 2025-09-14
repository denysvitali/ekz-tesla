package main

import (
	"github.com/sirupsen/logrus"
	"github.com/denysvitali/ekz-tesla/ekz"
)

var listLog = logrus.StandardLogger()

func doListCmd(c *ekz.Client) {
	chargingStations, err := c.GetUserChargingStations()
	if err != nil {
		listLog.Fatalf("failed to get user charging stations: %v", err)
	}
	printChargingStations(chargingStations)
}
