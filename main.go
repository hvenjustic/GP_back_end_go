package main

import (
	"flag"
	"image-api/api"
	"image-api/pkg/config"
	"image-api/pkg/db"
	"image-api/pkg/log"
)

var configFile = flag.String("f", "config/config-test.yaml", "the config file")

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
	api.Run()
}
