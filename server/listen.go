package server

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

func Listen(addr string) error {
	router := gin.Default()
	router.LoadHTMLGlob("templates/*")
	router.NoRoute(noRouteHandler)
	router.GET("/ping", pingHandler)
	router.GET("/status", statusHandler)

	/* handler for the GitHub WebHook event */
	router.POST("/webhook", webhookHandler)
	logrus.Info("Starting server at ", addr)
	return router.Run(addr)
}
