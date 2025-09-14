package autostart

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-co-op/gocron/v2"
	geo "github.com/kellydunn/golang-geo"
	"github.com/spf13/cobra"

	"github.com/denysvitali/ekz-tesla/cmd/root"
	"github.com/denysvitali/ekz-tesla/ekz"
	"github.com/denysvitali/ekz-tesla/teslamateapi"
)

var (
	carID           int
	teslaMateAPIURL string
	maximumCharge   int
	cronSchedule    string
	highTariffTimes []string
)

var AutostartCmd = &cobra.Command{
	Use:   "autostart",
	Short: "Automatically start charging based on conditions",
	Long: `Autostart monitors your Tesla's location and battery status,
automatically starting charging when conditions are met.`,
}

var autostartOnceCmd = &cobra.Command{
	Use:   "once",
	Short: "Run autostart check once",
	Long:  `Check conditions and start charging if needed, then exit.`,
	RunE:  runAutostartOnce,
}

var autostartScheduledCmd = &cobra.Command{
	Use:   "scheduled",
	Short: "Run autostart on a cron schedule",
	Long:  `Run autostart checks according to a cron schedule.`,
	RunE:  runScheduledAutostart,
}

var autostartSmartCmd = &cobra.Command{
	Use:   "smart",
	Short: "Smart autostart based on electricity tariffs",
	Long: `Automatically start charging during low-tariff periods only.
Default low-tariff times are outside of Monday-Friday 07:00-20:00.`,
	RunE: runSmartAutostart,
}

func init() {
	// Common flags for all autostart commands
	AutostartCmd.PersistentFlags().IntVar(&carID, "car-id", 0, "TeslaMate car ID (required)")
	AutostartCmd.PersistentFlags().StringVar(&teslaMateAPIURL, "teslamate-api-url", "", "TeslaMate API URL (required)")
	AutostartCmd.PersistentFlags().IntVar(&maximumCharge, "maximum-charge", 90, "Maximum charge percentage")

	AutostartCmd.MarkPersistentFlagRequired("car-id")
	AutostartCmd.MarkPersistentFlagRequired("teslamate-api-url")

	// Scheduled-specific flags
	autostartScheduledCmd.Flags().StringVar(&cronSchedule, "cron", "*/5 * * * *", "Cron schedule (default: every 5 minutes)")

	// Smart autostart flags
	autostartSmartCmd.Flags().StringSliceVar(&highTariffTimes, "high-tariff-times", []string{},
		"High tariff time ranges (format: 'HH:MM-HH:MM:Mon,Tue,Wed,Thu,Fri')")

	// Add subcommands
	AutostartCmd.AddCommand(autostartOnceCmd)
	AutostartCmd.AddCommand(autostartScheduledCmd)
	AutostartCmd.AddCommand(autostartSmartCmd)

	root.RootCmd.AddCommand(AutostartCmd)
}

func runAutostartOnce(cmd *cobra.Command, args []string) error {
	service, err := createAutostartService()
	if err != nil {
		return err
	}

	return service.TryAutostart()
}

func runScheduledAutostart(cmd *cobra.Command, args []string) error {
	// Wait for time to be set (useful for embedded systems)
	waitForTimeSync()

	service, err := createAutostartService()
	if err != nil {
		return err
	}

	// Create scheduler
	s, err := gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}
	defer func() { _ = s.Shutdown() }()

	_, err = s.NewJob(
		gocron.CronJob(cronSchedule, false),
		gocron.NewTask(func() {
			if err := service.TryAutostart(); err != nil {
				root.GetLogger().Errorf("Autostart failed: %v", err)
			}
		}),
	)
	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	fmt.Printf("Starting scheduled autostart with cron: %s\n", cronSchedule)
	s.Start()

	// Block forever
	select {}
}

func runSmartAutostart(cmd *cobra.Command, args []string) error {
	// Wait for time to be set
	waitForTimeSync()

	service, err := createAutostartService()
	if err != nil {
		return err
	}

	// Parse custom high tariff times if provided
	var tariffSchedule []ekz.TimeRange
	if len(highTariffTimes) > 0 {
		for _, timeStr := range highTariffTimes {
			tr, err := ekz.ParseTimeRangeString(timeStr)
			if err != nil {
				return fmt.Errorf("failed to parse high tariff time '%s': %w", timeStr, err)
			}
			tariffSchedule = append(tariffSchedule, tr)
		}
		fmt.Printf("Using custom high tariff schedule: %v\n", highTariffTimes)
	} else {
		tariffSchedule = ekz.DefaultHighTariffSchedule()
		fmt.Println("Using default high tariff schedule: Monday-Friday 07:00-20:00")
	}

	// Create schedule-based scheduler
	scheduler := ekz.NewScheduleScheduler(service.TryAutostart, tariffSchedule)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the scheduler
	if err := scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}

	// Show next low tariff period
	nextLowTariff := scheduler.GetNextLowTariffPeriod()
	if time.Now().Equal(nextLowTariff) || time.Now().Before(nextLowTariff) {
		if nextLowTariff.Equal(time.Now().Truncate(time.Minute)) {
			fmt.Println("Currently in low tariff period - charging attempts will begin")
		} else {
			fmt.Printf("Next low tariff period starts at: %s\n", nextLowTariff.Format("2006-01-02 15:04:05 Mon"))
		}
	}

	fmt.Println("Smart autostart scheduler started. Will charge only during low tariff periods.")
	fmt.Println("Press Ctrl+C to stop")

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\nShutting down scheduler...")

	// Stop the scheduler
	scheduler.Stop()
	fmt.Println("Smart autostart scheduler stopped")

	return nil
}

func createAutostartService() (*AutostartService, error) {
	cfg := root.GetConfig()
	if cfg == nil {
		return nil, fmt.Errorf("configuration not loaded")
	}

	// Validate charging station config
	if err := root.ValidateChargingStationConfig(); err != nil {
		return nil, fmt.Errorf("invalid charging station config: %w", err)
	}

	// Initialize EKZ client if not already done
	client := root.GetClient()
	if client == nil {
		var err error
		client, err = ekz.New(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create EKZ client: %w", err)
		}

		configPath := root.GetConfigPath()
		if configPath != "" {
			client.SetConfigPath(configPath)
		}

		if err := client.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize EKZ client: %w", err)
		}
	}

	return NewAutostartService(client, teslaMateAPIURL, carID, maximumCharge, &cfg.ChargingStation)
}

func waitForTimeSync() {
	epochPlus1Year := time.Unix(0, 0).Add(365 * 24 * time.Hour)
	for time.Now().Before(epochPlus1Year) {
		root.GetLogger().Debug("Waiting for time to be set...")
		time.Sleep(1 * time.Second)
	}
}

// AutostartService handles the logic for automatically starting charging
type AutostartService struct {
	ekzClient       *ekz.Client
	carAPI          *teslamateapi.Client
	carID           int
	maxCharge       int
	chargingStation *ekz.ChargingStationConfig
}

// NewAutostartService creates a new autostart service
func NewAutostartService(ekzClient *ekz.Client, teslaMateAPIURL string, carID int, maxCharge int, chargingStation *ekz.ChargingStationConfig) (*AutostartService, error) {
	// Normalize the URL
	teslaMateAPIURL = strings.TrimSuffix(teslaMateAPIURL, "/")

	carAPI, err := teslamateapi.New(teslaMateAPIURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create TeslaMate API client: %w", err)
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
	log := root.GetLogger()
	log.Debugf("Checking autostart conditions for car %d (max charge: %d%%)", as.carID, as.maxCharge)

	status, err := as.carAPI.GetCarStatus(as.carID)
	if err != nil {
		return fmt.Errorf("failed to get car status: %w", err)
	}

	// Check if already charging
	if status.Status.State == "charging" {
		log.Info("Car is already charging")
		return nil
	}

	// Check if plugged in
	if !status.Status.ChargingDetails.PluggedIn {
		log.Warn("Car is not plugged in")
		return nil
	}

	// Check battery level
	if status.Status.BatteryDetails.BatteryLevel >= as.maxCharge {
		log.Infof("Car battery at %d%% (max: %d%%)", status.Status.BatteryDetails.BatteryLevel, as.maxCharge)
		return nil
	}

	// Check distance from charging station
	p1 := geo.NewPoint(as.chargingStation.Latitude, as.chargingStation.Longitude)
	p2 := geo.NewPoint(status.Status.CarGeodata.Latitude, status.Status.CarGeodata.Longitude)

	distanceKm := p1.GreatCircleDistance(p2)
	distanceMeters := distanceKm * 1000
	log.Debugf("Distance from charging station: %.1f meters", distanceMeters)

	if distanceMeters > 100 {
		log.Warn("Car is not near the charging station")
		return nil
	}

	// All conditions met, start charging
	log.Info("All conditions met, starting charge...")
	if err := as.ekzClient.StartCharge(as.chargingStation.BoxId, as.chargingStation.ConnectorId); err != nil {
		return fmt.Errorf("failed to start charge: %w", err)
	}

	log.Info("✅ Successfully started charging")
	return nil
}