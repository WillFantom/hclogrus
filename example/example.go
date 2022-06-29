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
	hcHook, err := hclogrus.New(checkID, time.Hour, logrus.ErrorLevel)
	if err != nil {
		panic(err)
	}
	logrus.AddHook(hcHook)
	logrus.WithField("example", "healthchecks.io hook").Infoln("Hello, world!")
	time.Sleep(time.Second * 3)
	logrus.WithField(hclogrus.JobStartField, true).Infoln("Job Starting")
	time.Sleep(time.Millisecond * 1500)
	logrus.WithField("someError", "some data to attribute to the error").Errorln("leHow, olldr!")
	time.Sleep(time.Second * 3)
}
