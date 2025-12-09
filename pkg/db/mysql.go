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
	db     *gorm.DB
	readDb *gorm.DB
}

func initMysqlDB() {
	initMysql()
	initReadMysql()
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

func initReadMysql() {
	log.Info("initReadMysql", "begin")

	// 如果没有配置只读库地址，则使用主库作为只读库
	if len(config.Config.Mysql.ReadDBAddress) == 0 {
		log.Info("initReadMysql", "no read db address, use master db instead")
		DB.MysqlDB.readDb = DB.MysqlDB.db
		return
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=%s&parseTime=true&loc=Local",
		config.Config.Mysql.ReadDBUserName, config.Config.Mysql.ReadDBPassword, config.Config.Mysql.ReadDBAddress[0],
		config.Config.Mysql.ReadDBDatabaseName, config.Config.Mysql.ReadDBCharSet)
	var db *gorm.DB
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
		log.Error("initReadMysql", "open failed", err.Error(), dsn)
		log.Warn("initReadMysql", "fallback to master db due to connection error")
		DB.MysqlDB.readDb = DB.MysqlDB.db
		return
	}

	//Check the database and table during initialization
	sql := "show tables;"
	log.Info("initReadMysql", "exec sql:", sql, "begin")
	err = db.Exec(sql).Error
	if err != nil {
		log.Error("initReadMysql", "exec sql failed", err.Error(), sql)
		log.Warn("initReadMysql", "fallback to master db due to sql execution error")
		DB.MysqlDB.readDb = DB.MysqlDB.db
		return
	}
	log.Info("initReadMysql", "exec sql: ", sql, "end")

	sqlDB, err := db.DB()
	if err != nil {
		log.Error("initReadMysql", "get sql db failed", err.Error())
		log.Warn("initReadMysql", "fallback to master db due to sql.DB error")
		DB.MysqlDB.readDb = DB.MysqlDB.db
		return
	}

	// 使用读库的连接池配置，如果没有则使用主库配置
	maxLifeTime := config.Config.Mysql.ReadDBMaxLifeTime
	if maxLifeTime == 0 {
		maxLifeTime = config.Config.Mysql.DBMaxLifeTime
	}
	maxIdleTime := config.Config.Mysql.ReadDBMaxIdleTime
	if maxIdleTime == 0 {
		maxIdleTime = config.Config.Mysql.DBMaxIdleTime
	}
	maxOpenConns := config.Config.Mysql.ReadDBMaxOpenConns
	if maxOpenConns == 0 {
		maxOpenConns = config.Config.Mysql.DBMaxOpenConns
	}
	maxIdleConns := config.Config.Mysql.ReadDBMaxIdleConns
	if maxIdleConns == 0 {
		maxIdleConns = config.Config.Mysql.DBMaxIdleConns
	}

	sqlDB.SetConnMaxLifetime(time.Second * time.Duration(maxLifeTime))
	sqlDB.SetConnMaxIdleTime(time.Second * time.Duration(maxIdleTime))
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)

	fmt.Println("initReadMysql open read mysql ok ", dsn)

	DB.MysqlDB.readDb = db
}

func (m *mysqlDB) DB() *gorm.DB {
	return DB.MysqlDB.db
}

func (m *mysqlDB) ReadDB() *gorm.DB {
	return DB.MysqlDB.readDb
}
