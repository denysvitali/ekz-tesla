package ekz

import (
	"errors"
	"fmt"
	"time"
)

// StartCharge calls the remote start API of the backend and starts fetching some live data
func (c *Client) StartCharge(chargeBoxID string, connectorID int) error {
	// Check if we're already charging
	livedata, err := c.GetLiveData(chargeBoxID, connectorID, ConnectorStatusCharging)
	if err != nil {
		if !errors.Is(err, ErrTransactionNotFoundInTable) {
			return err
		}
	}

	if livedata != nil {
		printLiveData(livedata)
		return nil
	}

	remoteStart, err := c.RemoteStart(chargeBoxID, connectorID)
	if err != nil {
		return err
	}

	log.Debugf("remote start: %+v", remoteStart)

	// We call live data until the ChargedEnergy is > 0
	attempts := 0
	// One attempt every 10 seconds, so we wait 5 minutes
	maxAttempts := 6 * 5
	for {
		if attempts >= maxAttempts {
			return fmt.Errorf("max attempts reached")
		}
		livedata, err := c.GetLiveData(chargeBoxID, connectorID, ConnectorStatusCharging)
		if err != nil {
			if errors.Is(err, ErrTransactionNotFoundInTable) {
				attempts++
				time.Sleep(5 * time.Second)
				continue
			}
			return err
		}

		printLiveData(livedata)

		if livedata.Power > 0 {
			break
		}
		attempts++
		log.Debugf("Power is %.2f, waiting 5 seconds", livedata.Power)
		time.Sleep(10 * time.Second)
	}

	return nil
}

func printLiveData(livedata *LiveDataResponse) {
	fmt.Printf("Status: %s\nPower: %.2f\nChargedEnergy: %.2f\n",
		livedata.Status,
		livedata.Power,
		livedata.ChargedEnergy,
	)
	log.Debugf("live data: %+v", livedata)
}
