package api

import (
	"image-api/api/controller/health"
	"image-api/api/controller/crawl"
	"image-api/api/middleware"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
)

func SetRouter(r *gin.Engine) {
	checkGroup := r.Group("/check")
	{
		checkGroup.GET("/status", health.CheckHealth)
	}

	// 爬虫任务相关接口（前端仅对接 Go 端）
	apiGroup := r.Group("/api")
	{
		apiGroup.POST("/tasks", crawl.SubmitTasks)
		apiGroup.GET("/tasks/status", crawl.GetStatus)

		apiGroup.POST("/tasks/result", crawl.PostTaskResult) // Python 回传任务结果

		apiGroup.GET("/results", crawl.ListResults)
		apiGroup.GET("/results/:id", crawl.GetResultDetail)
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
