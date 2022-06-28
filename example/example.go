package main

import (
	"time"

	"github.com/sirupsen/logrus"
	"github.com/willfantom/hclogrus"
)

var (
	checkID string = ""
)

func main() {
	hcHook, _ := hclogrus.New(checkID, time.Second, logrus.ErrorLevel)
	logrus.AddHook(hcHook)
	logrus.WithField("example", "healthchecks.io hook").Infoln("Hello, world!")
	time.Sleep(time.Second * 3)
	logrus.WithField("someError", "some data to attribute to the error").Errorln("leHow, olldr!")
	time.Sleep(time.Second * 3)
}
