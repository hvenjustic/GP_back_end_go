package db

import (
	"fmt"
	"time"

	"image-api/pkg/config"
	"image-api/pkg/log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Writer struct{}

func (w Writer) Printf(format string, args ...interface{}) {
	log.Warn("gorm", fmt.Sprintf(format, args...))
}

type mysqlDB struct {
	db *gorm.DB
}

func initMysqlDB() {
	initMysql()
	return
}

func initMysql() {
	log.Info("initMysql", "begin")
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=%s&parseTime=true&loc=Local",
		config.Config.Mysql.DBUserName, config.Config.Mysql.DBPassword, config.Config.Mysql.DBAddress[0],
		config.Config.Mysql.DBDatabaseName, config.Config.Mysql.DBCharSet)
	var db *gorm.DB
	var err1 error
	newLogger := logger.New(
		Writer{},
		logger.Config{
			SlowThreshold:             time.Duration(config.Config.Mysql.SlowThreshold) * time.Millisecond, // Slow SQL threshold
			LogLevel:                  logger.LogLevel(config.Config.Mysql.LogLevel),                       // Log level
			IgnoreRecordNotFoundError: false,                                                               // Ignore ErrRecordNotFound error for logger
			Colorful:                  false,                                                               // Disable color
		},
	)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Error("initMysql", "open failed", err.Error(), dsn)
	}
	if err != nil {
		time.Sleep(time.Duration(1) * time.Second)
		db, err1 = gorm.Open(mysql.Open(dsn), nil)
		if err1 != nil {
			log.Error("initMysql", "twice open failed", err1.Error(), dsn)
			panic(err1.Error())
		}
	}

	//Check the database and table during initialization
	sql := "show tables;"
	log.Info("initMysql", "exec sql:", sql, "begin")
	err = db.Exec(sql).Error
	if err != nil {
		log.Error("initMysql", "exec sql failed", err.Error(), sql)
		panic(err.Error())
	}
	log.Info("initMysql", "exec sql: ", sql, "end")

	sqlDB, err := db.DB()
	if err != nil {
		log.Error("initMysql", "get sql db failed", err.Error())
		panic(err.Error())
	}

	sqlDB.SetConnMaxLifetime(time.Second * time.Duration(config.Config.Mysql.DBMaxLifeTime))
	sqlDB.SetConnMaxIdleTime(time.Second * time.Duration(config.Config.Mysql.DBMaxIdleTime))
	sqlDB.SetMaxOpenConns(config.Config.Mysql.DBMaxOpenConns)
	sqlDB.SetMaxIdleConns(config.Config.Mysql.DBMaxIdleConns)

	fmt.Println("initMysql open mysql ok ", dsn)

	DB.MysqlDB.db = db
}

func (m *mysqlDB) DB() *gorm.DB {
	return DB.MysqlDB.db
}
