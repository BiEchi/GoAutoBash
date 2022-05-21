package main

import (
	"GoAutoBash/queue"
	"GoAutoBash/server"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"time"
)

func init() {
	// init logrus
	logrus.SetReportCaller(true)
	logrus.SetFormatter(&logrus.JSONFormatter{})
	file, err := os.OpenFile("./logs/server.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		logrus.Fatal(err)
	}
	mw := io.MultiWriter(os.Stdout, file)
	logrus.SetOutput(mw)
}

func main() {
	if err := queue.StartQueue(2, 400, time.Minute*10); err != nil { logrus.Fatal(err) }
	/* the server listens at 0.0.0.0:8080 */
	if err := server.Listen("0.0.0.0:8080"); err != nil { logrus.Fatal(err) }
}
