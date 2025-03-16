package ekz

import "time"

type Tariff struct {
	TariffStatus string      `json:"tariff_status"`
	TariffPrice  interface{} `json:"tariff_price"`
}

type RemoteStartResult struct {
	StartTime      time.Time   `json:"start_time"`
	ChargingStatus string      `json:"charging_status"`
	CurrentTariff  Tariff      `json:"current_tariff"`
	CurrentPower   interface{} `json:"current_power"`
}
