package main

import (
	"os"

	"github.com/sirupsen/logrus"

	_ "github.com/denysvitali/ekz-tesla/cmd/autostart"
	_ "github.com/denysvitali/ekz-tesla/cmd/list"
	_ "github.com/denysvitali/ekz-tesla/cmd/livedata"
	"github.com/denysvitali/ekz-tesla/cmd/root"
	_ "github.com/denysvitali/ekz-tesla/cmd/start"
	_ "github.com/denysvitali/ekz-tesla/cmd/stop"
)

func main() {
	if err := root.RootCmd.Execute(); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}
}