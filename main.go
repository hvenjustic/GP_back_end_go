package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"image-api/api"
	"image-api/pkg/config"
	"image-api/pkg/db"
	"image-api/pkg/log"
	"image-api/pkg/worker"
)

var configFile = flag.String("f", "config/config-local.yaml", "the config file")

func main() {
	flag.Parse()
	if configFile == nil || len(*configFile) == 0 {
		panic("config file path invalid")
	}
	config.InitConfig(*configFile)
	log.NewPrivateLog(config.Config.LogSetting.LogName, config.Config.LogSetting.RotationNum,
		config.Config.LogSetting.CompressSize, config.Config.LogSetting.LogLevel)
	log.Info("main", "begin init db")
	db.InitDb()

	// Crawl4AI 异步爬取调度器：从 Redis 队列取任务、最多并发3个、每10秒轮询进度。
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go worker.NewCrawlScheduler().Start(ctx)

	api.Run()
}
