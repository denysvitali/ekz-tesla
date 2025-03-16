package main

import "github.com/denysvitali/ekz-tesla/ekz"

func doListCmd(c *ekz.Client) {
	chargingStations, err := c.GetUserChargingStations()
	if err != nil {
		log.Fatalf("failed to get user charging stations: %v", err)
	}
	printChargingStations(chargingStations)
}
