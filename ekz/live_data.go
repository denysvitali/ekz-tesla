package ekz

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type LiveDataRequest struct {
	ChargeBoxId     string          `json:"charge_box_id"`
	ConnectorId     int             `json:"connector_id"`
	ConnectorStatus ConnectorStatus `json:"connector_status"`
}

type CurrentTariff struct {
	TariffPrice  float64 `json:"tariff_price"`
	TariffStatus string  `json:"tariff_status"`
}

type LiveDataResponse struct {
	ChargeBoxID       string        `json:"chargeBoxId"`
	ChargedEnergy     float64       `json:"charged_energy"`
	ConnectorID       string        `json:"connectorId"`
	CurrentTariff     CurrentTariff `json:"current_tariff"`
	Duration          any           `json:"duration"`
	Hightariff        any           `json:"hightariff"`
	HightariffPv      any           `json:"hightariff_pv"`
	Hightariffcost    any           `json:"hightariffcost"`
	HightariffcostPv  any           `json:"hightariffcost_pv"`
	Hightariffusage   any           `json:"hightariffusage"`
	HightariffusagePv any           `json:"hightariffusage_pv"`
	ID                int           `json:"id"`
	IDTag             string        `json:"idTag"`
	Lowtariff         any           `json:"lowtariff"`
	LowtariffPv       any           `json:"lowtariff_pv"`
	Lowtariffcost     any           `json:"lowtariffcost"`
	LowtariffcostPv   any           `json:"lowtariffcost_pv"`
	Lowtariffusage    any           `json:"lowtariffusage"`
	LowtariffusagePv  any           `json:"lowtariffusage_pv"`
	Metervaluestart   int           `json:"metervaluestart"`
	Metervaluestop    any           `json:"metervaluestop"`
	Power             float64       `json:"power"`
	Starttimestamp    int           `json:"starttimestamp"`
	StationName       any           `json:"stationName"`
	StationStreet     any           `json:"stationStreet"`
	Status            string        `json:"status"`
	Stoptimestamp     any           `json:"stoptimestamp"`
	Totalcost         any           `json:"totalcost"`
	Totalusage        any           `json:"totalusage"`
	TransactionID     int           `json:"transaction_id"`
}

func (c *Client) GetLiveData(chargeBoxId string, connectorId int, connectorStatus ConnectorStatus) (*LiveDataResponse, error) {
	jsonBody, err := toJson(
		LiveDataRequest{
			ChargeBoxId:     chargeBoxId,
			ConnectorId:     connectorId,
			ConnectorStatus: connectorStatus,
		})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, Backend+"/charging-stations/charging-live-data", jsonBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusNotFound {
		// Parse error message
		return nil, getError(res)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", res.Status)
	}

	var response LiveDataResponse
	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func getError(res *http.Response) error {
	var errorResponse ErrorResponse
	err := json.NewDecoder(res.Body).Decode(&errorResponse)
	if err != nil {
		return err
	}

	if errorResponse.Message == "transaction not found in table" {
		return ErrTransactionNotFoundInTable
	}
	return fmt.Errorf("%s", errorResponse.Message)
}

func toJson[T any](request T) (io.Reader, error) {
	buffer := bytes.NewBuffer(nil)
	err := json.NewEncoder(buffer).Encode(request)
	return buffer, err
}
