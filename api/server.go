package api

import (
	"fmt"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/natefinch/lumberjack.v2"
	"image-api/api/middleware"
	"image-api/pkg/config"
	"image-api/pkg/log"
)

func Run() {
	gin.SetMode(gin.ReleaseMode)
	logFile := &lumberjack.Logger{
		Filename:   "logs/server.log",
		MaxSize:    200, // MB
		MaxBackups: 10,
		MaxAge:     3, // days
		Compress:   true,
		LocalTime:  true,
	}
	gin.DefaultWriter = logFile
	gin.DefaultErrorWriter = logFile
	r := gin.Default()
	r.Use(gin.Recovery())
	r.Use(cors.Default())
	r.Use(middleware.TimeCostLog())

	if config.Config.Prometheus.Enable {
		r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	SetRouter(r)
	address := fmt.Sprintf("0.0.0.0:%s", config.Config.Server.DefaultPort)

	fmt.Println("start api server, address:", address)
	err := r.Run(address)
	if err != nil {
		log.Error("main", "api run failed ", address, err.Error())
		panic("api start failed " + err.Error())
	}
}
