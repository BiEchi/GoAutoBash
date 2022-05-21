package server

import (
	"GoAutoBash/queue"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/webhooks.v5/github"
)

var hook = new(github.Webhook)

/* naive help function in case the user gives wrong requests */
func noRouteHandler(c *gin.Context) {
	c.JSON(404, gin.H{
		"code":    "PAGE_NOT_FOUND",
		"message": "No page found. Please specify an operation like '/ping' after the website",
	})
}

/* naive function for testing whether the server is running and opening its port 8080 */
func pingHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}

/* when the user want to get a snapshot of the status through the Internet*/
func statusHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"message":     "ok",
		"waiting_num": len(queue.Queue),
		"status":      queue.GetStatus(),
	})
}

/* when an event is triggered, this callback function will be triggered */
func webhookHandler(c *gin.Context) {
	// append all the webhook types you want to the hook.Parse argument.
	payload, err := hook.Parse(c.Request, github.PushEvent, github.WatchEvent)
	if err != nil {
		_ = c.AbortWithError(400, err)
	}
	// append all the operations for each event below.
	switch payload.(type) {
	case github.PushPayload:
		/* if the server is the pusher, return without enqueuing */
		if payload.(github.PushPayload).HeadCommit.Message == "Report Generated." {
			return
		}
		push := payload.(github.PushPayload)
		if err := queue.TaskEnqueue(&push); err != nil {
			_ = c.AbortWithError(500, err)
		}
		logrus.Info("Someone has pushed to your repo!")
		c.JSON(200, gin.H{
			"message": "OK",
		})
	case github.WatchPayload:
		logrus.Info("Someone has starred your repo!")
		c.JSON(200, gin.H{
			"message": "OK",
		})
	default:
		return
	}
}
