package api

import (
	"GP_back_end_go/api/controller/crawl"
	"GP_back_end_go/api/controller/health"
	"GP_back_end_go/api/middleware"

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
		apiGroup.POST("/queues/clear", crawl.ClearQueue)
		apiGroup.GET("/results", crawl.ListResults)
		apiGroup.GET("/results/:id", crawl.GetResultDetail)
		apiGroup.POST("/results/preprocess", crawl.PreprocessResult)
		apiGroup.GET("/results/preprocess/status", crawl.GetPreprocessStatus)
		apiGroup.POST("/results/graph", crawl.BuildGraph)
		apiGroup.GET("/results/graph/status", crawl.GetGraphStatus)
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
