package main

import (
	"context"
	"flag"
	"os/signal"
	"syscall"

	"GP_back_end_go/api"
	"GP_back_end_go/internal/service"
	"GP_back_end_go/pkg/config"
	"GP_back_end_go/pkg/db"
	"GP_back_end_go/pkg/log"
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
	go service.NewCrawlScheduler().Start(ctx)
	go service.StartPreprocessWorkers(ctx)
	go service.StartGraphWorkers(ctx)

	api.Run()
}
