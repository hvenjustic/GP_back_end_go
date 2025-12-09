package middleware

import (
	"time"

	"image-api/pkg/log"

	"github.com/gin-gonic/gin"
)

func TimeCostLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		cost := time.Since(start).Milliseconds()
		url := c.Request.URL.String()
		log.Info(url, "cost", cost, "ms")
	}
}
