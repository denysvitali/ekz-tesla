package ekz

import (
	"os"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

type ChargingStationConfig struct {
	Latitude    float64 `yaml:"latitude"`
	Longitude   float64 `yaml:"longitude"`
	BoxId       string  `yaml:"box_id"`
	ConnectorId int     `yaml:"connector_id"`
}

type Config struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Token    string `yaml:"token"`

	ChargingStation ChargingStationConfig `yaml:"charging_station"`
}

var defaultConfigFilePath = xdg.ConfigHome + "/ekz-tesla/config.yaml"

func GetConfigFromFile(inputConfigFile string) (*Config, error) {
	if inputConfigFile == "" {
		inputConfigFile = defaultConfigFilePath
	}
	f, err := os.Open(inputConfigFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	err = yaml.NewDecoder(f).Decode(&cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func SaveConfig(cfg *Config, configFile string) error {
	f, err := os.OpenFile(configFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return yaml.NewEncoder(f).Encode(cfg)
}
