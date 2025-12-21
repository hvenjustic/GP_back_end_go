package db

import (
	"context"
	"time"

	"GP_back_end_go/pkg/config"
	"GP_back_end_go/pkg/log"
	"GP_back_end_go/pkg/utils"

	go_redis "github.com/go-redis/redis/v8"
)

func initRedis() {
	log.Info("initRedis", "begin")
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if config.Config.Redis.EnableCluster {
		DB.RDB = go_redis.NewClusterClient(&go_redis.ClusterOptions{
			Addrs:    config.Config.Redis.DBAddress,
			Username: config.Config.Redis.DBUserName,
			Password: config.Config.Redis.DBPassWord, // no password set
			PoolSize: 50,
		})
		_, err = DB.RDB.Ping(ctx).Result()
		if err != nil {
			log.Error("InitRedisCluster", "ping failed", err.Error(), "config", utils.StructToJsonString(config.Config.Redis))
			panic(err.Error())
		}
	} else {
		DB.RDB = go_redis.NewClient(&go_redis.Options{
			Addr:        config.Config.Redis.DBAddress[0],
			Username:    config.Config.Redis.DBUserName,
			Password:    config.Config.Redis.DBPassWord, // no password set
			DB:          config.Config.Redis.DBIndex,
			IdleTimeout: time.Duration(config.Config.Redis.DBIdleTimeout) * time.Second,
			PoolSize:    config.Config.Redis.DBPoolSize, // 连接池大小
		})
		_, err = DB.RDB.Ping(ctx).Result()
		if err != nil {
			log.Error("InitRedisClient", "ping failed", err.Error(), "config", utils.StructToJsonString(config.Config.Redis))
			panic(err.Error())
		}
		log.Info("initRedis", "init redis client suc")
	}
}
