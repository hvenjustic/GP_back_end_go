package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

var Config config

type config struct {
	Server struct {
		DefaultPort   string `yaml:"defaultPort"`
		Env           string `yaml:"env"`
		Source        string `yaml:"source"`
		InferencePath string `yaml:"inferencePath"`
	} `yaml:"server"`

	Crawl4AI struct {
		BaseURL         string `yaml:"baseUrl"`
		TimeoutSeconds  int    `yaml:"timeoutSeconds"`
		DefaultMaxDepth int    `yaml:"defaultMaxDepth"`
		DefaultMaxPages int    `yaml:"defaultMaxPages"`
	} `yaml:"crawl4ai"`

	Mysql struct {
		DBAddress      []string `yaml:"dbMysqlAddress"`
		DBUserName     string   `yaml:"dbMysqlUserName"`
		DBPassword     string   `yaml:"dbMysqlPassword"`
		DBDatabaseName string   `yaml:"dbMysqlDatabaseName"`
		DBCharSet      string   `yaml:"dbCharset"`
		DBMaxOpenConns int      `yaml:"dbMaxOpenConns"`
		DBMaxIdleConns int      `yaml:"dbMaxIdleConns"`
		DBMaxLifeTime  int      `yaml:"dbMaxLifeTime"`
		DBMaxIdleTime  int      `yaml:"dbMaxIdleTime"`
		LogLevel       int      `yaml:"logLevel"`
		SlowThreshold  int      `yaml:"slowThreshold"`
	} `yaml:"mysql"`

	Redis struct {
		DBAddress     []string `yaml:"dbAddress"`
		DBIdleTimeout int      `yaml:"dbIdleTimeout"`
		DBUserName    string   `yaml:"dbUserName"`
		DBPassWord    string   `yaml:"dbPassWord"`
		DBIndex       int      `yaml:"dbIndex"`
		DBPoolSize    int      `yaml:"dbPoolSize"`
		EnableCluster bool     `yaml:"enableCluster"`
	} `yaml:"redis"`

	LogSetting struct {
		LogLevel     uint32 `yaml:"logLevel"`
		LogName      string `yaml:"logName"`
		RotationNum  int    `yaml:"rotationNum"`
		CompressSize int    `yaml:"compressSize"`
	} `yaml:"logSetting"`

	Prometheus struct {
		Enable bool `yaml:"enable"`
	} `yaml:"prometheus"`
}

func InitConfig(configPath string) {
	cfgName := filepath.Join("", configPath)
	bytes, err := ioutil.ReadFile(cfgName)
	if err != nil {
		panic("open config file err: " + err.Error())
	}
	if err = yaml.Unmarshal(bytes, &Config); err != nil {
		panic(err.Error())
	}
	fmt.Printf("config: %+v\n", Config)
}
