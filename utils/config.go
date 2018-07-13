package utils

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"log"
	"os"
	"strings"
)

const (
	defaultLogPath = "./conf.yaml"
)

type Config struct {
	Server struct {
		Name             string   `yaml:"Name"`
		ListenHost       string   `yaml:"ListenHost"`
		ListenPort       int      `yaml:"ListenPort"`
		ListenDomainName []string `yaml:"ListenDomainName"`

		Concurrency        int  `yaml:"Concurrency"`
		DisabledKeepAlive  bool `yaml:"DisabledKeepAlive"`
		ReadBufferSize     int  `yaml:"ReadBufferSize"`
		WriteBufferSize    int  `yaml:"WriteBufferSize"`
		MaxRequestBodySize int  `yaml:"MaxRequestBodySize"`
		ReduceMemoryUsage  bool `yaml:"ReduceMemoryUsage"`
	} `yaml:"Server"`

	Client struct {
		Name                string `yaml:"Name"`
		MaxConnsPerHost     int    `yaml:"MaxConnsPerHost"`
		MaxIdleConnDuration int    `yaml:"MaxIdleConnDuration"`
	} `yaml:"Client"`

	Etcd struct {
		Name      string   `yaml:"Name"`
		Endpoints []string `yaml:"Endpoints"`
		Username  string   `yaml:"Username"`
		Password  string   `yaml:"Password"`

		AutoSyncInterval     int `yaml:"AutoSyncInterval"`
		DialTimeout          int `yaml:"DialTimeout"`
		DialKeepAliveTime    int `yaml:"DialKeepAliveTime"`
		DialKeepAliveTimeout int `yaml:"DialKeepAliveTimeout"`
	} `yaml:"Etcd"`
}

var (
	Conf *Config
)

func ReadConfig(path ...string) *Config {
	var f *os.File
	var err error
	var config Config
	var isProductionEnv bool
	var configFilePath string

	// env IS_PRODUCTION
	if os.Getenv("IS_PRODUCTION") == "1" {
		isProductionEnv = true
	} else {
		isProductionEnv = false
	}

	// env CONFIG_PATH
	if len(path) == 1 {
		configFilePath = path[0]
	} else if len(path) == 0 {
		configFilePath = os.Getenv("CONFIG_PATH")
		if configFilePath == "" {
			configFilePath = defaultLogPath
		}
	} else {
		log.Fatal("only one path could be passed in")
	}

	if !isProductionEnv {
		fp := strings.Split(configFilePath, ".")
		fpLen := len(fp)
		if fpLen > 1 {
			fp[fpLen-2] += "_test"
			configFilePath = strings.Join(fp, ".")
		} else if fpLen == 1 {
			configFilePath += "_test"
		} else {
			log.Fatal(fmt.Sprintf("lack of config file path, %s", configFilePath))
		}
	}
	f, err = os.OpenFile(configFilePath, os.O_RDONLY, 0666)
	defer f.Close()
	if err != nil {
		log.Fatal(err)
	}

	length, _ := f.Seek(0, 2)
	conf := make([]byte, length)
	f.Seek(0, 0)
	_, err = f.Read(conf)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(conf, &config)
	if err != nil {
		log.Fatal(err)
	}
	return &config
}

func init() {
	Conf = ReadConfig()
}
