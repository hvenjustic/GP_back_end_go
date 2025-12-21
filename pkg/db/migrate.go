package db

import (
	"GP_back_end_go/models/mysql"
	"GP_back_end_go/pkg/log"
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
