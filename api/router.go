package api

import (
	"image-api/api/controller/health"
	"image-api/api/middleware"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func SetRouter(r *gin.Engine) {
	checkGroup := r.Group("/check")
	{
		checkGroup.GET("/status", health.CheckHealth)
	}

	// 需要认证的接口组 - 现在V1接口具有V2的所有特征
	authGroup := r.Group("/v1")
	authGroup.Use(middleware.AuthMiddleware())
	{

	}

	debugGroup := r.Group("/monitor", func(c *gin.Context) {
		c.Next()
	})
	pprof.RouteRegister(debugGroup, "pprof")
}
