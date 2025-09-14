package main

import (
	geo "github.com/kellydunn/golang-geo"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/ekz-tesla/ekz"
	"github.com/denysvitali/ekz-tesla/teslamateapi"
)

var log = logrus.StandardLogger()

// AutostartService handles the logic for automatically starting charging
type AutostartService struct {
	ekzClient    *ekz.Client
	carAPI       *teslamateapi.Client
	carID        int
	maxCharge    int
	chargingStation *ekz.ChargingStationConfig
}

// NewAutostartService creates a new autostart service
func NewAutostartService(ekzClient *ekz.Client, teslaMateAPIURL string, carID int, maxCharge int, chargingStation *ekz.ChargingStationConfig) (*AutostartService, error) {
	carAPI, err := teslamateapi.New(teslaMateAPIURL)
	if err != nil {
		return nil, err
	}

	return &AutostartService{
		ekzClient:       ekzClient,
		carAPI:          carAPI,
		carID:           carID,
		maxCharge:       maxCharge,
		chargingStation: chargingStation,
	}, nil
}

// TryAutostart attempts to start charging if conditions are met
func (as *AutostartService) TryAutostart() error {
	log.Debugf("autostart: carId=%d, maxCharge=%d", as.carID, as.maxCharge)

	status, err := as.carAPI.GetCarStatus(as.carID)
	if err != nil {
		return err
	}

	if status.Status.State == "charging" {
		log.Infof("car is already charging, will not request to charge again")
		return nil
	}

	if !status.Status.ChargingDetails.PluggedIn {
		log.Warnf("car is not plugged in")
		return nil
	}

	if status.Status.BatteryDetails.BatteryLevel >= as.maxCharge {
		log.Warnf("car is already charged to %d%%", status.Status.BatteryDetails.BatteryLevel)
		return nil
	}

	// Get the charging station location from the config and compare it to the car's location
	p1 := geo.NewPoint(as.chargingStation.Latitude, as.chargingStation.Longitude)
	p2 := geo.NewPoint(status.Status.CarGeodata.Latitude, status.Status.CarGeodata.Longitude)

	distanceKm := p1.GreatCircleDistance(p2)
	distanceMeters := distanceKm * 1000
	log.Debugf("distance: %f meters", distanceMeters)

	if distanceMeters > 100 {
		log.Warnf("car is not near the charging station")
		return nil
	}

	err = as.ekzClient.StartCharge(as.chargingStation.BoxId, as.chargingStation.ConnectorId)
	if err != nil {
		return err
	}

	log.Infof("Successfully started charging")
	return nil
}