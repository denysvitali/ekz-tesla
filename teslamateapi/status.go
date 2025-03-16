package teslamateapi

import (
	"time"
)

type Units struct {
	UnitOfLength      string `json:"unit_of_length"`
	UnitOfTemperature string `json:"unit_of_temperature"`
}

type TpmDetails struct {
	TpmsPressureFl float64 `json:"tpms_pressure_fl"`
	TpmsPressureFr float64 `json:"tpms_pressure_fr"`
	TpmsPressureRl float64 `json:"tpms_pressure_rl"`
	TpmsPressureRr float64 `json:"tpms_pressure_rr"`
}

type DrivingDetails struct {
	Elevation  int    `json:"elevation"`
	Heading    int    `json:"heading"`
	Power      int    `json:"power"`
	ShiftState string `json:"shift_state"`
	Speed      int    `json:"speed"`
}

type ClimateDetails struct {
	InsideTemp        float64 `json:"inside_temp"`
	IsClimateOn       bool    `json:"is_climate_on"`
	IsPreconditioning bool    `json:"is_preconditioning"`
	OutsideTemp       float64 `json:"outside_temp"`
}

type ChargingDetails struct {
	ChargeCurrentRequest       float32   `json:"charge_current_request"`
	ChargeCurrentRequestMax    float32   `json:"charge_current_request_max"`
	ChargeEnergyAdded          float32   `json:"charge_energy_added"`
	ChargeLimitSoc             float32   `json:"charge_limit_soc"`
	ChargePortDoorOpen         bool      `json:"charge_port_door_open"`
	ChargerActualCurrent       float32   `json:"charger_actual_current"`
	ChargerPhases              int       `json:"charger_phases"`
	ChargerPower               float32   `json:"charger_power"`
	ChargerVoltage             float32   `json:"charger_voltage"`
	PluggedIn                  bool      `json:"plugged_in"`
	ScheduledChargingStartTime time.Time `json:"scheduled_charging_start_time"`
	TimeToFullCharge           float32   `json:"time_to_full_charge"`
}

type PhysicalStatus struct {
	DoorsOpen     bool `json:"doors_open"`
	FrunkOpen     bool `json:"frunk_open"`
	Healthy       bool `json:"healthy"`
	IsUserPresent bool `json:"is_user_present"`
	Locked        bool `json:"locked"`
	SentryMode    bool `json:"sentry_mode"`
	TrunkOpen     bool `json:"trunk_open"`
	WindowsOpen   bool `json:"windows_open"`
}

type GeoData struct {
	Geofence  string  `json:"geofence"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Exterior struct {
	ExteriorColor string `json:"exterior_color"`
	SpoilerType   string `json:"spoiler_type"`
	WheelType     string `json:"wheel_type"`
}

type Details struct {
	Model       string `json:"model"`
	TrimBadging string `json:"trim_badging"`
}

type BatteryDetails struct {
	BatteryLevel       int     `json:"battery_level"`
	EstBatteryRange    float64 `json:"est_battery_range"`
	IdealBatteryRange  float64 `json:"ideal_battery_range"`
	RatedBatteryRange  float64 `json:"rated_battery_range"`
	UsableBatteryLevel int     `json:"usable_battery_level"`
}

type Car struct {
	CarID   int    `json:"car_id"`
	CarName string `json:"car_name"`
}

type Versions struct {
	UpdateAvailable bool   `json:"update_available"`
	UpdateVersion   string `json:"update_version"`
	Version         string `json:"version"`
}

type CarStatus struct {
	BatteryDetails  BatteryDetails  `json:"battery_details"`
	CarDetails      Details         `json:"car_details"`
	CarExterior     Exterior        `json:"car_exterior"`
	CarGeodata      GeoData         `json:"car_geodata"`
	CarStatus       PhysicalStatus  `json:"car_status"`
	CarVersions     Versions        `json:"car_versions"`
	ChargingDetails ChargingDetails `json:"charging_details"`
	ClimateDetails  ClimateDetails  `json:"climate_details"`
	DisplayName     string          `json:"display_name"`
	DrivingDetails  DrivingDetails  `json:"driving_details"`
	Odometer        float64         `json:"odometer"`
	State           string          `json:"state"`
	StateSince      time.Time       `json:"state_since"`
	TpmsDetails     TpmDetails      `json:"tpms_details"`
}

type CarStatusResponse struct {
	Car    Car       `json:"car"`
	Status CarStatus `json:"status"`
	Units  Units     `json:"units"`
}

type genericResponse[T any] struct {
	Data T `json:"data"`
}
