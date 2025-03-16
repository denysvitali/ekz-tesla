package main

import (
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

var args struct {
	ConfigFile         string                 `arg:"-c,--config,env:CONFIG_FILE" default:""`
	StartCmd           *StartCmd              `arg:"subcommand:start"`
	StopCmd            *StopCmd               `arg:"subcommand:stop"`
	ListCmd            *ListCmd               `arg:"subcommand:list"`
	LiveData           *LiveDataCmd           `arg:"subcommand:live-data"`
	AutoStart          *AutoStartCmd          `arg:"subcommand:autostart"`
	ScheduledAutostart *ScheduledAutostartCmd `arg:"subcommand:scheduled-autostart"`
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
