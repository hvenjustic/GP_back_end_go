package db

import (
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
	//mysql init
	initMysqlDB()
	migrateMysqlSchemas()
	initRedis()
	initCache()
	log.Info("InitDb", "db init end")
}
