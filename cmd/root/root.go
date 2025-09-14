package root

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/denysvitali/ekz-tesla/ekz"
)

var (
	cfgFile  string
	logLevel string
	cfg      *ekz.Config
	client   *ekz.Client
	log      = logrus.StandardLogger()
)

var RootCmd = &cobra.Command{
	Use:   "ekz-tesla",
	Short: "EKZ Tesla CLI - Manage your EKZ charging stations",
	Long: `EKZ Tesla CLI is a command line tool to interact with EKZ charging stations.
You can list available stations, start/stop charging, view live data, and set up automated charging.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Set log level
		if err := setLogLevel(); err != nil {
			return err
		}

		// Skip initialization for commands that don't need the client
		if cmd.Name() == "version" || cmd.Name() == "help" {
			return nil
		}

		// Load configuration
		if err := initConfig(); err != nil {
			return fmt.Errorf("failed to initialize config: %w", err)
		}

		// Initialize EKZ client for commands that need it
		needsClient := []string{"list", "start", "stop", "live-data"}
		for _, cmdName := range needsClient {
			if cmd.Name() == cmdName || cmd.Parent().Name() == cmdName {
				if err := initClient(); err != nil {
					return fmt.Errorf("unable to initialize client: %w", err)
				}
				break
			}
		}

		return nil
	},
}

func init() {
	RootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is $XDG_CONFIG_HOME/ekz-tesla/config.yaml)")
	RootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")

	// Bind flags to viper
	viper.BindPFlag("config", RootCmd.PersistentFlags().Lookup("config"))
	viper.BindPFlag("log-level", RootCmd.PersistentFlags().Lookup("log-level"))

	// Set environment variable prefix
	viper.SetEnvPrefix("EKZ")
	viper.AutomaticEnv()
}

func initConfig() error {
	// Determine config file path
	configPath := ""

	if cfgFile != "" {
		// Use config file from the flag
		configPath = cfgFile
		viper.SetConfigFile(cfgFile)
	} else {
		// Use XDG config directory with adrg/xdg
		configPath = filepath.Join(xdg.ConfigHome, "ekz-tesla", "config.yaml")

		// Add config paths for viper
		viper.AddConfigPath(filepath.Join(xdg.ConfigHome, "ekz-tesla"))
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Read in environment variables
	viper.AutomaticEnv()

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found; use defaults or environment variables
		log.Debug("No config file found, using defaults and environment variables")
	} else {
		log.Debugf("Using config file: %s", viper.ConfigFileUsed())
		configPath = viper.ConfigFileUsed()
	}

	// Try to load config from file
	var err error
	cfg, err = ekz.GetConfigFromFile(configPath)
	if err != nil {
		// If file doesn't exist, create an empty config
		if os.IsNotExist(err) {
			log.Debug("Config file not found, creating empty config")
			cfg = &ekz.Config{}
		} else {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	// Override with viper values if set (including environment variables)
	if viper.IsSet("username") {
		cfg.Username = viper.GetString("username")
	}
	if viper.IsSet("password") {
		cfg.Password = viper.GetString("password")
	}
	if viper.IsSet("token") {
		cfg.Token = viper.GetString("token")
	}
	if viper.IsSet("charging_station.box_id") {
		cfg.ChargingStation.BoxId = viper.GetString("charging_station.box_id")
	}
	if viper.IsSet("charging_station.connector_id") {
		cfg.ChargingStation.ConnectorId = viper.GetInt("charging_station.connector_id")
	}
	if viper.IsSet("charging_station.latitude") {
		cfg.ChargingStation.Latitude = viper.GetFloat64("charging_station.latitude")
	}
	if viper.IsSet("charging_station.longitude") {
		cfg.ChargingStation.Longitude = viper.GetFloat64("charging_station.longitude")
	}

	return nil
}

func initClient() error {
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}

	var err error
	client, err = ekz.New(cfg)
	if err != nil {
		return fmt.Errorf("failed to create EKZ client: %w", err)
	}

	if cfgFile != "" {
		client.SetConfigPath(cfgFile)
	} else if viper.ConfigFileUsed() != "" {
		client.SetConfigPath(viper.ConfigFileUsed())
	}

	// Initialize client - authentication is required
	if err := client.Init(); err != nil {
		return err
	}

	return nil
}

func setLogLevel() error {
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %s", logLevel)
	}
	log.SetLevel(lvl)
	return nil
}

func Execute() error {
	return RootCmd.Execute()
}

func GetClient() *ekz.Client {
	return client
}

func GetConfig() *ekz.Config {
	return cfg
}

func GetLogger() *logrus.Logger {
	return log
}

func ValidateChargingStationConfig() error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if cfg.ChargingStation.Latitude == 0 {
		return fmt.Errorf("charging_station.latitude is not set")
	}

	if cfg.ChargingStation.Longitude == 0 {
		return fmt.Errorf("charging_station.longitude is not set")
	}

	if cfg.ChargingStation.BoxId == "" {
		return fmt.Errorf("charging_station.box_id is not set")
	}

	if cfg.ChargingStation.ConnectorId == 0 {
		return fmt.Errorf("charging_station.connector_id is not set")
	}

	return nil
}

// Helper to get config file path
func GetConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	if viper.ConfigFileUsed() != "" {
		return viper.ConfigFileUsed()
	}
	// Use XDG config directory with adrg/xdg
	return filepath.Join(xdg.ConfigHome, "ekz-tesla", "config.yaml")
}
