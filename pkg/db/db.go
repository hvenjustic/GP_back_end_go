package db

import (
	"image-api/pkg/config"
	"image-api/pkg/log"

	go_redis "github.com/go-redis/redis/v8"
	"github.com/patrickmn/go-cache"
)

var DB DataBases

type DataBases struct {
	MysqlDB mysqlDB
	RDB     go_redis.UniversalClient
	Cache   *cache.Cache
}

func key(dbAddress, dbName string) string {
	return dbAddress + "_" + dbName
}

func InitDb() {
	// 如果是本地测试环境，跳过数据库初始化
	if config.Config.Server.Env == "dev" {
		log.Info("InitDb", "skip database init for temp test environment")
		return
	}

	//mysql init
	initMysqlDB()
	initRedis()
	initCache()
	log.Info("InitDb", "db init end")
}
