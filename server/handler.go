package server

import (
	"github.com/gin-gonic/gin"
)

func noRouteHandler(c *gin.Context) {
	c.JSON(404, gin.H{
		"code":    "PAGE_NOT_FOUND",
		"message": "No page found. Please specify an operation like /ping",
	})
}

func pingHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

func webhookHandler(c *gin.Context) {
	message := c.PostForm("message")
	nick := c.DefaultPostForm("nick", "anonymous")
	c.JSON(200, gin.H{
		"status":  "posted",
		"message": message,
		"nick":    nick,
	})
}
