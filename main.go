package main

import (
	"GoAutoBash/server"
	"github.com/sirupsen/logrus"
	"io"
	"os"
)

func init() {
	// init logrus
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	file, err := os.OpenFile("server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logrus.Fatal(err)
	}
	mw := io.MultiWriter(os.Stdout, file)
	logrus.SetOutput(mw)
}

func main() {
	if err := server.Listen("0.0.0.0:8080"); err != nil { logrus.Fatal(err) }
}
