package api

import (
	"fmt"

	"image-api/api/middleware"
	"image-api/pkg/config"
	"image-api/pkg/log"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/natefinch/lumberjack.v2"
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
	if config.Config.Server.Env == "prod" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:           "https://dcf720463a4539ee9423eae86d733805@o4508963658268672.ingest.us.sentry.io/4509053306667008",
			EnableTracing: true,
			// Set TracesSampleRate to 1.0 to capture 100%
			// of transactions for tracing.
			// We recommend adjusting this value in production,
			TracesSampleRate: 0.1,
		})
		if err != nil {
			log.Error("init sentry", "failed", err)
		} else {
			r.Use(sentrygin.New(sentrygin.Options{
				Repanic:         true,
				WaitForDelivery: false,
			}))
			log.Info("init sentry", "success")
		}
	}
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
