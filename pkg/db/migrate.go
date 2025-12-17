package db

import (
	"image-api/models/mysql"
	"image-api/pkg/log"
)

func migrateMysqlSchemas() {
	if DB.MysqlDB.db == nil {
		return
	}
	if err := DB.MysqlDB.db.AutoMigrate(&mysql.CrawlTarget{}); err != nil {
		log.Error("migrateMysqlSchemas", "auto migrate failed", err.Error())
		panic(err.Error())
	}
}

