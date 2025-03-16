package ekz

type TariffData struct {
	Prices struct {
		Current string  `json:"current"`
		High    float64 `json:"high"`
		Low     float64 `json:"low"`
	} `json:"prices"`
}

type Connector struct {
	ChargingProcessStatus string     `json:"chargingProcessStatus"`
	ConnectorID           int        `json:"connectorId"`
	ConnectorName         string     `json:"connectorName"`
	ConnectorStatus       string     `json:"connectorStatus"`
	HasPermission         bool       `json:"hasPermission"`
	PlugType              string     `json:"plugType"`
	Status                string     `json:"status"`
	TariffData            TariffData `json:"tariff_data"`
}

type ChargeBox struct {
	ChargeBoxID           string      `json:"chargeBoxId"`
	ChargeBoxName         string      `json:"chargeBoxName"`
	ChargingProcessStatus string      `json:"chargingProcessStatus"`
	City                  string      `json:"city"`
	ConnectorStatus       string      `json:"connectorStatus"`
	ConnectorCount        int         `json:"connector_count"`
	Connectors            []Connector `json:"connectors"`
	Country               string      `json:"country"`
	GpsLat                float64     `json:"gpsLat"`
	GpsLng                float64     `json:"gpsLng"`
	Online                bool        `json:"online"`
	PlugType              string      `json:"plugType"`
	Street                string      `json:"street"`
	TariffSchedule        string      `json:"tariff_schedule"`
	Zip                   string      `json:"zip"`
}

type ChargingStation struct {
	ChargeBoxes []ChargeBox `json:"chargeBoxes"`
	Invoiced    bool        `json:"invoiced"`
}
type ChargingStationResult struct {
	ChargingStations   []ChargingStation `json:"charging_stations"`
	Quantity           int               `json:"quantity"`
	UserContractExists bool              `json:"user_contract_exists"`
}

type Response[T any] struct {
	Data       T      `json:"data"`
	Message    string `json:"message"`
	StatusCode int    `json:"status_code"`
}
