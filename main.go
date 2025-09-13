package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/go-co-op/gocron/v2"
	geo "github.com/kellydunn/golang-geo"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/ekz-tesla/ekz"
	"github.com/denysvitali/ekz-tesla/teslamateapi"
)

type StartCmd struct {
}
type StopCmd struct {
}
type ListCmd struct{}
type LiveDataCmd struct{}

type AutoStartCmd struct {
	CarId           int    `arg:"--car-id,required"`
	TeslaMateApiUrl string `arg:"--teslamate-api-url,required"`
	MaximumCharge   int    `arg:"--maximum-charge" default:"90"`
}

type ScheduledAutostartCmd struct {
	CarId           int    `arg:"--car-id,required"`
	TeslaMateApiUrl string `arg:"--teslamate-api-url,required"`
	MaximumCharge   int    `arg:"--maximum-charge" default:"90"`
	CronjobLine     string `arg:"--cronjob-line,required"`
}

type SmartAutostartCmd struct {
	CarId             int      `arg:"--car-id,required"`
	TeslaMateApiUrl   string   `arg:"--teslamate-api-url,required"`
	MaximumCharge     int      `arg:"--maximum-charge" default:"90"`
	HighTariffTimes   []string `arg:"--high-tariff-times" help:"High tariff time ranges (format: 'HH:MM-HH:MM:Mon,Tue,Wed,Thu,Fri'). Default: 7:00-20:00:Mon,Tue,Wed,Thu,Fri"`
}

var args struct {
	ConfigFile         string                 `arg:"-c,--config,env:CONFIG_FILE" default:""`
	StartCmd           *StartCmd              `arg:"subcommand:start"`
	StopCmd            *StopCmd               `arg:"subcommand:stop"`
	ListCmd            *ListCmd               `arg:"subcommand:list"`
	LiveData           *LiveDataCmd           `arg:"subcommand:live-data"`
	AutoStart          *AutoStartCmd          `arg:"subcommand:autostart"`
	ScheduledAutostart *ScheduledAutostartCmd `arg:"subcommand:scheduled-autostart"`
	SmartAutostart     *SmartAutostartCmd     `arg:"subcommand:smart-autostart"`
	LogLevel           string                 `arg:"--log-level,env:LOG_LEVEL" default:"info"`
}

var log = logrus.StandardLogger()
var cfg *ekz.Config = nil

func main() {
	arg.MustParse(&args)
	setLogLevel()

	var err error
	cfg, err = ekz.GetConfigFromFile(args.ConfigFile)
	if err != nil {
		log.Fatalf("failed to get config: %v", err)
	}

	validateCfg()

	if args.AutoStart != nil {
		doAutostart(args.ConfigFile, args.AutoStart.TeslaMateApiUrl, args.AutoStart.CarId, args.AutoStart.MaximumCharge)
		return
	}

	if args.ScheduledAutostart != nil {
		doScheduledAutostart()
		return
	}

	if args.SmartAutostart != nil {
		doSmartAutostart()
		return
	}

	c, err := ekz.New(cfg)
	if err != nil {
		log.Fatalf("failed to create ekz client: %v", err)
	}
	c.SetConfigPath(args.ConfigFile)
	err = c.Init()
	if err != nil {
		log.Fatalf("failed to init: %v", err)
	}

	if args.StartCmd != nil {
		remoteStart, err := c.RemoteStart(cfg.ChargingStation.BoxId, cfg.ChargingStation.ConnectorId)
		if err != nil {
			log.Fatalf("failed to remote start: %v", err)
		}
		log.Debugf("remote start: %+v", remoteStart)
		return
	}

	if args.StopCmd != nil {
		remoteStop, err := c.RemoteStop(cfg.ChargingStation.BoxId, cfg.ChargingStation.ConnectorId)
		if err != nil {
			log.Fatalf("failed to remote stop: %v", err)
		}
		log.Debugf("remote stop: %+v", remoteStop)
		return
	}

	if args.ListCmd != nil {
		doListCmd(c)
		return
	}

	if args.LiveData != nil {
		doLiveData(c)
		return
	}
}

func doScheduledAutostart() {
	epochPlus1Year := time.Unix(0, 0).Add(365 * 24 * time.Hour)
	for {
		if time.Now().After(epochPlus1Year) {
			break
		}
		log.Debugf("waiting for time to be set")
		time.Sleep(1 * time.Second)
	}

	// Create scheduler
	s, err := gocron.NewScheduler()
	if err != nil {
		log.Fatalf("failed to create scheduler: %v", err)
	}
	defer func() { _ = s.Shutdown() }()
	_, err = s.NewJob(
		gocron.CronJob(args.ScheduledAutostart.CronjobLine, false),
		gocron.NewTask(scheduledStart, scheduledStartArgs{
			ConfigFile:      args.ConfigFile,
			CarId:           args.ScheduledAutostart.CarId,
			TeslaMateApiUrl: args.ScheduledAutostart.TeslaMateApiUrl,
			MaximumCharge:   args.ScheduledAutostart.MaximumCharge,
		}),
	)
	if err != nil {
		log.Fatalf("failed to create job: %v", err)
	}

	log.Infof("Starting scheduler")
	s.Start()

	// Block forever
	select {}
}

type scheduledStartArgs struct {
	ConfigFile      string
	CarId           int
	TeslaMateApiUrl string
	MaximumCharge   int
}

func scheduledStart(sArgs scheduledStartArgs) {
	doAutostart(sArgs.ConfigFile, sArgs.TeslaMateApiUrl, sArgs.CarId, sArgs.MaximumCharge)
}

func doSmartAutostart() {
	epochPlus1Year := time.Unix(0, 0).Add(365 * 24 * time.Hour)
	for {
		if time.Now().After(epochPlus1Year) {
			break
		}
		log.Debugf("waiting for time to be set")
		time.Sleep(1 * time.Second)
	}

	// Create EKZ client
	c, err := ekz.New(cfg)
	if err != nil {
		log.Fatalf("failed to create ekz client: %v", err)
	}
	c.SetConfigPath(args.ConfigFile)
	err = c.Init()
	if err != nil {
		log.Fatalf("failed to init: %v", err)
	}

	// Create autostart service
	autostartService, err := NewAutostartService(
		c,
		args.SmartAutostart.TeslaMateApiUrl,
		args.SmartAutostart.CarId,
		args.SmartAutostart.MaximumCharge,
		&cfg.ChargingStation,
	)
	if err != nil {
		log.Fatalf("failed to create autostart service: %v", err)
	}

	// Parse custom high tariff times if provided
	var highTariffTimes []ekz.TimeRange
	if len(args.SmartAutostart.HighTariffTimes) > 0 {
		for _, timeStr := range args.SmartAutostart.HighTariffTimes {
			tr, err := ekz.ParseTimeRangeString(timeStr)
			if err != nil {
				log.Fatalf("failed to parse high tariff time '%s': %v", timeStr, err)
			}
			highTariffTimes = append(highTariffTimes, tr)
		}
		log.Infof("Using custom high tariff schedule: %v", args.SmartAutostart.HighTariffTimes)
	} else {
		highTariffTimes = ekz.DefaultHighTariffSchedule()
		log.Infof("Using default high tariff schedule: Monday-Friday 07:00-20:00")
	}

	// Create schedule-based scheduler
	scheduler := ekz.NewScheduleScheduler(autostartService.TryAutostart, highTariffTimes)

	// Set up context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the scheduler
	err = scheduler.Start(ctx)
	if err != nil {
		log.Fatalf("failed to start scheduler: %v", err)
	}

	// Show next low tariff period
	nextLowTariff := scheduler.GetNextLowTariffPeriod()
	if time.Now().Equal(nextLowTariff) || time.Now().Before(nextLowTariff) {
		if nextLowTariff.Equal(time.Now().Truncate(time.Minute)) {
			log.Infof("Currently in low tariff period - charging attempts will begin")
		} else {
			log.Infof("Next low tariff period starts at: %s", nextLowTariff.Format("2006-01-02 15:04:05 Mon"))
		}
	}

	log.Infof("Smart autostart scheduler started. Will charge only during low tariff periods.")

	// Wait for shutdown signal
	<-sigChan
	log.Infof("Shutdown signal received, stopping scheduler...")

	// Stop the scheduler
	scheduler.Stop()
	log.Infof("Smart autostart scheduler stopped")
}

func validateCfg() {
	if cfg == nil {
		log.Fatalf("config is nil")
	}

	if cfg.ChargingStation.Latitude == 0 {
		log.Fatalf("config: charging_station.latitude is not set")
	}

	if cfg.ChargingStation.Longitude == 0 {
		log.Fatalf("config: charging_station.longitude is not set")
	}

	if cfg.ChargingStation.BoxId == "" {
		log.Fatalf("config: charging_station.box_id is not set")
	}

	if cfg.ChargingStation.ConnectorId == 0 {
		log.Fatalf("config: charging_station.connector_id is not set")
	}
}

func doAutostart(configFile string, teslamateApiUrl string, carId int, maximumCharge int) {
	log.Debugf("autostart: configFile=%s, carId=%d, maximumCharge=%d", configFile, carId, maximumCharge)
	carApi, err := teslamateapi.New(teslamateApiUrl)
	if err != nil {
		log.Fatalf("failed to create teslamate api client: %v", err)
	}

	status, err := carApi.GetCarStatus(carId)
	if err != nil {
		log.Fatalf("failed to get car status: %v", err)
	}

	if status.Status.State == "charging" {
		log.Infof("car is already charging, will not request to charge again")
		return
	}

	if !status.Status.ChargingDetails.PluggedIn {
		log.Warnf("car is not plugged in")
		return
	}

	if status.Status.BatteryDetails.BatteryLevel >= maximumCharge {
		log.Warnf("car is already charged to %d%%", status.Status.BatteryDetails.BatteryLevel)
		return
	}

	// Get the charging station location from the config and compare it to the car's location
	p1 := geo.NewPoint(cfg.ChargingStation.Latitude, cfg.ChargingStation.Longitude)
	p2 := geo.NewPoint(status.Status.CarGeodata.Latitude, status.Status.CarGeodata.Longitude)

	distanceKm := p1.GreatCircleDistance(p2)
	distanceMeters := distanceKm * 1000
	log.Debugf("distance: %f meters", distanceMeters)

	if distanceMeters > 100 {
		log.Warnf("car is not near the charging station")
		return
	}

	c, err := ekz.New(cfg)
	if err != nil {
		log.Fatalf("failed to create ekz client: %v", err)
	}
	err = c.Init()
	if err != nil {
		log.Fatalf("autostart: failed to init ekz client: %v", err)
	}

	err = c.StartCharge(cfg.ChargingStation.BoxId, cfg.ChargingStation.ConnectorId)
	if err != nil {
		log.Fatalf("failed to start charge: %v", err)
	}
}

func setLogLevel() {
	lvl, err := logrus.ParseLevel(args.LogLevel)
	if err != nil {
		log.Fatalf("invalid log level: %s", args.LogLevel)
	}
	log.SetLevel(lvl)
}
